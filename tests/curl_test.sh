#!/usr/bin/env bash

set -e

BASE_URL="http://localhost:8080"
SKU="STEAM-TOPUP-500"
EVENT_ID="test-event-$(date +%s)"

echo "=== 1. Создание заказа ==="
CREATE_RESP=$(curl -s -X POST "$BASE_URL/api/orders" \
  -H "Content-Type: application/json" \
  -d "{\"sku\": \"$SKU\"}")

echo "Ответ: $CREATE_RESP"

ORDER_ID=$(echo "$CREATE_RESP" | grep -o '"order_id":"[^"]*' | grep -o '[^"]*$')

if [ -z "$ORDER_ID" ]; then
  echo "Ошибка: не удалось получить order_id"
  exit 1
fi

echo "Получен ORDER_ID: $ORDER_ID"
echo ""

echo "=== 2. Получение информации о заказе ==="
GET_RESP=$(curl -s -X GET "$BASE_URL/api/orders/$ORDER_ID")
echo "Ответ: $GET_RESP"
echo ""

echo "=== 3. Отправка вебхука оплаты ==="
WEBHOOK_RESP=$(curl -s -X POST "$BASE_URL/webhook/payment" \
  -H "Content-Type: application/json" \
  -d "{
    \"event_id\": \"$EVENT_ID\",
    \"order_id\": \"$ORDER_ID\",
    \"status\": \"paid\",
    \"amount\": 500
  }")
echo "Ответ: $WEBHOOK_RESP"
echo ""

echo "=== 4. Проверка идемпотентности вебхука дубликатом ==="
DUPLICATE_RESP=$(curl -s -X POST "$BASE_URL/webhook/payment" \
  -H "Content-Type: application/json" \
  -d "{
    \"event_id\": \"$EVENT_ID\",
    \"order_id\": \"$ORDER_ID\",
    \"status\": \"paid\",
    \"amount\": 500
  }")
echo "Ответ: $DUPLICATE_RESP"
echo ""

echo "=== 5. Проверка статуса сверки ==="
RECONCILE_RESP=$(curl -s -X GET "$BASE_URL/api/reconcile")
echo "Ответ: $RECONCILE_RESP"
echo ""
