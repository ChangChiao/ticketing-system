package linepay

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Client struct {
	channelID     string
	channelSecret string
	baseURL       string
	appBaseURL    string
	httpClient    *http.Client
}

func NewClient(channelID, channelSecret, baseURL, appBaseURL string) *Client {
	return &Client{
		channelID:     channelID,
		channelSecret: channelSecret,
		baseURL:       baseURL,
		appBaseURL:    appBaseURL,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
	}
}

type RequestPaymentInput struct {
	OrderID       string
	Amount        int
	ProductName   string
	Quantity      int
	Price         int
	CallbackToken string
}

type RequestPaymentOutput struct {
	PaymentURL    string
	TransactionID int64
}

func (c *Client) RequestPayment(input RequestPaymentInput) (*RequestPaymentOutput, error) {
	body := map[string]interface{}{
		"amount":   input.Amount,
		"currency": "TWD",
		"orderId":  input.OrderID,
		"packages": []map[string]interface{}{
			{
				"id":     "pkg-1",
				"amount": input.Amount,
				"name":   input.ProductName,
				"products": []map[string]interface{}{
					{
						"name":     input.ProductName,
						"quantity": input.Quantity,
						"price":    input.Price,
					},
				},
			},
		},
		"redirectUrls": map[string]string{
			"confirmUrl": fmt.Sprintf("%s/api/payments/confirm?orderId=%s&token=%s", c.appBaseURL, input.OrderID, input.CallbackToken),
			"cancelUrl":  fmt.Sprintf("%s/api/payments/cancel?orderId=%s&token=%s", c.appBaseURL, input.OrderID, input.CallbackToken),
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	path := "/v3/payments/request"
	nonce := uuid.New().String()

	req, err := http.NewRequest("POST", c.baseURL+path, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	signature := c.sign(c.channelSecret, path, string(jsonBody), nonce)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-LINE-ChannelId", c.channelID)
	req.Header.Set("X-LINE-Authorization-Nonce", nonce)
	req.Header.Set("X-LINE-Authorization", signature)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		ReturnCode    string `json:"returnCode"`
		ReturnMessage string `json:"returnMessage"`
		Info          struct {
			PaymentURL struct {
				Web string `json:"web"`
				App string `json:"app"`
			} `json:"paymentUrl"`
			TransactionID int64 `json:"transactionId"`
		} `json:"info"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	if result.ReturnCode != "0000" {
		return nil, fmt.Errorf("LINE Pay error: %s - %s", result.ReturnCode, result.ReturnMessage)
	}

	return &RequestPaymentOutput{
		PaymentURL:    result.Info.PaymentURL.Web,
		TransactionID: result.Info.TransactionID,
	}, nil
}

type ConfirmPaymentInput struct {
	TransactionID string
	Amount        int
}

// ConfirmPaymentWithRetry calls ConfirmPayment with exponential backoff (up to 3 retries).
func (c *Client) ConfirmPaymentWithRetry(input ConfirmPaymentInput) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		lastErr = c.ConfirmPayment(input)
		if lastErr == nil {
			return nil
		}
		// Exponential backoff: 1s, 2s, 4s
		backoff := time.Duration(1<<uint(attempt)) * time.Second
		time.Sleep(backoff)
	}
	return fmt.Errorf("LINE Pay confirm failed after 3 retries: %w", lastErr)
}

func (c *Client) ConfirmPayment(input ConfirmPaymentInput) error {
	body := map[string]interface{}{
		"amount":   input.Amount,
		"currency": "TWD",
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/v3/payments/%s/confirm", input.TransactionID)
	nonce := uuid.New().String()

	req, err := http.NewRequest("POST", c.baseURL+path, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}

	signature := c.sign(c.channelSecret, path, string(jsonBody), nonce)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-LINE-ChannelId", c.channelID)
	req.Header.Set("X-LINE-Authorization-Nonce", nonce)
	req.Header.Set("X-LINE-Authorization", signature)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result struct {
		ReturnCode    string `json:"returnCode"`
		ReturnMessage string `json:"returnMessage"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return err
	}

	if result.ReturnCode != "0000" {
		return fmt.Errorf("LINE Pay confirm error: %s - %s", result.ReturnCode, result.ReturnMessage)
	}

	return nil
}

func (c *Client) VoidPaymentWithRetry(transactionID string) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		lastErr = c.VoidPayment(transactionID)
		if lastErr == nil {
			return nil
		}
		backoff := time.Duration(1<<uint(attempt)) * time.Second
		time.Sleep(backoff)
	}
	return fmt.Errorf("LINE Pay void failed after 3 retries: %w", lastErr)
}

func (c *Client) VoidPayment(transactionID string) error {
	path := fmt.Sprintf("/v3/payments/authorizations/%s/void", transactionID)
	nonce := uuid.New().String()

	req, err := http.NewRequest("POST", c.baseURL+path, nil)
	if err != nil {
		return err
	}

	signature := c.sign(c.channelSecret, path, "", nonce)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-LINE-ChannelId", c.channelID)
	req.Header.Set("X-LINE-Authorization-Nonce", nonce)
	req.Header.Set("X-LINE-Authorization", signature)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result struct {
		ReturnCode    string `json:"returnCode"`
		ReturnMessage string `json:"returnMessage"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return err
	}

	if result.ReturnCode != "0000" && result.ReturnCode != "1165" {
		return fmt.Errorf("LINE Pay void error: %s - %s", result.ReturnCode, result.ReturnMessage)
	}

	return nil
}

func (c *Client) sign(secret, path, body, nonce string) string {
	message := secret + path + body + nonce
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
