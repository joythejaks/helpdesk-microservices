// Golden-path load test for the Kubernetes deployment.
//
//   LB=$(kubectl -n helpdesk get svc caddy -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
//   docker run --rm -i --network kind -e LB_IP=$LB grafana/k6 run --insecure-skip-tls-verify - < golden-path.js
//
// It runs on the cluster's Docker network and targets the Caddy LoadBalancer
// so the whole path is exercised: Caddy -> gateway -> auth / ticket ->
// Postgres/RabbitMQ. Two workarounds for a local setup:
//  - Caddy only has a certificate for "localhost" (local_certs), so requests
//    must keep that name (SNI) — `hosts` below maps localhost to the LB IP
//    instead of using the IP or host.docker.internal, which Caddy rejects.
//  - --insecure-skip-tls-verify, because Caddy's local CA isn't trusted here.
//
// The per-IP rate limits (gateway 10 rps, auth 5 rps by default) would answer
// 429 to a single load-generating source and mask any scaling, so raise
// *_RATE_LIMIT_RPS/BURST in the app ConfigMaps for the run (and restart the
// deployments), then re-apply the manifests to restore the defaults.
import http from 'k6/http';
import { check, sleep } from 'k6';

const BASE = 'https://localhost';

export const options = {
  hosts: { localhost: __ENV.LB_IP },
  stages: [
    { duration: '30s', target: 20 },
    { duration: '90s', target: 60 },
    { duration: '60s', target: 60 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_failed: ['rate<0.05'],
    http_req_duration: ['p(95)<2000'],
  },
};

const JSON_HEADERS = { 'Content-Type': 'application/json' };

// One account per VU, created lazily on that VU's first iteration.
let account = null;

function ensureAccount() {
  if (account) return;
  const email = `k6-${__VU}-${Date.now()}@example.com`;
  const password = 'k6-password-123';
  const reg = http.post(
    `${BASE}/auth/register`,
    JSON.stringify({ name: `k6 ${__VU}`, email, password, department: 'IT' }),
    { headers: JSON_HEADERS },
  );
  check(reg, { 'register 200': (r) => r.status === 200 });
  account = { email, password };
}

export default function () {
  ensureAccount();

  // Login is bcrypt-bound, so it's what actually burns CPU in auth-service.
  const login = http.post(
    `${BASE}/auth/login`,
    JSON.stringify(account),
    { headers: JSON_HEADERS },
  );
  const ok = check(login, { 'login 200': (r) => r.status === 200 });
  if (!ok) {
    sleep(1);
    return;
  }
  const token = login.json('data.access_token');
  const auth = { headers: { ...JSON_HEADERS, Authorization: `Bearer ${token}` } };

  const created = http.post(
    `${BASE}/tickets`,
    JSON.stringify({
      title: `k6 ticket ${__VU}-${__ITER}`,
      description: 'load test',
      priority: 'Low',
      department: 'IT',
    }),
    auth,
  );
  check(created, { 'create ticket 200': (r) => r.status === 200 });

  const list = http.get(`${BASE}/tickets`, auth);
  check(list, { 'list tickets 200': (r) => r.status === 200 });

  sleep(0.5);
}
