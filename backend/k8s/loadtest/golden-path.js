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

// A real client logs in once and reuses its token, so by default a VU logs in
// every LOGIN_EVERY iterations (default 20). LOGIN_EVERY=1 logs in on every
// iteration: a deliberate worst case, because login is bcrypt-bound (~150ms
// of CPU each on a laptop VM) — at ~16 logins/s it alone needs ~2.3 cores
// and starves everything else on a 4-CPU VM, which measures the machine
// rather than the deployment.
const LOGIN_EVERY = parseInt(__ENV.LOGIN_EVERY || '20', 10);
let token = null;

export default function () {
  ensureAccount();

  if (!token || __ITER % LOGIN_EVERY === 0) {
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
    token = login.json('data.access_token');
  }
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
