import http from 'k6/http';
import { check } from 'k6';

export const options = {
  vus: 10,
  duration: '10s',
};

export default function () {
  const orderRes = http.post('http://localhost:8080/api/orders', JSON.stringify({
    items: [{ sku: 'STEAM-TOPUP-500', price: 500.0 }]
  }), { headers: { 'Content-Type': 'application/json' } });

  if (orderRes.status !== 201) return;
  const orderId = JSON.parse(orderRes.body).order_id;

  const eventId = `${Date.now()}-${__VU}-${__ITER}`;

  const webhookRes = http.post('http://localhost:8080/webhook/payment', JSON.stringify({
    event_id: eventId,
    order_id: orderId,
    status: 'success'
  }), { headers: { 'Content-Type': 'application/json' } });

  check(webhookRes, {
    'status is 200': (r) => r.status === 200,
  });
}
