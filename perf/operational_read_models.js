import http from "k6/http";
import { check, sleep } from "k6";
import { Trend } from "k6/metrics";

const base = __ENV.BASE_URL || "http://localhost:8080";
const tenant = __ENV.TENANT_ID || "bank-demo";
const programPage = new Trend("program_summary_page", true);
const matterPage = new Trend("matter_summary_page", true);

export const options = {
  vus: Number(__ENV.VUS || 10),
  duration: __ENV.DURATION || "30s",
  thresholds: {
    http_req_failed: ["rate<0.01"],
    program_summary_page: ["p(95)<500"],
    matter_summary_page: ["p(95)<500"],
  },
};

export default function () {
  const programs = http.get(`${base}/api/v1/program-summaries?tenant_id=${encodeURIComponent(tenant)}&limit=50`);
  programPage.add(programs.timings.duration);
  check(programs, {
    "program summaries return 200": (response) => response.status === 200,
    "program page is bounded": (response) => {
      const body = response.json();
      return Array.isArray(body.items) && body.items.length <= 50;
    },
  });

  const matters = http.get(`${base}/api/v1/matter-summaries?tenant_id=${encodeURIComponent(tenant)}&status=OPEN&limit=50`);
  matterPage.add(matters.timings.duration);
  check(matters, {
    "matter summaries return 200": (response) => response.status === 200,
    "matter page is bounded": (response) => {
      const body = response.json();
      return Array.isArray(body.items) && body.items.length <= 50;
    },
  });
  sleep(1);
}
