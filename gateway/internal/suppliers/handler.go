package suppliers

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type IssueReq struct {
	SKU       string `json:"sku"`
	RequestID string `json:"request_id"`
}

type IssueResp struct {
	Code  string `json:"code,omitempty"`
	Error string `json:"error,omitempty"`
}

func (s *suppliersservice) handleSupplierA(w http.ResponseWriter, r *http.Request) {
	s.processIssue(w, r, "SUPPLIER_A")
}

func (s *suppliersservice) handleSupplierB(w http.ResponseWriter, r *http.Request) {
	s.processIssue(w, r, "SUPPLIER_B")
}

func (s *suppliersservice) processIssue(w http.ResponseWriter, r *http.Request, envPrefix string) {
	const op = "suppliers.processIssue"

	var req IssueReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.SKU == "" || req.RequestID == "" {
		http.Error(w, "sku and request_id are required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.ctxTimeout)
	defer cancel()

	existingKey, err := s.kdb.GetByRequestID(ctx, req.RequestID)
	if err == nil && existingKey != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(IssueResp{Code: existingKey.Code})
		return
	}

	delayMs := getEnvInt(envPrefix+"_DELAY_MS", 0)
	if delayMs > 0 {
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}

	errorRate := getEnvInt(envPrefix+"_ERROR_RATE", 0)
	if errorRate > 0 && rand.Intn(100) < errorRate {
		http.Error(w, "internal supplier error", http.StatusInternalServerError)
		return
	}

	key, err := s.kdb.ReserveAndIssueKey(ctx, req.SKU, req.RequestID)
	if err != nil {
		if strings.Contains(err.Error(), "out of stock") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(IssueResp{Error: "out of stock"})
			return
		}
		http.Error(w, fmt.Sprintf("%s: kdb err: %s", op, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(IssueResp{Code: key.Code})
}

func getEnvInt(key string, def int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return def
	}
	valInt, err := strconv.Atoi(valStr)
	if err != nil {
		return def
	}
	return valInt
}
