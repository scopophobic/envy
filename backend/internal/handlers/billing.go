package handlers

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/envo/backend/internal/middleware"
	"github.com/envo/backend/internal/services"
	"github.com/gin-gonic/gin"
)

type BillingHandler struct {
	billingService *services.BillingService // nil until RAZORPAY_KEY_ID and RAZORPAY_KEY_SECRET are set
}

func NewBillingHandler(billingService *services.BillingService) *BillingHandler {
	return &BillingHandler{billingService: billingService}
}

func (h *BillingHandler) requireBilling(c *gin.Context) bool {
	if h.billingService != nil {
		return true
	}
	c.JSON(http.StatusServiceUnavailable, gin.H{
		"error": "billing_not_configured",
		"message": "Razorpay is not configured on this API. Set RAZORPAY_KEY_ID and RAZORPAY_KEY_SECRET (and plan IDs), restart the server, and ensure the frontend VITE_API_URL points to this API—not the Vite dev port.",
	})
	return false
}

// GET /api/v1/billing/status
func (h *BillingHandler) Status(c *gin.Context) {
	if _, err := middleware.GetCurrentUser(c); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	if h.billingService == nil {
		c.JSON(http.StatusOK, gin.H{
			"checkout_enabled":    false,
			"starter_plan_ready":    false,
			"team_plan_ready":       false,
			"message":               "Billing is not configured on the server.",
		})
		return
	}
	starterOK, teamOK := services.RazorpayPlanIDsConfigured()
	c.JSON(http.StatusOK, gin.H{
		"checkout_enabled":     true,
		"starter_plan_ready":   starterOK,
		"team_plan_ready":      teamOK,
	})
}

// POST /api/v1/billing/checkout
func (h *BillingHandler) CreateCheckoutSession(c *gin.Context) {
	if !h.requireBilling(c) {
		return
	}
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

// POST /api/v1/billing/portal
func (h *BillingHandler) CreatePortalSession(c *gin.Context) {
	if !h.requireBilling(c) {
		return
	}
	user, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	url, err := h.billingService.CreatePortal(user.PaymentCustomerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url})
}

// POST /api/v1/billing/webhook
func (h *BillingHandler) HandleWebhook(c *gin.Context) {
	if h.billingService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "billing_not_configured"})
		return
	}
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read body"})
		return
	}

	// Razorpay uses X-Razorpay-Signature, Stripe uses Stripe-Signature
	sig := c.GetHeader("X-Razorpay-Signature")
	if sig == "" {
		sig = c.GetHeader("Stripe-Signature")
	}

	if err := h.billingService.HandleWebhook(payload, sig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}

// POST /api/v1/billing/orders — create a Razorpay order for Standard Web Checkout.
func (h *BillingHandler) CreateOrder(c *gin.Context) {
	if !h.requireBilling(c) {
		return
	}
	_, err := middleware.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var body struct {
		Amount   int64  `json:"amount" binding:"required"`
		Currency string `json:"currency"`
		Receipt  string `json:"receipt"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount is required (in smallest currency unit, min 100)"})
		return
	}

	orderID, amount, currency, err := h.billingService.CreatePaymentOrder(body.Amount, body.Currency, body.Receipt)
	if err != nil {
		err = services.MapRazorpayOrderError(err)
		var authErr *services.ErrRazorpayUnauthorized
		if errors.As(err, &authErr) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": authErr.Error()})
			return
		}
		if isRazorpayOrderClientError(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"order_id": orderID,
		"amount":   amount,
		"currency": currency,
	})
}

// POST /api/v1/billing/verify-payment — verify Standard Checkout payment signature.
func (h *BillingHandler) VerifyPayment(c *gin.Context) {
	if !h.requireBilling(c) {
		return
	}
	if _, err := middleware.GetCurrentUser(c); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var body struct {
		RazorpayPaymentID   string `json:"razorpay_payment_id" binding:"required"`
		RazorpayOrderID     string `json:"razorpay_order_id" binding:"required"`
		RazorpaySignature   string `json:"razorpay_signature" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "razorpay_payment_id, razorpay_order_id, and razorpay_signature are required"})
		return
	}

	if err := h.billingService.VerifyStandardPayment(body.RazorpayOrderID, body.RazorpayPaymentID, body.RazorpaySignature); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "verified": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{"verified": true, "success": true})
}

func isRazorpayOrderClientError(err error) bool {
	s := err.Error()
	return strings.HasPrefix(s, "amount ") ||
		strings.HasPrefix(s, "currency ") ||
		strings.Contains(s, "exceeds maximum allowed")
}
