package services

import (
	"fmt"
	"log"

	"github.com/envo/backend/internal/database"
	"github.com/envo/backend/internal/models"
	"github.com/google/uuid"
)

// BillingService orchestrates payment flows through a PaymentProvider.
type BillingService struct {
	provider    PaymentProvider
	frontendURL string
}

// NewBillingService creates a billing service with the given payment provider.
func NewBillingService(provider PaymentProvider, frontendURL string) *BillingService {
	return &BillingService{provider: provider, frontendURL: frontendURL}
}

// CreateCheckout starts a subscription checkout for the given user + plan.
func (s *BillingService) CreateCheckout(userID uuid.UUID, email string, plan string) (string, error) {
	if plan != "starter" && plan != "team" {
		return "", fmt.Errorf("invalid plan: %s", plan)
	}

	successURL := fmt.Sprintf("%s/settings?checkout=success", s.frontendURL)
	cancelURL := fmt.Sprintf("%s/settings?checkout=cancel", s.frontendURL)

	url, err := s.provider.CreateCheckoutSession(email, userID.String(), plan, successURL, cancelURL)
	if err != nil {
		return "", err
	}
	return url, nil
}

// CreatePortal opens the provider's billing management portal.
func (s *BillingService) CreatePortal(stripeCustomerID string) (string, error) {
	if stripeCustomerID == "" {
		return "", fmt.Errorf("no payment customer ID on file; user may still be on free tier")
	}
	returnURL := fmt.Sprintf("%s/settings", s.frontendURL)
	return s.provider.CreatePortalSession(stripeCustomerID, returnURL)
}

// HandleWebhook processes a verified webhook event.
func (s *BillingService) HandleWebhook(payload []byte, sigHeader string) error {
	evt, err := s.provider.VerifyWebhookPayload(payload, sigHeader)
	if err != nil {
		return err
	}

	switch evt.Type {
	case "checkout.session.completed":
		return s.onCheckoutCompleted(evt)
	case "customer.subscription.updated":
		return s.onSubscriptionUpdated(evt)
	case "customer.subscription.deleted":
		return s.onSubscriptionDeleted(evt)
	default:
		log.Printf("[billing] unhandled event: %s", evt.Type)
	}
	return nil
}

func (s *BillingService) onCheckoutCompleted(evt WebhookEvent) error {
	if evt.UserID == "" {
		log.Printf("[billing] checkout.session.completed: no user_id in metadata")
		return nil
	}

	userID, err := uuid.Parse(evt.UserID)
	if err != nil {
		return fmt.Errorf("invalid user_id in metadata: %w", err)
	}

	db := database.GetDB()

	updates := map[string]interface{}{
		"subscription_tier":   evt.Plan,
		"subscription_status": "active",
		"stripe_customer_id":  evt.CustomerID,
	}

	if err := db.Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		return fmt.Errorf("update user after checkout: %w", err)
	}

	log.Printf("[billing] user %s upgraded to %s", userID, evt.Plan)
	return nil
}

func (s *BillingService) onSubscriptionUpdated(evt WebhookEvent) error {
	db := database.GetDB()
	updates := map[string]interface{}{
		"subscription_status": evt.Status,
	}
	if evt.Plan != "" {
		updates["subscription_tier"] = evt.Plan
	}
	if err := db.Model(&models.User{}).Where("stripe_customer_id = ?", evt.CustomerID).Updates(updates).Error; err != nil {
		return fmt.Errorf("update subscription: %w", err)
	}
	return nil
}

func (s *BillingService) onSubscriptionDeleted(evt WebhookEvent) error {
	db := database.GetDB()
	updates := map[string]interface{}{
		"subscription_tier":   "free",
		"subscription_status": "cancelled",
	}
	if err := db.Model(&models.User{}).Where("stripe_customer_id = ?", evt.CustomerID).Updates(updates).Error; err != nil {
		return fmt.Errorf("cancel subscription: %w", err)
	}
	return nil
}
