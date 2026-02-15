package services

// PaymentProvider is the interface all payment gateways implement.
// Currently backed by Stripe; designed for future Razorpay addition.
type PaymentProvider interface {
	// CreateCheckoutSession starts a new subscription checkout.
	// Returns a URL to redirect the user to.
	CreateCheckoutSession(customerEmail string, userID string, plan string, successURL string, cancelURL string) (string, error)

	// CreatePortalSession creates a self-service billing portal session.
	// Returns a URL to redirect the user to.
	CreatePortalSession(customerID string, returnURL string) (string, error)

	// VerifyWebhookPayload verifies an incoming webhook signature and returns
	// the raw event payload. Implementation-specific.
	VerifyWebhookPayload(payload []byte, sigHeader string) (WebhookEvent, error)
}

// WebhookEvent is a normalised representation of a payment webhook event.
type WebhookEvent struct {
	Type           string // e.g. "checkout.session.completed", "customer.subscription.updated"
	CustomerID     string // provider customer id
	CustomerEmail  string
	SubscriptionID string
	Plan           string // mapped plan id (free/starter/team)
	Status         string // active, cancelled, past_due …
	UserID         string // from metadata
}
