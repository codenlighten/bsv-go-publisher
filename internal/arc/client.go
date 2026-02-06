package arc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client handles communication with BSV ARC API
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new ARC client
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// TxStatus represents the status of a transaction in ARC
type TxStatus string

const (
	TxStatusReceived      TxStatus = "RECEIVED"
	TxStatusStored        TxStatus = "STORED"
	TxStatusAnnounced     TxStatus = "ANNOUNCED"
	TxStatusSent          TxStatus = "SENT"
	TxStatusSeenOnNetwork TxStatus = "SEEN_ON_NETWORK"
	TxStatusAccepted      TxStatus = "ACCEPTED_BY_NETWORK"
	TxStatusMined         TxStatus = "MINED"
	TxStatusRejected      TxStatus = "REJECTED"
	TxStatusDoubleSpend   TxStatus = "DOUBLE_SPEND_ATTEMPTED"
)

// TxResponse represents ARC's response for a single transaction
type TxResponse struct {
	TxID         string   `json:"txid"`
	TxStatus     TxStatus `json:"txStatus"`
	ExtraInfo    string   `json:"extraInfo,omitempty"`
	CompetingTxs []string `json:"competingTxs,omitempty"`
	BlockHash    string   `json:"blockHash,omitempty"`
	BlockHeight  int64    `json:"blockHeight,omitempty"`
	Timestamp    string   `json:"timestamp,omitempty"`
	MerklePath   string   `json:"merklePath,omitempty"`
	// Error fields (present when status >= 400)
	Status int    `json:"status,omitempty"`
	Title  string `json:"title,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// ErrorResponse represents an ARC error
type ErrorResponse struct {
	Status int    `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	TxID   string `json:"txid,omitempty"`
}

// BroadcastBatch submits multiple transactions to ARC in a single request
// This is the key method for the "train" functionality
func (c *Client) BroadcastBatch(ctx context.Context, hexes []string) ([]TxResponse, error) {
	if len(hexes) == 0 {
		return nil, fmt.Errorf("no transactions to broadcast")
	}

	// ARC expects transactions separated by newlines
	body := strings.Join(hexes, "\n")

	url := c.baseURL + "/v1/txs"

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers according to ARC spec
	req.Header.Set("Content-Type", "text/plain")
	if c.apiKey != "" && c.apiKey != "your_arc_api_token_here" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	req.Header.Set("X-WaitForStatus", "7") // Wait until ACCEPTED_BY_NETWORK
	// Don't set X-CallbackUrl if empty - ARC rejects empty values

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Debug logging
	fmt.Printf("ARC Response Status: %d\n", resp.StatusCode)
	fmt.Printf("ARC Response Body: %s\n", string(bodyBytes))

	// Handle error responses
	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if err := json.Unmarshal(bodyBytes, &errResp); err == nil {
			return nil, fmt.Errorf("ARC error %d: %s - %s", errResp.Status, errResp.Title, errResp.Detail)
		}
		return nil, fmt.Errorf("ARC returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse successful response
	var txResponses []TxResponse
	if err := json.Unmarshal(bodyBytes, &txResponses); err != nil {
		return nil, fmt.Errorf("failed to parse response (body: %s): %w", string(bodyBytes), err)
	}

	return txResponses, nil
}

// BroadcastSingle broadcasts a single transaction
func (c *Client) BroadcastSingle(ctx context.Context, hex string) (*TxResponse, error) {
	responses, err := c.BroadcastBatch(ctx, []string{hex})
	if err != nil {
		return nil, err
	}

	if len(responses) == 0 {
		return nil, fmt.Errorf("no response from ARC")
	}

	return &responses[0], nil
}

// GetTransactionStatus queries the status of a transaction
func (c *Client) GetTransactionStatus(ctx context.Context, txid string) (*TxResponse, error) {
	url := fmt.Sprintf("%s/v1/tx/%s", c.baseURL, txid)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("transaction not found")
	}

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ARC returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var txResp TxResponse
	if err := json.NewDecoder(resp.Body).Decode(&txResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &txResp, nil
}

// PolicyQuote gets the current fee policy from ARC
type PolicyQuote struct {
	Policy struct {
		MaxScriptSizePolicy    int `json:"maxscriptsizepolicy"`
		MaxTxSigopsCountPolicy int `json:"maxtxsigopscountspolicy"`
		MaxTxSizePolicy        int `json:"maxtxsizepolicy"`
		MiningFee              struct {
			Satoshis int `json:"satoshis"`
			Bytes    int `json:"bytes"`
		} `json:"miningFee"`
	} `json:"policy"`
	Timestamp string `json:"timestamp"`
}

// GetPolicyQuote retrieves current mining policy
func (c *Client) GetPolicyQuote(ctx context.Context) (*PolicyQuote, error) {
	url := c.baseURL + "/v1/policy"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var quote PolicyQuote
	if err := json.NewDecoder(resp.Body).Decode(&quote); err != nil {
		return nil, err
	}

	return &quote, nil
}

// Health checks if ARC is reachable and healthy
func (c *Client) Health(ctx context.Context) error {
	url := c.baseURL + "/v1/health"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ARC unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ARC health check failed: status %d", resp.StatusCode)
	}

	return nil
}
