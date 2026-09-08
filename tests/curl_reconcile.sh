#!/usr/bin/env bash

set -e

BASE_URL="http://localhost:8080"

echo "=== Запрос финансовой сверки (Reconciliation) ==="
RECON_RESP=$(curl -s -X GET "$BASE_URL/api/reconcile")
echo "Ответ эндпоинта реконсиляции:"
echo "$RECON_RESP"
