package services

// Normalised webhook event types — providers map their specific events to these.
const (
	EventSubscriptionActivated = "subscription_activated"
	EventSubscriptionUpdated   = "subscription_updated"
	EventSubscriptionCancelled = "subscription_cancelled"
)

// PaymentProvider is the interface all payment gateways implement.
// Currently backed by Razorpay; designed for future Stripe addition.
type PaymentProvider interface {
	// CreateCheckoutSession starts a new subscription checkout.
	// Returns a URL to redirect the user to.
	CreateCheckoutSession(customerEmail string, userID string, plan string, successURL string, cancelURL string) (string, error)

	// CreatePortalSession returns a URL for the user to manage billing.
	// Razorpay: returns a self-managed cancel endpoint.
	// Stripe: returns the Stripe billing portal URL.
	CreatePortalSession(customerID string, returnURL string) (string, error)

	// VerifyWebhookPayload verifies an incoming webhook and returns a normalised event.
	VerifyWebhookPayload(payload []byte, sigHeader string) (WebhookEvent, error)
}

// WebhookEvent is a normalised representation of a payment webhook event.
type WebhookEvent struct {
	Type           string // one of the Event* constants above
	CustomerID     string // provider customer id
	CustomerEmail  string
	SubscriptionID string
	Plan           string // mapped plan id (free/starter/team)
	Status         string // active, cancelled, past_due …
	UserID         string // from metadata/notes
}
