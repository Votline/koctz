package supplierclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"
)

type IssueRequest struct {
	SKU       string `json:"sku"`
	RequestID string `json:"request_id"`
}

type IssueResponse struct {
	Code  string `json:"code,omitempty"`
	Error string `json:"error,omitempty"`
}

func (c *SuplierClient) IssueKey(ctx context.Context, sku, requestID string) (string, error) {
	const op = "supplierclient.IssueKey"

	reqBody := IssueRequest{
		SKU:       sku,
		RequestID: requestID,
	}

	code, err := c.sendRequest(ctx, c.urlA, reqBody)
	if err == nil {
		return code, nil
	}
	c.log.Warn("supplier A failed, trying fallback supplier B", zap.Error(err), zap.String("request_id", requestID))

	code, errB := c.sendRequest(ctx, c.urlB, reqBody)
	if errB == nil {
		return code, nil
	}

	return "", fmt.Errorf("%s: both suppliers failed. A err: %v, B err: %v", op, err, errB)
}

func (c *SuplierClient) sendRequest(ctx context.Context, targetURL string, payload IssueRequest) (string, error) {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		var issueResp IssueResponse
		if json.Unmarshal(respBytes, &issueResp) == nil && issueResp.Error == "out of stock" {
			return "", fmt.Errorf("out of stock")
		}
		return "", fmt.Errorf("not found error: %s", string(respBytes))
	}

	if resp.StatusCode >= 500 {
		return "", fmt.Errorf("server error status: %d, body: %s", resp.StatusCode, string(respBytes))
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respBytes))
	}

	var issueResp IssueResponse
	if err := json.Unmarshal(respBytes, &issueResp); err != nil {
		return "", fmt.Errorf("decode success response: %w, body: %s", err, string(respBytes))
	}

	if issueResp.Code == "" {
		return "", fmt.Errorf("empty code received with ok status")
	}

	return issueResp.Code, nil
}
