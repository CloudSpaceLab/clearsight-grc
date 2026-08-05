import http from "k6/http";
import { check, sleep } from "k6";

export const options = {
  scenarios: { common_reads: { executor: "constant-vus", vus: 25, duration: "30s" } },
  thresholds: {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(95)<1500"],
    "http_req_duration{endpoint:authority}": ["p(95)<150"]
  }
};

const base = __ENV.BASE_URL || "http://localhost:8080";

export default function () {
  const today = http.get(`${base}/api/v1/today`, { tags: { endpoint: "today" } });
  check(today, { "today 200": (response) => response.status === 200 });
  const authority = http.post(`${base}/api/v1/authority/resolve`, JSON.stringify({ tenant_id: "bank-demo", legal_entity_id: "bank-ng", object_type: "MATTER", object_id: "matter-load-test", responsibility: "AUTHORIZER", materiality: 5 }), { headers: { "Content-Type": "application/json" }, tags: { endpoint: "authority" } });
  check(authority, { "authority 200": (response) => response.status === 200 });
  sleep(0.2);
}
