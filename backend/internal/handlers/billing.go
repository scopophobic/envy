package handlers

import (
	"io"
	"net/http"

	"github.com/envo/backend/internal/middleware"
	"github.com/envo/backend/internal/services"
	"github.com/gin-gonic/gin"
)

// BillingHandler handles billing-related HTTP endpoints.
type BillingHandler struct {
	billingService *services.BillingService
}

// NewBillingHandler creates a new billing handler.
func NewBillingHandler(billingService *services.BillingService) *BillingHandler {
	return &BillingHandler{billingService: billingService}
}

// CreateCheckoutSession starts a Stripe checkout for subscription upgrade.
// POST /api/v1/billing/checkout
func (h *BillingHandler) CreateCheckoutSession(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var body struct {
		Plan string `json:"plan" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plan is required"})
		return
	}

	url, err := h.billingService.CreateCheckout(user.ID, user.Email, body.Plan)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url})
}

// CreatePortalSession opens Stripe billing portal for managing subscription.
// POST /api/v1/billing/portal
func (h *BillingHandler) CreatePortalSession(c *gin.Context) {
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	url, err := h.billingService.CreatePortal(user.StripeCustomerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url})
}

// HandleWebhook processes incoming Stripe webhook events.
// POST /api/v1/billing/webhook
func (h *BillingHandler) HandleWebhook(c *gin.Context) {
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read body"})
		return
	}

	sigHeader := c.GetHeader("Stripe-Signature")

	if err := h.billingService.HandleWebhook(payload, sigHeader); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}
