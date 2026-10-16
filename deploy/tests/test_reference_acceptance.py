"""Regression tests for the persisted reference journey acceptance receipt."""

import copy
import importlib.util
from pathlib import Path
import unittest


SCRIPT = Path(__file__).resolve().parents[1] / "scripts/verify_reference_acceptance.py"
SPEC = importlib.util.spec_from_file_location("reference_acceptance", SCRIPT)
assert SPEC and SPEC.loader
verifier = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(verifier)


def receipts():
    program = "program-1"
    journeys = [
        {"code": "NDPA_CONTINUOUS", "program_id": program},
        {"code": "REGULATORY_CHANGE", "matter_id": "matter-1"},
        {"code": "AUTHORITY_REQUEST", "matter_id": "matter-2"},
        {"code": "FINDING_REMEDIATION", "matter_id": "matter-3"},
    ]
    form = {"subject_id": program, "form_id": "form-1", "form_version": 2}
    responses = [
        {
            "id": f"response-{index}",
            "current": True,
            "subject_type": "PROGRAM",
            "subject_id": program,
            "form_template_id": form["form_id"],
            "form_template_version": form["form_version"],
            "score": {"state": "FINAL"},
        }
        for index in range(3)
    ]
    policy = {
        "policy_id": "policy-1",
        "policy_version": 1,
        "matter_id": "policy-matter-1",
        "poor_response_id": "response-poor",
        "poor_execution_state": "APPLIED",
        "same_episode_response_id": "response-poor-replay",
        "same_episode_execution_state": "REUSED",
        "same_episode_reused_matter": True,
        "exactly_once_matter": True,
    }
    initial = {
        "tenant_id": "demo-bank",
        "journeys": journeys,
        "scoring_acceptance": {**form, "responses": responses},
        "operating_form_samples": {"states": dict(verifier.EXPECTED_SAMPLE_STATES)},
        "operating_vendor_count": 5,
        "form_policy_acceptance": policy,
        "oversight_population": 8,
        "oversight_source_high_water": {
            "matters": "2026-10-16T09:00:00Z",
            "continuity_events": "2026-10-16T09:00:00Z",
        },
    }
    repeated = copy.deepcopy(initial)
    repeated["scoring_acceptance"]["responses"] = None
    repeated["scoring_acceptance"]["already_seeded"] = True
    repeated["form_policy_acceptance"]["poor_execution_state"] = "REUSED"
    return initial, repeated


class ReferenceAcceptanceTests(unittest.TestCase):
    def test_accepts_exact_source_identity_and_idempotent_replay(self):
        data = verifier.verify(*receipts())
        self.assertEqual(data["submitted_responses"], 3)
        self.assertTrue(data["policy_issue_deduplicated"])
        self.assertEqual(data["authorized_report_download"], "not_verified")

    def test_rejects_program_replacement_on_rerun(self):
        first, repeated = receipts()
        repeated["journeys"][0]["program_id"] = "different-program"
        with self.assertRaisesRegex(ValueError, "program_id changed"):
            verifier.verify(first, repeated)

    def test_rejects_wrong_subject_and_form_revision(self):
        for attribute, bad in (("subject_id", "other-program"), ("form_template_version", 3)):
            with self.subTest(attribute=attribute):
                first, repeated = receipts()
                first["scoring_acceptance"]["responses"][0][attribute] = bad
                with self.assertRaisesRegex(ValueError, "exact Program/Form"):
                    verifier.verify(first, repeated)

    def test_rejects_duplicate_or_missing_response(self):
        first, repeated = receipts()
        first["scoring_acceptance"]["responses"][1]["id"] = "response-0"
        with self.assertRaisesRegex(ValueError, "distinct"):
            verifier.verify(first, repeated)

    def test_rejects_missing_or_rewritten_form_sample(self):
        first, repeated = receipts()
        repeated["operating_form_samples"]["states"]["sentinel-risk-review"] = "SUBMITTED"
        with self.assertRaisesRegex(ValueError, "source state|state changed"):
            verifier.verify(first, repeated)

    def test_rejects_duplicate_policy_issue_on_rerun(self):
        first, repeated = receipts()
        repeated["form_policy_acceptance"]["matter_id"] = "another-issue"
        with self.assertRaisesRegex(ValueError, "matter_id changed"):
            verifier.verify(first, repeated)

    def test_rejects_missing_source_high_water(self):
        first, repeated = receipts()
        first["oversight_source_high_water"].pop("continuity_events")
        with self.assertRaisesRegex(ValueError, "high-water"):
            verifier.verify(first, repeated)


if __name__ == "__main__":
    unittest.main()
