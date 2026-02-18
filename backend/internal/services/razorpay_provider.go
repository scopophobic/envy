package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"

	razorpay "github.com/razorpay/razorpay-go"
)

var razorpayPlanIDs = map[string]string{
	"starter": "",
	"team":    "",
}

// RazorpayProvider implements PaymentProvider using Razorpay Subscriptions.
type RazorpayProvider struct {
	client        *razorpay.Client
	webhookSecret string
}

// NewRazorpayProvider initialises the Razorpay client.
func NewRazorpayProvider(keyID, keySecret, webhookSecret, planStarter, planTeam string) *RazorpayProvider {
	client := razorpay.NewClient(keyID, keySecret)
	if planStarter != "" {
		razorpayPlanIDs["starter"] = planStarter
	}
	if planTeam != "" {
		razorpayPlanIDs["team"] = planTeam
	}
	return &RazorpayProvider{client: client, webhookSecret: webhookSecret}
}

func (r *RazorpayProvider) CreateCheckoutSession(customerEmail, userID, plan, successURL, _ string) (string, error) {
	planID, ok := razorpayPlanIDs[plan]
	if !ok || planID == "" {
		return "", fmt.Errorf("no Razorpay plan configured for %q — create one in the Razorpay Dashboard first", plan)
	}

	data := map[string]interface{}{
		"plan_id":     planID,
		"total_count": 120,
		"quantity":    1,
		"notes": map[string]interface{}{
			"user_id": userID,
			"plan":    plan,
			"email":   customerEmail,
		},
		"notify_info": map[string]interface{}{
			"notify_email": customerEmail,
		},
	}

	body, err := r.client.Subscription.Create(data, nil)
	if err != nil {
		return "", fmt.Errorf("razorpay: create subscription: %w", err)
	}

	shortURL, _ := body["short_url"].(string)
	if shortURL == "" {
		return "", fmt.Errorf("razorpay: no short_url in subscription response")
	}

	return shortURL, nil
}

// CreatePortalSession — Razorpay has no Stripe-like customer portal.
// We return a link to our own settings page with a flag; the frontend
// shows a "Cancel subscription" button that calls our cancel endpoint.
func (r *RazorpayProvider) CreatePortalSession(_ string, returnURL string) (string, error) {
	return returnURL + "?manage=true", nil
}

func (r *RazorpayProvider) VerifyWebhookPayload(payload []byte, sigHeader string) (WebhookEvent, error) {
	mac := hmac.New(sha256.New, []byte(r.webhookSecret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(sigHeader)) {
		return WebhookEvent{}, fmt.Errorf("razorpay: invalid webhook signature")
	}

	var raw struct {
		Event   string          `json:"event"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return WebhookEvent{}, fmt.Errorf("razorpay: unmarshal: %w", err)
	}

	we := WebhookEvent{}

	// Razorpay nests everything under payload.subscription.entity
	var envelope struct {
		Subscription struct {
			Entity struct {
				ID         string `json:"id"`
				CustomerID string `json:"customer_id"`
				PlanID     string `json:"plan_id"`
				Status     string `json:"status"`
				Notes      struct {
					UserID string `json:"user_id"`
					Plan   string `json:"plan"`
					Email  string `json:"email"`
				} `json:"notes"`
			} `json:"entity"`
		} `json:"subscription"`
	}
	_ = json.Unmarshal(raw.Payload, &envelope)

	ent := envelope.Subscription.Entity
	we.CustomerID = ent.CustomerID
	we.SubscriptionID = ent.ID
	we.UserID = ent.Notes.UserID
	we.Plan = ent.Notes.Plan
	we.CustomerEmail = ent.Notes.Email

	// Map Razorpay event names → normalised constants
	switch raw.Event {
	case "subscription.activated", "subscription.charged":
		we.Type = EventSubscriptionActivated
		we.Status = "active"
	case "subscription.updated":
		we.Type = EventSubscriptionUpdated
		we.Status = ent.Status
	case "subscription.cancelled", "subscription.completed", "subscription.expired":
		we.Type = EventSubscriptionCancelled
		we.Status = "cancelled"
	default:
		log.Printf("[razorpay] unhandled webhook event: %s", raw.Event)
		we.Type = raw.Event
	}

	return we, nil
}
