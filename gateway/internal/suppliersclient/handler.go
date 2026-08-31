package supplierclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

func (c *Client) IssueKey(ctx context.Context, sku, requestID string) (string, error) {
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

func (c *Client) sendRequest(ctx context.Context, targetURL string, payload IssueRequest) (string, error) {
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

	var issueResp IssueResponse
	if err := json.NewDecoder(resp.Body).Decode(&issueResp); err != nil {
		return "", fmt.Errorf("decode response (status: %d): %w", resp.StatusCode, err)
	}

	if resp.StatusCode == http.StatusNotFound {
		if issueResp.Error == "out of stock" {
			return "", fmt.Errorf("out of stock")
		}
		return "", fmt.Errorf("not found error: %s", issueResp.Error)
	}

	if resp.StatusCode >= 500 {
		return "", fmt.Errorf("server error status: %d", resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d, err: %s", resp.StatusCode, issueResp.Error)
	}

	if issueResp.Code == "" {
		return "", fmt.Errorf("empty code received with ok status")
	}

	return issueResp.Code, nil
}
