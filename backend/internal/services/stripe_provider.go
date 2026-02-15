package services

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/stripe/stripe-go/v82"
	billingSession "github.com/stripe/stripe-go/v82/billingportal/session"
	checkoutSession "github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/webhook"
)

// Stripe price IDs — set these from env or hardcode for now.
// You create these in the Stripe Dashboard under Products → Pricing.
var stripePriceIDs = map[string]string{
	"starter": "", // e.g. "price_1xxxxxx"
	"team":    "", // e.g. "price_1xxxxxx"
}

// StripeProvider implements PaymentProvider using Stripe.
type StripeProvider struct {
	webhookSecret string
}

// NewStripeProvider initialises Stripe with the given secret key.
func NewStripeProvider(secretKey string, webhookSecret string, priceStarter string, priceTeam string) *StripeProvider {
	stripe.Key = secretKey
	if priceStarter != "" {
		stripePriceIDs["starter"] = priceStarter
	}
	if priceTeam != "" {
		stripePriceIDs["team"] = priceTeam
	}
	return &StripeProvider{webhookSecret: webhookSecret}
}

// CreateCheckoutSession creates a Stripe Checkout Session for a subscription.
func (s *StripeProvider) CreateCheckoutSession(customerEmail string, userID string, plan string, successURL string, cancelURL string) (string, error) {
	priceID, ok := stripePriceIDs[plan]
	if !ok || priceID == "" {
		return "", fmt.Errorf("no Stripe price configured for plan %q", plan)
	}

	params := &stripe.CheckoutSessionParams{
		Mode:              stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL:        stripe.String(successURL),
		CancelURL:         stripe.String(cancelURL),
		CustomerEmail:     stripe.String(customerEmail),
		ClientReferenceID: stripe.String(userID),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
	}
	params.AddMetadata("user_id", userID)
	params.AddMetadata("plan", plan)

	session, err := checkoutSession.New(params)
	if err != nil {
		return "", fmt.Errorf("stripe: create checkout session: %w", err)
	}

	return session.URL, nil
}

// CreatePortalSession creates a Stripe billing portal session.
func (s *StripeProvider) CreatePortalSession(customerID string, returnURL string) (string, error) {
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(returnURL),
	}

	session, err := billingSession.New(params)
	if err != nil {
		return "", fmt.Errorf("stripe: create portal session: %w", err)
	}

	return session.URL, nil
}

// VerifyWebhookPayload verifies and parses a Stripe webhook event.
func (s *StripeProvider) VerifyWebhookPayload(payload []byte, sigHeader string) (WebhookEvent, error) {
	event, err := webhook.ConstructEvent(payload, sigHeader, s.webhookSecret)
	if err != nil {
		return WebhookEvent{}, fmt.Errorf("stripe: verify webhook: %w", err)
	}

	we := WebhookEvent{
		Type: string(event.Type),
	}

	switch event.Type {
	case "checkout.session.completed":
		var cs stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &cs); err != nil {
			return we, fmt.Errorf("stripe: unmarshal checkout session: %w", err)
		}
		we.CustomerID = string(cs.Customer.ID)
		we.CustomerEmail = cs.CustomerEmail
		if cs.Metadata != nil {
			we.UserID = cs.Metadata["user_id"]
			we.Plan = cs.Metadata["plan"]
		}
		we.Status = "active"

	case "customer.subscription.updated", "customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			return we, fmt.Errorf("stripe: unmarshal subscription: %w", err)
		}
		we.CustomerID = string(sub.Customer.ID)
		we.SubscriptionID = sub.ID
		we.Status = strings.ToLower(string(sub.Status))

		// Try to infer plan from price
		if len(sub.Items.Data) > 0 {
			priceID := sub.Items.Data[0].Price.ID
			for plan, pid := range stripePriceIDs {
				if pid == priceID {
					we.Plan = plan
					break
				}
			}
		}

	default:
		log.Printf("[stripe] unhandled webhook event type: %s", event.Type)
	}

	return we, nil
}
