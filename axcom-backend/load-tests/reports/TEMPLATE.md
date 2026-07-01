# Load Test Run Report Template

Use this template to document the results of individual load testing runs. Copy this file to a new file named `load-test-YYYY-MM-DD-<profile>.md` in this directory.

---

## 1. Run Metadata

| Parameter | Value / Details |
| :--- | :--- |
| **Date & Time** | YYYY-MM-DD HH:MM UTC/Local |
| **Tested By** | [Name/Operator] |
| **Commit/Branch** | `main` / `commit-hash` |
| **Target URL** | `http://localhost:8080` (or staging URL) |
| **Performance Profile** | `smoke` / `load` / `stress` / `spike` / `soak` |
| **Scenario Override** | *None* / `browsing` / `cart` / `checkout` |

---

## 2. Environment Configuration

*   **Database**: MongoDB Replica Set single node (`rs0`)
*   **Rate Limit Middleware**: Bypassed (`SKIP_RATE_LIMIT=true`) / Enabled
*   **Hardware/Environment Specs**: e.g., local machine (CPU, RAM, OS) or cloud instances

---

## 3. Results Summary

### SLO Assertions

| Metric | Threshold | Actual Value | Pass/Fail |
| :--- | :--- | :--- | :--- |
| **http_req_failed** | `< 0.01` (< 1% errors) | `X.XX%` | `PASS` / `FAIL` |
| **http_req_duration (p95)** | `< 350ms` | `XXX.XXms` | `PASS` / `FAIL` |

### Key Performance Indicators (KPIs)

| Metric | Min | Average | p90 | p95 | Max |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **http_req_duration** | `Xms` | `Xms` | `Xms` | `Xms` | `Xms` |
| **http_req_connecting** | `Xms` | `Xms` | `Xms` | `Xms` | `Xms` |
| **http_req_waiting (TTFB)**| `Xms` | `Xms` | `Xms` | `Xms` | `Xms` |

*   **VUs (Virtual Users)**: Peak: `XXX` VUs
*   **Total Requests**: `XXXX` requests
*   **Request Rate**: `XXX.XX/s`
*   **Data Sent/Received**: `X.X MB` sent / `X.X MB` received

---

## 4. Scenario Breakdown (Checks Summary)

Record the success rates of the key assertions executed during this test run:

| Check Name | Target Scenario | Success Rate | Pass/Fail |
| :--- | :--- | :--- | :--- |
| `healthz status is 200` | Browsing | `XX.XX%` | `PASS` / `FAIL` |
| `category detail status is 200` | Browsing | `XX.XX%` | `PASS` / `FAIL` |
| `products list status is 200` | Browsing | `XX.XX%` | `PASS` / `FAIL` |
| `cart user register status is 200` | Cart / Checkout | `XX.XX%` | `PASS` / `FAIL` |
| `add to cart status is 200` | Cart | `XX.XX%` | `PASS` / `FAIL` |
| `clear cart status is 200` | Cart | `XX.XX%` | `PASS` / `FAIL` |
| `guest checkout status is 200` | Checkout | `XX.XX%` | `PASS` / `FAIL` |
| `customer checkout status is 200` | Checkout | `XX.XX%` | `PASS` / `FAIL` |
| `token refresh status is 200` | Checkout | `XX.XX%` | `PASS` / `FAIL` |
| `cancel order status is 200` | Checkout | `XX.XX%` | `PASS` / `FAIL` |
| `transition to processing status is 200` | Checkout (Admin) | `XX.XX%` | `PASS` / `FAIL` |

---

## 5. Notes & Observations

*   Describe any performance degradation, latency spikes, database locks, or unexpected transaction bottlenecks observed.
*   Identify CPU or memory spikes in the ecom-backend or MongoDB containers.
*   Note down any actions needed to optimize performance or fix bottlenecks.
