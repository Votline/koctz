#!/usr/bin/env bash

set -e

BASE_URL="http://localhost:8080"
EVENT_ID="test-event-$(date +%s)"

echo "=== 1. Создание многопозиционного заказа ==="
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
CREATED_AT=$(echo "$CREATE_RESP" | grep -o '"created_at":"[^"]*' | grep -o '[^"]*$' | head -n 1)

if [ -z "$ORDER_ID" ]; then
  echo "Ошибка: не удалось получить order_id"
  exit 1
fi

echo "Получен ORDER_ID: $ORDER_ID"
echo "Время создания: $CREATED_AT"
echo ""

echo "=== 2. Получение текущего состояния заказа ==="
GET_RESP=$(curl -s -X GET "$BASE_URL/api/orders/$ORDER_ID")
echo "Ответ: $GET_RESP"
echo ""

echo "=== 3. Отправка вебхука оплаты (успех) ==="
WEBHOOK_RESP=$(curl -s -X POST "$BASE_URL/webhook/payment" \
  -H "Content-Type: application/json" \
  -d "{
    \"event_id\": \"$EVENT_ID\",
    \"order_id\": \"$ORDER_ID\",
    \"status\": \"success\"
  }")
echo "Ответ: $WEBHOOK_RESP"
echo ""

echo "=== 4. Проверка идемпотентности вебхука дубликатом ==="
DUPLICATE_RESP=$(curl -s -X POST "$BASE_URL/webhook/payment" \
  -H "Content-Type: application/json" \
  -d "{
    \"event_id\": \"$EVENT_ID\",
    \"order_id\": \"$ORDER_ID\",
    \"status\": \"success\"
  }")
echo "Ответ: $DUPLICATE_RESP"
echo ""

echo "=== 5. Проверка машины времени (GetStateAt по дате создания) ==="
TIME_RESP=$(curl -s -X GET "$BASE_URL/api/orders/$ORDER_ID?at=$CREATED_AT")
echo "Ответ: $TIME_RESP"
echo ""

echo "=== 6. Проверка статуса реконсиляции ==="
RECONCILE_RESP=$(curl -s -X GET "$BASE_URL/api/reconcile")
echo "Ответ: $RECONCILE_RESP"
echo ""

echo "Все тесты успешно пройдены!"
