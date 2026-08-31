import http from 'k6/http';
import { check, group } from 'k6';

export const options = {
  vus: 10,
  duration: '5s',
};

let orderId = '';

export function setup() {
  const url = 'http://localhost:8080/api/orders';
  const payload = JSON.stringify({ sku: 'STEAM-TOPUP-500' });
  const params = { headers: { 'Content-Type': 'application/json' } };

  const res = http.post(url, payload, params);
  const body = JSON.parse(res.body);
  return { orderId: body.order_id };
}

export default function (data) {
  const url = 'http://localhost:8080/webhook/payment';
  const payload = JSON.stringify({
    event_id: 'concurrent-race-test-event',
    order_id: data.orderId,
    status: 'paid',
    amount: 500
  });
  const params = { headers: { 'Content-Type': 'application/json' } };

  const res = http.post(url, payload, params);
  
  check(res, {
    'status is 200': (r) => r.status === 200,
  });
}
