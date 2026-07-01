/**
 * Copyright 2026 Axiolon Labs
 * SPDX-License-Identifier: Apache-2.0
 */

export const PROFILES = {
  smoke: {
    executor: 'constant-vus',
    vus: 1,
    duration: '10s',
  },
  load: {
    executor: 'ramping-vus',
    startVUs: 0,
    stages: [
      { duration: '30s', target: 20 }, // ramp up to 20 users
      { duration: '1m', target: 20 },  // stay at 20 users
      { duration: '30s', target: 50 }, // ramp up to 50 users
      { duration: '1m', target: 50 },  // stay at 50 users
      { duration: '30s', target: 0 },  // ramp down to 0
    ],
  },
  stress: {
    executor: 'ramping-vus',
    startVUs: 0,
    stages: [
      { duration: '30s', target: 50 },
      { duration: '1m', target: 50 },
      { duration: '30s', target: 100 },
      { duration: '1m', target: 100 },
      { duration: '30s', target: 150 },
      { duration: '1m', target: 150 },
      { duration: '30s', target: 0 },
    ],
  },
  spike: {
    executor: 'ramping-vus',
    startVUs: 0,
    stages: [
      { duration: '10s', target: 20 },  // baseline
      { duration: '10s', target: 200 }, // sudden spike to 200
      { duration: '30s', target: 200 }, // hold at 200
      { duration: '10s', target: 20 },  // scale back down
      { duration: '10s', target: 0 },
    ],
  },
  soak: {
    executor: 'ramping-vus',
    startVUs: 0,
    stages: [
      { duration: '30s', target: 30 },
      { duration: '30m', target: 30 },  // continuous load
      { duration: '30s', target: 0 },
    ],
  },
};

export const THRESHOLDS = {
  http_req_failed: ['rate<0.01'], // error rate should be less than 1%
  http_req_duration: ['p(95)<350'], // 95% of requests should be below 350ms
};

export function getOptions(profileName) {
  const profile = PROFILES[profileName];
  if (!profile) {
    const valid = Object.keys(PROFILES).join(', ');
    throw new Error(`Unknown profile "${profileName}". Valid profiles: ${valid}`);
  }
  
  const baseOptions = {
    thresholds: THRESHOLDS,
  };

  if (profile.executor === 'constant-vus') {
    return Object.assign({}, baseOptions, {
      vus: profile.vus,
      duration: profile.duration,
    });
  } else {
    return Object.assign({}, baseOptions, {
      stages: profile.stages,
    });
  }
}
