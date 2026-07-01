/**
 * Copyright 2026 Axiolon Labs
 * SPDX-License-Identifier: Apache-2.0
 */

import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: 50,
  duration: '30s',
};

export default function () {
  // We hit the liveness endpoint (/healthz) or readiness endpoint (/readyz)
  // that we registered at the server root.
  const res = http.get('http://host.docker.internal:8080/healthz');

  check(res, {
    'status 200': (r) => r.status === 200,
    'correct response body': (r) => r.json().status === 'UP',
  });

  sleep(0.1); // add a small sleep to avoid overwhelming the server completely
}
