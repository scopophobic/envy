package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	razorpay "github.com/razorpay/razorpay-go"
)

func TestVerifyStandardPaymentSignature_match(t *testing.T) {
	secret := "whsec_test"
	client := razorpay.NewClient("rzp_test_x", secret)
	r := &RazorpayProvider{client: client, webhookSecret: ""}

	orderID := "order_ABC"
	paymentID := "pay_XYZ"
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(orderID + "|" + paymentID))
	sig := hex.EncodeToString(mac.Sum(nil))

	if err := r.VerifyStandardPaymentSignature(orderID, paymentID, sig); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyStandardPaymentSignature_mismatch(t *testing.T) {
	client := razorpay.NewClient("key", "secret")
	r := &RazorpayProvider{client: client, webhookSecret: ""}

	if r.VerifyStandardPaymentSignature("order_1", "pay_1", "deadbeef") == nil {
		t.Fatal("expected signature mismatch")
	}
}

func TestVerifyStandardPaymentSignature_missingFields(t *testing.T) {
	client := razorpay.NewClient("k", "s")
	r := &RazorpayProvider{client: client, webhookSecret: ""}
	if r.VerifyStandardPaymentSignature("", "pay_1", "sig") == nil {
		t.Fatal("expected error for empty order id")
	}
}
