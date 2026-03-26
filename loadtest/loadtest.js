import http from "k6/http";
import { check } from "k6";

const BASE_URL = __ENV.BASE_URL;
if (!BASE_URL) {
  throw new Error("BASE_URL environment variable is required");
}

export const options = {
  scenarios: {
    session_uniqueness: {
      executor: "shared-iterations",
      vus: 250,
      iterations: 250,
      maxDuration: "60s",
      gracefulStop: "10s",
    },
    concurrent_execute: {
      executor: "shared-iterations",
      vus: 250,
      iterations: 250,
      maxDuration: "120s",
      gracefulStop: "10s",
      startTime: "70s",
    },
  },
  thresholds: {
    checks: ["rate==1.0"],
    http_req_failed: ["rate==0.0"],
    "http_req_duration{scenario:session_uniqueness}": ["p(95)<5000"],
    "http_req_duration{scenario:concurrent_execute}": ["p(95)<10000"],
  },
};

/**
 * Extract cookie value from Set-Cookie headers in the response.
 * @param {Object} res - k6 HTTP response.
 * @param {string} name - Cookie name to extract.
 * @returns {string} Cookie value, or empty string if not found.
 */
function getCookie(res, name) {
  const cookies = res.cookies;
  if (cookies && cookies[name] && cookies[name].length > 0) {
    return cookies[name][0].value;
  }
  return "";
}

/**
 * Create a session via POST /api/session and return cookies.
 * @returns {{runnerId: string, sessionId: string}} Session cookies.
 */
function createSession() {
  const res = http.post(`${BASE_URL}/api/session`, null, {
    redirects: 0,
  });
  check(res, {
    "POST /api/session returns 204": (r) => r.status === 204,
  });
  const runnerId = getCookie(res, "runner_id");
  const sessionId = getCookie(res, "session_id");
  check(null, {
    "runner_id cookie present": () => runnerId !== "",
    "session_id cookie present": () => sessionId !== "",
  });
  return { runnerId, sessionId };
}

/**
 * Build a Cookie header string from session cookies.
 * @param {{runnerId: string, sessionId: string}} cookies - Session cookies.
 * @returns {string} Cookie header value.
 */
function cookieHeader(cookies) {
  return `runner_id=${cookies.runnerId}; session_id=${cookies.sessionId}`;
}

/**
 * Delete a session via DELETE /api/session.
 * @param {{runnerId: string, sessionId: string}} cookies - Session cookies.
 */
function deleteSession(cookies) {
  const res = http.del(`${BASE_URL}/api/session`, null, {
    headers: { Cookie: cookieHeader(cookies) },
  });
  check(res, {
    "DELETE /api/session returns 204": (r) => r.status === 204,
  });
}

/**
 * Scenario 1: Verify that 250 concurrent session creations yield unique runner IDs.
 * Each VU creates a session, logs its runner_id for external dedup check, then cleans up.
 */
export function session_uniqueness() {
  const cookies = createSession();
  console.log(`RUNNER_ID:${cookies.runnerId}`);
  deleteSession(cookies);
}

/**
 * Scenario 2: Verify that 250 concurrent execute requests complete successfully.
 * Each VU creates a session, runs two commands via SSE, validates output, then cleans up.
 */
export function concurrent_execute() {
  const cookies = createSession();
  const cookie = cookieHeader(cookies);

  // Execute "ls" command
  const lsRes = http.post(
    `${BASE_URL}/api/execute`,
    JSON.stringify({ command: "ls" }),
    {
      headers: {
        "Content-Type": "application/json",
        Cookie: cookie,
      },
      timeout: "30s",
    },
  );
  check(lsRes, {
    "POST /api/execute (ls) returns 200": (r) => r.status === 200,
    "ls response contains complete event": (r) =>
      r.body.includes('"type":"complete"') ||
      r.body.includes('"type": "complete"'),
    "ls response contains exitCode 0": (r) =>
      r.body.includes('"exitCode":0') || r.body.includes('"exitCode": 0'),
  });

  // Execute "echo hello" command
  const echoRes = http.post(
    `${BASE_URL}/api/execute`,
    JSON.stringify({ command: "echo hello" }),
    {
      headers: {
        "Content-Type": "application/json",
        Cookie: cookie,
      },
      timeout: "30s",
    },
  );
  check(echoRes, {
    "POST /api/execute (echo) returns 200": (r) => r.status === 200,
    "echo response contains complete event": (r) =>
      r.body.includes('"type":"complete"') ||
      r.body.includes('"type": "complete"'),
    "echo response contains exitCode 0": (r) =>
      r.body.includes('"exitCode":0') || r.body.includes('"exitCode": 0'),
  });

  deleteSession(cookies);
}
