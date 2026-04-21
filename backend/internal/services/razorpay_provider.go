package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode"

	razorpay "github.com/razorpay/razorpay-go"
	rzperrors "github.com/razorpay/razorpay-go/errors"
)

var razorpayPlanIDs = map[string]string{
	"starter": "",
	"team":    "",
}

// RazorpayPlanIDsConfigured reports whether dashboard plan IDs were set for subscription checkout.
func RazorpayPlanIDsConfigured() (starterOK, teamOK bool) {
	return strings.TrimSpace(razorpayPlanIDs["starter"]) != "",
		strings.TrimSpace(razorpayPlanIDs["team"]) != ""
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

const (
	minOrderAmountPaise = 100
	maxOrderAmountPaise = 10_000_000 // ₹1,00,000.00
)

// CreateOrder creates a Razorpay order for Standard Web Checkout (one-time payment).
// amount is in the smallest currency unit (e.g. paise for INR). receipt must be ≤40 chars; pass empty to auto-generate.
func (r *RazorpayProvider) CreateOrder(amount int64, currency, receipt string) (orderID string, respAmount int64, respCurrency string, err error) {
	if amount < minOrderAmountPaise {
		return "", 0, "", fmt.Errorf("amount must be at least %d (smallest currency unit)", minOrderAmountPaise)
	}
	if amount > maxOrderAmountPaise {
		return "", 0, "", fmt.Errorf("amount exceeds maximum allowed")
	}
	if currency == "" {
		currency = "INR"
	}
	if len(currency) != 3 {
		return "", 0, "", fmt.Errorf("currency must be a 3-letter ISO code")
	}
	currency = strings.ToUpper(currency)

	receipt = strings.TrimSpace(receipt)
	if receipt == "" {
		receipt = fmt.Sprintf("rcpt%d", time.Now().Unix())
	}
	receipt = sanitizeReceipt(receipt)
	if len(receipt) > 40 {
		receipt = receipt[:40]
	}

	body, err := r.client.Order.Create(map[string]interface{}{
		"amount":   amount,
		"currency": currency,
		"receipt":  receipt,
	}, nil)
	if err != nil {
		return "", 0, "", err
	}

	id, _ := body["id"].(string)
	if id == "" {
		return "", 0, "", fmt.Errorf("razorpay: no order id in response")
	}
	amt, ok := parseRZPAmount(body["amount"])
	if !ok {
		return "", 0, "", fmt.Errorf("razorpay: invalid amount in response")
	}
	cur, _ := body["currency"].(string)
	if cur == "" {
		cur = currency
	}
	return id, amt, strings.ToUpper(cur), nil
}

// VerifyStandardPaymentSignature checks the payment signature from the checkout success callback.
func (r *RazorpayProvider) VerifyStandardPaymentSignature(orderID, paymentID, signature string) error {
	if orderID == "" || paymentID == "" || signature == "" {
		return fmt.Errorf("order_id, payment_id, and signature are required")
	}
	secret := r.client.Auth.Secret
	payload := orderID + "|" + paymentID
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return fmt.Errorf("signature mismatch")
	}
	return nil
}

// ErrRazorpayUnauthorized indicates invalid or missing Razorpay API credentials.
type ErrRazorpayUnauthorized struct {
	Message string
}

func (e *ErrRazorpayUnauthorized) Error() string { return e.Message }

// MapRazorpayOrderError maps Razorpay SDK / API errors to handler-friendly errors.
func MapRazorpayOrderError(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "unauthorized") || strings.Contains(msg, "authentication") {
		return &ErrRazorpayUnauthorized{Message: err.Error()}
	}
	var bad *rzperrors.BadRequestError
	if errors.As(err, &bad) && strings.Contains(strings.ToLower(bad.Error()), "unauthorized") {
		return &ErrRazorpayUnauthorized{Message: bad.Error()}
	}
	return err
}

func parseRZPAmount(v interface{}) (int64, bool) {
	switch x := v.(type) {
	case float64:
		return int64(x), true
	case int:
		return int64(x), true
	case int64:
		return x, true
	case json.Number:
		i, e := x.Int64()
		return i, e == nil
	default:
		return 0, false
	}
}

func sanitizeReceipt(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return fmt.Sprintf("rcpt%d", time.Now().Unix())
	}
	return out
}
