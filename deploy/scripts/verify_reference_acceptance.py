#!/usr/bin/env python3
"""Verify persisted, explicitly non-production reference journey receipts.

The checks certify the seeded Program/Form/response-policy/Issue projection only.
They do not certify live actor authorization, a report download, or customer UAT.
"""

import argparse
import json
from pathlib import Path


EXPECTED_SAMPLE_STATES = {
    "paywave-controls": "SUBMITTED",
    "archiveguard-controls": "SUBMITTED",
    "peoplelink-due-diligence": "IN_PROGRESS",
    "sentinel-risk-review": "READY",
    "cloudspace-oem-risk-register": "SUBMITTED",
}


def require(condition: bool, message: str) -> None:
    if not condition:
        raise ValueError(message)


def verify(first: dict, repeated: dict) -> dict:
    require(first.get("tenant_id") == repeated.get("tenant_id") and first.get("tenant_id"), "tenant changed between sample runs")
    journeys = {item["code"]: item for item in first.get("journeys", [])}
    programs = journeys.get("NDPA_CONTINUOUS", {})
    program_id = programs.get("program_id")
    require(bool(program_id), "canonical Program journey is missing")
    for code in ("REGULATORY_CHANGE", "AUTHORITY_REQUEST", "FINDING_REMEDIATION"):
        require(bool(journeys.get(code, {}).get("matter_id")), f"{code} Matter is missing")

    repeat_journeys = {item["code"]: item for item in repeated.get("journeys", [])}
    for code, item in journeys.items():
        require(code in repeat_journeys, f"{code} disappeared on rerun")
        for key in ("program_id", "matter_id", "evidence_request_id"):
            require(item.get(key) == repeat_journeys[code].get(key), f"{code} {key} changed on rerun")

    scoring = first["scoring_acceptance"]
    replay_scoring = repeated["scoring_acceptance"]
    require(scoring.get("subject_id") == program_id, "scored Form does not belong to the canonical Program")
    require(bool(scoring.get("form_id")) and scoring.get("form_version", 0) >= 1, "current Form revision missing")
    responses = scoring.get("responses") or []
    require(len(responses) == 3, "first run must submit all three assessed test responses")
    response_ids = [response.get("id") for response in responses]
    require(all(response_ids) and len(set(response_ids)) == 3, "response revisions must be distinct")
    response_scopes = {(response.get("tenant_id"), response.get("legal_entity_id")) for response in responses}
    require(len(response_scopes) == 1 and all(next(iter(response_scopes))), "response revisions cross tenant/legal-entity scope")
    for response in responses:
        require(
            response.get("subject_type") == "PROGRAM"
            and response.get("subject_id") == program_id
            and response.get("form_template_id") == scoring["form_id"]
            and response.get("form_template_version") == scoring["form_version"]
            and response.get("current") is True
            and response.get("score") is not None,
            "completed response lacks its exact Program/Form revision or scoring source",
        )
    for field in ("subject_id", "form_id", "form_version"):
        require(scoring[field] == replay_scoring.get(field), f"scored Form {field} changed on rerun")
    require(replay_scoring.get("already_seeded") is True, "second sample run did not recognize existing scored responses")

    first_samples = first["operating_form_samples"]["states"]
    second_samples = repeated["operating_form_samples"]["states"]
    for key, state in EXPECTED_SAMPLE_STATES.items():
        require(first_samples.get(key) == state, f"{key} source state is not {state}")
        require(second_samples.get(key) == state, f"{key} state changed on rerun")
    require(first_samples == second_samples, "operating Form state population changed on rerun")
    require(first.get("operating_vendor_count", 0) > 0, "no vendor relationship population was installed")

    first_policy = first["form_policy_acceptance"]
    second_policy = repeated["form_policy_acceptance"]
    for record in (first_policy, second_policy):
        require(record.get("exactly_once_matter") is True, "response policy did not prove exactly-once Issue execution")
        require(record.get("same_episode_reused_matter") is True, "second scored response created an extra Issue")
        require(bool(record.get("matter_id")) and bool(record.get("policy_id")), "Issue/policy record identity missing")
        require(record.get("poor_execution_state") in ("APPLIED", "REUSED"), "poor response did not create or reuse the Issue")
        require(record.get("same_episode_execution_state") == "REUSED", "same-episode response was not deduplicated")
        require(
            bool(record.get("same_episode_response_id"))
            and record["same_episode_response_id"] != record.get("poor_response_id"),
            "distinct response revisions were not retained",
        )
    for field in ("policy_id", "policy_version", "matter_id", "poor_response_id"):
        require(first_policy[field] == second_policy.get(field), f"response-policy {field} changed on rerun")

    for receipt in (first, repeated):
        require(receipt.get("oversight_population", 0) > 0, "oversight population is empty")
        high_water = receipt.get("oversight_source_high_water") or {}
        require(bool(high_water.get("matters")) and bool(high_water.get("continuity_events")), "oversight source high-water is missing")

    return {
        "programs": 1,
        "related_matters": 3,
        "submitted_responses": len(responses),
        "operating_form_samples": len(first_samples),
        "policy_issue_deduplicated": True,
        "source_replay_stable": True,
        "hosted_actor_approval": "not_verified",
        "authorized_report_download": "not_verified",
    }


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("first", type=Path)
    parser.add_argument("repeated", type=Path)
    args = parser.parse_args()
    with args.first.open(encoding="utf-8") as source:
        initial = json.load(source)
    with args.repeated.open(encoding="utf-8") as source:
        replay = json.load(source)
    print(json.dumps(verify(initial, replay), sort_keys=True))


if __name__ == "__main__":
    main()
