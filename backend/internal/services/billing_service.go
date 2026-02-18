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

func NewBillingService(provider PaymentProvider, frontendURL string) *BillingService {
	return &BillingService{provider: provider, frontendURL: frontendURL}
}

func (s *BillingService) CreateCheckout(userID uuid.UUID, email string, plan string) (string, error) {
	if plan != "starter" && plan != "team" {
		return "", fmt.Errorf("invalid plan: %s", plan)
	}

	successURL := fmt.Sprintf("%s/settings?checkout=success", s.frontendURL)
	cancelURL := fmt.Sprintf("%s/settings?checkout=cancel", s.frontendURL)

	return s.provider.CreateCheckoutSession(email, userID.String(), plan, successURL, cancelURL)
}

func (s *BillingService) CreatePortal(paymentCustomerID string) (string, error) {
	if paymentCustomerID == "" {
		return "", fmt.Errorf("no payment customer on file; user may still be on free tier")
	}
	returnURL := fmt.Sprintf("%s/settings", s.frontendURL)
	return s.provider.CreatePortalSession(paymentCustomerID, returnURL)
}

func (s *BillingService) HandleWebhook(payload []byte, sigHeader string) error {
	evt, err := s.provider.VerifyWebhookPayload(payload, sigHeader)
	if err != nil {
		return err
	}

	switch evt.Type {
	case EventSubscriptionActivated:
		return s.onActivated(evt)
	case EventSubscriptionUpdated:
		return s.onUpdated(evt)
	case EventSubscriptionCancelled:
		return s.onCancelled(evt)
	default:
		log.Printf("[billing] unhandled normalised event: %s", evt.Type)
	}
	return nil
}

func (s *BillingService) onActivated(evt WebhookEvent) error {
	if evt.UserID == "" {
		log.Printf("[billing] subscription activated but no user_id in metadata")
		return nil
	}

	userID, err := uuid.Parse(evt.UserID)
	if err != nil {
		return fmt.Errorf("invalid user_id: %w", err)
	}

	db := database.GetDB()
	updates := map[string]interface{}{
		"subscription_tier":    evt.Plan,
		"subscription_status":  "active",
		"payment_customer_id":  evt.CustomerID,
	}

	if err := db.Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		return fmt.Errorf("update user after activation: %w", err)
	}

	log.Printf("[billing] user %s upgraded to %s", userID, evt.Plan)
	return nil
}

func (s *BillingService) onUpdated(evt WebhookEvent) error {
	db := database.GetDB()
	updates := map[string]interface{}{
		"subscription_status": evt.Status,
	}
	if evt.Plan != "" {
		updates["subscription_tier"] = evt.Plan
	}
	if err := db.Model(&models.User{}).Where("payment_customer_id = ?", evt.CustomerID).Updates(updates).Error; err != nil {
		return fmt.Errorf("update subscription: %w", err)
	}
	return nil
}

func (s *BillingService) onCancelled(evt WebhookEvent) error {
	db := database.GetDB()
	updates := map[string]interface{}{
		"subscription_tier":   "free",
		"subscription_status": "cancelled",
	}
	if err := db.Model(&models.User{}).Where("payment_customer_id = ?", evt.CustomerID).Updates(updates).Error; err != nil {
		return fmt.Errorf("cancel subscription: %w", err)
	}
	return nil
}
