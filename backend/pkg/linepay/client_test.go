package linepay

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestPayment_Success(t *testing.T) {
	// Mock LINE Pay server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/payments/request" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		// Verify required headers
		if r.Header.Get("X-LINE-ChannelId") != "test-channel" {
			t.Error("missing or wrong X-LINE-ChannelId")
		}
		if r.Header.Get("X-LINE-Authorization") == "" {
			t.Error("missing X-LINE-Authorization")
		}
		if r.Header.Get("X-LINE-Authorization-Nonce") == "" {
			t.Error("missing X-LINE-Authorization-Nonce")
		}

		resp := map[string]interface{}{
			"returnCode":    "0000",
			"returnMessage": "Success.",
			"info": map[string]interface{}{
				"paymentUrl": map[string]interface{}{
					"web": "https://sandbox-web-pay.line.me/web/payment/wait?transactionReserveId=test123",
					"app": "line://pay/payment/test123",
				},
				"transactionId": 2024123456789,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-channel", "test-secret", server.URL, "http://localhost:3000")

	output, err := client.RequestPayment(RequestPaymentInput{
		OrderID:     "order-001",
		Amount:      5600,
		ProductName: "演唱會門票",
		Quantity:    2,
		Price:       2800,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.PaymentURL == "" {
		t.Error("expected non-empty payment URL")
	}
	if output.TransactionID == 0 {
		t.Error("expected non-zero transaction ID")
	}
}

func TestRequestPayment_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"returnCode":    "1101",
			"returnMessage": "A purchaser status error.",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-channel", "test-secret", server.URL, "http://localhost:3000")
	_, err := client.RequestPayment(RequestPaymentInput{
		OrderID: "order-001",
		Amount:  5600,
	})

	if err == nil {
		t.Fatal("expected error for LINE Pay failure")
	}
}

func TestConfirmPayment_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		// Path should contain transaction ID
		expectedPath := "/v3/payments/tx-12345/confirm"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}

		resp := map[string]interface{}{
			"returnCode":    "0000",
			"returnMessage": "Success.",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-channel", "test-secret", server.URL, "http://localhost:3000")
	err := client.ConfirmPayment(ConfirmPaymentInput{
		TransactionID: "tx-12345",
		Amount:        5600,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfirmPayment_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"returnCode":    "1150",
			"returnMessage": "Transaction has already been confirmed.",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-channel", "test-secret", server.URL, "http://localhost:3000")
	err := client.ConfirmPayment(ConfirmPaymentInput{
		TransactionID: "tx-12345",
		Amount:        5600,
	})

	if err == nil {
		t.Fatal("expected error for failed confirmation")
	}
}

func TestVoidPayment_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		expectedPath := "/v3/payments/authorizations/tx-void/void"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}
		if r.Header.Get("X-LINE-Authorization") == "" {
			t.Error("missing X-LINE-Authorization")
		}

		resp := map[string]interface{}{
			"returnCode":    "0000",
			"returnMessage": "Success.",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-channel", "test-secret", server.URL, "http://localhost:3000")
	if err := client.VoidPayment("tx-void"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVoidPayment_AlreadyVoidedIsSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"returnCode":    "1165",
			"returnMessage": "A transaction has already been voided.",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-channel", "test-secret", server.URL, "http://localhost:3000")
	if err := client.VoidPayment("tx-voided"); err != nil {
		t.Fatalf("expected already-voided response to be idempotent success, got: %v", err)
	}
}

func TestConfirmPaymentWithRetry_EventualSuccess(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		var resp map[string]interface{}
		if attempts < 3 {
			resp = map[string]interface{}{
				"returnCode":    "9000",
				"returnMessage": "Internal error.",
			}
		} else {
			resp = map[string]interface{}{
				"returnCode":    "0000",
				"returnMessage": "Success.",
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-channel", "test-secret", server.URL, "http://localhost:3000")
	err := client.ConfirmPaymentWithRetry(ConfirmPaymentInput{
		TransactionID: "tx-retry",
		Amount:        5600,
	})

	if err != nil {
		t.Fatalf("expected retry to eventually succeed, got: %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestConfirmPaymentWithRetry_AllFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"returnCode":    "9000",
			"returnMessage": "Internal error.",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-channel", "test-secret", server.URL, "http://localhost:3000")
	err := client.ConfirmPaymentWithRetry(ConfirmPaymentInput{
		TransactionID: "tx-allfail",
		Amount:        5600,
	})

	if err == nil {
		t.Fatal("expected error after all retries exhausted")
	}
}

func TestSignature(t *testing.T) {
	client := NewClient("channel1", "my-secret", "https://example.com", "http://localhost:3000")
	sig := client.sign("my-secret", "/v3/payments/request", `{"amount":100}`, "nonce123")
	if sig == "" {
		t.Error("expected non-empty signature")
	}

	// Same inputs should produce same signature
	sig2 := client.sign("my-secret", "/v3/payments/request", `{"amount":100}`, "nonce123")
	if sig != sig2 {
		t.Error("signature should be deterministic")
	}

	// Different inputs should produce different signature
	sig3 := client.sign("my-secret", "/v3/payments/request", `{"amount":200}`, "nonce123")
	if sig == sig3 {
		t.Error("different body should produce different signature")
	}
}
