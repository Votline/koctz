#!/usr/bin/env bash

set -e

BASE_URL="http://localhost:8080"
EVENT_ID="partial-evt-$(date +%s)"

echo "=== 1. Создание заказа с несколькими товарами ==="
CREATE_RESP=$(curl -s -X POST "$BASE_URL/api/orders" \
  -H "Content-Type: application/json" \
  -d '{
    "items": [
      {"sku": "STEAM-TOPUP-500", "price": 500.0},
      {"sku": "STEAM-TOPUP-100", "price": 100.0}
    ]
  }')
echo "Ответ: $CREATE_RESP"

ORDER_ID=$(echo "$CREATE_RESP" | grep -o '"order_id":"[^"]*' | grep -o '[^"]*$')
if [ -z "$ORDER_ID" ]; then
  echo "Ошибка: не удалось получить order_id"
  exit 1
fi
echo "Создан заказ: $ORDER_ID"
echo ""

echo "=== 2. Симуляция оплаты и запроса к поставщикам (возможен частичный сбой) ==="
WEBHOOK_RESP=$(curl -s -X POST "$BASE_URL/webhook/payment" \
  -H "Content-Type: application/json" \
  -d "{
    \"event_id\": \"$EVENT_ID\",
    \"order_id\": \"$ORDER_ID\",
    \"status\": \"success\"
  }")
echo "Ответ вебхука: $WEBHOOK_RESP"
echo ""

echo "=== 3. Проверка идемпотентности (повторный вебхук с тем же event_id) ==="
DUP_RESP=$(curl -s -X POST "$BASE_URL/webhook/payment" \
  -H "Content-Type: application/json" \
  -d "{
    \"event_id\": \"$EVENT_ID\",
    \"order_id\": \"$ORDER_ID\",
    \"status\": \"success\"
  }")
echo "Ответ дубликата: $DUP_RESP"
echo ""

echo "=== 4. Ожидание фоновой выдачи и проверка статуса заказа ==="
sleep 2
FINAL_RESP=$(curl -s -X GET "$BASE_URL/api/orders/$ORDER_ID")
echo "Итоговое состояние заказа (часть выдана, часть в возврате/ошибке):"
echo "$FINAL_RESP"
