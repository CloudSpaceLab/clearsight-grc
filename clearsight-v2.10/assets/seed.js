/* Synthetic demonstration data. No customer or licensed regulatory content. */
window.CS_SEED = {
  "schemaVersion": 3,
  "asOf": "2026-11-13",
  "organization": "Meridian Bank",
  "requirements": [
    {
      "id": "R01",
      "code": "BA-01",
      "version": "1.0",
      "group": "Governance",
      "title": "Assign accountable ownership",
      "source": "Assign an accountable owner and review responsibilities for the assurance program.",
      "questions": [
        "Q01",
        "Q02"
      ]
    },
    {
      "id": "R02",
      "code": "BA-02",
      "version": "1.0",
      "group": "Governance",
      "title": "Maintain approved policies",
      "source": "Keep relevant policies approved, versioned and available to the people who need them.",
      "questions": [
        "Q03",
        "Q04"
      ]
    },
    {
      "id": "R03",
      "code": "BA-03",
      "version": "1.0",
      "group": "Access management",
      "title": "Protect privileged access",
      "source": "Protect privileged accounts and review their access using appropriate evidence.",
      "questions": [
        "Q05",
        "Q06"
      ]
    },
    {
      "id": "R04",
      "code": "BA-04",
      "version": "1.0",
      "group": "Access management",
      "title": "Review access rights",
      "source": "Review access rights and remove access that is no longer justified.",
      "questions": [
        "Q06",
        "Q07"
      ]
    },
    {
      "id": "R05",
      "code": "BA-05",
      "version": "1.0",
      "group": "Data protection",
      "title": "Protect sensitive information",
      "source": "Protect sensitive information at rest and in transit within the agreed service scope.",
      "questions": [
        "Q08",
        "Q09"
      ]
    },
    {
      "id": "R06",
      "code": "BA-06",
      "version": "1.0",
      "group": "Resilience",
      "title": "Test service recovery",
      "source": "Test recovery procedures against documented service objectives and record the results.",
      "questions": [
        "Q10",
        "Q11"
      ]
    },
    {
      "id": "R07",
      "code": "BA-07",
      "version": "1.0",
      "group": "Third-party oversight",
      "title": "Oversee outsourced services",
      "source": "Review critical service relationships, assurance evidence and contractual responsibilities.",
      "questions": [
        "Q12",
        "Q13"
      ]
    },
    {
      "id": "R08",
      "code": "BA-08",
      "version": "1.0",
      "group": "Incident management",
      "title": "Manage and learn from incidents",
      "source": "Record significant incidents, assign corrective actions and test escalation arrangements.",
      "questions": [
        "Q14",
        "Q15"
      ]
    }
  ],
  "questions": [
    {
      "id": "Q01",
      "title": "Who is accountable for this program?",
      "type": "text",
      "freshDays": 365,
      "evidence": false,
      "connector": null
    },
    {
      "id": "Q02",
      "title": "When were responsibilities last reviewed?",
      "type": "date",
      "freshDays": 90,
      "evidence": true,
      "connector": null
    },
    {
      "id": "Q03",
      "title": "Is the applicable policy approved?",
      "type": "choice",
      "freshDays": 365,
      "evidence": true,
      "connector": null
    },
    {
      "id": "Q04",
      "title": "What is the effective policy version?",
      "type": "text",
      "freshDays": 365,
      "evidence": false,
      "connector": null
    },
    {
      "id": "Q05",
      "title": "Is MFA enforced for every privileged account?",
      "type": "choice",
      "freshDays": 30,
      "evidence": true,
      "connector": "Identity fixture"
    },
    {
      "id": "Q06",
      "title": "Who owns privileged-access reviews?",
      "type": "text",
      "freshDays": 90,
      "evidence": false,
      "connector": null
    },
    {
      "id": "Q07",
      "title": "When was the last access review completed?",
      "type": "date",
      "freshDays": 90,
      "evidence": true,
      "connector": null
    },
    {
      "id": "Q08",
      "title": "Is encryption at rest enabled in the selected service?",
      "type": "choice",
      "freshDays": 30,
      "evidence": true,
      "connector": "Cloud fixture"
    },
    {
      "id": "Q09",
      "title": "Is encryption in transit enforced?",
      "type": "choice",
      "freshDays": 30,
      "evidence": true,
      "connector": null
    },
    {
      "id": "Q10",
      "title": "When was recovery last tested?",
      "type": "date",
      "freshDays": 180,
      "evidence": true,
      "connector": null
    },
    {
      "id": "Q11",
      "title": "Did the test meet the documented recovery objective?",
      "type": "choice",
      "freshDays": 180,
      "evidence": true,
      "connector": null
    },
    {
      "id": "Q12",
      "title": "Who owns the vendor service relationship?",
      "type": "text",
      "freshDays": 90,
      "evidence": false,
      "connector": null
    },
    {
      "id": "Q13",
      "title": "Is current service-specific assurance available?",
      "type": "choice",
      "freshDays": 90,
      "evidence": true,
      "connector": null
    },
    {
      "id": "Q14",
      "title": "Are incident escalation routes documented and tested?",
      "type": "choice",
      "freshDays": 180,
      "evidence": true,
      "connector": null
    },
    {
      "id": "Q15",
      "title": "How many corrective actions remain overdue?",
      "type": "number",
      "freshDays": 30,
      "evidence": false,
      "connector": null
    }
  ],
  "frameworks": [
    {
      "id": "FW-BANK",
      "name": "Banking assurance baseline",
      "short": "BA",
      "version": "1.0",
      "type": "Internal baseline",
      "description": "Governance, access, data protection and operational resilience.",
      "reqs": [
        "R01",
        "R02",
        "R03",
        "R04",
        "R05",
        "R06",
        "R07",
        "R08"
      ],
      "color": "blue"
    },
    {
      "id": "FW-VENDOR",
      "name": "Third-party assurance",
      "short": "TP",
      "version": "1.0",
      "type": "Contractual assurance",
      "description": "Service-specific checks for outsourced technology and data services.",
      "reqs": [
        "R03",
        "R04",
        "R05",
        "R06",
        "R07"
      ],
      "color": "teal"
    },
    {
      "id": "FW-RES",
      "name": "Operational resilience",
      "short": "OR",
      "version": "1.0",
      "type": "Internal baseline",
      "description": "Recovery testing, incident handling and ownership.",
      "reqs": [
        "R01",
        "R06",
        "R08"
      ],
      "color": "violet"
    }
  ],
  "units": [
    "Technology",
    "Payments",
    "Retail Operations",
    "Information Security",
    "Compliance",
    "Vendor Management"
  ],
  "people": [
    "Amina Bello",
    "Chidi Okafor",
    "Ifeoma Nwosu",
    "Daniel Mensah",
    "Sarah Adams"
  ],
  "programs": [
    {
      "id": "PRG-104",
      "name": "Bank assurance · 2026",
      "frameworkId": "FW-BANK",
      "frameworkVersion": "1.0",
      "scopeType": "bank",
      "targets": [
        "Meridian Bank"
      ],
      "owner": "Amina Bello",
      "reviewer": "Ifeoma Nwosu",
      "respondents": [
        "Information Security",
        "Technology",
        "Compliance"
      ],
      "due": "2026-11-30",
      "cadence": "Quarterly",
      "reqs": [
        "R01",
        "R02",
        "R03",
        "R04",
        "R05",
        "R06",
        "R07",
        "R08"
      ],
      "excluded": {},
      "overrides": {
        "R03": "Require MFA for every privileged account and recheck each month."
      },
      "rules": {},
      "assignments": {
        "R03": [
          "Information Security"
        ],
        "R06": [
          "Technology"
        ]
      },
      "evaluations": {
        "Meridian Bank": {
          "R01": {
            "status": "Satisfied",
            "reviewer": "Ifeoma Nwosu",
            "date": "2026-11-10",
            "note": "Illustrative accepted evaluation; not regulatory evidence."
          },
          "R02": {
            "status": "Satisfied",
            "reviewer": "Ifeoma Nwosu",
            "date": "2026-11-10",
            "note": "Illustrative accepted evaluation; not regulatory evidence."
          },
          "R03": {
            "status": "Partially satisfied",
            "reviewer": "Ifeoma Nwosu",
            "date": "2026-11-10",
            "note": "Illustrative accepted evaluation; not regulatory evidence."
          },
          "R04": {
            "status": "Needs refresh",
            "reviewer": "Ifeoma Nwosu",
            "date": "2026-11-10",
            "note": "Illustrative accepted evaluation; not regulatory evidence."
          },
          "R05": {
            "status": "Satisfied",
            "reviewer": "Ifeoma Nwosu",
            "date": "2026-11-10",
            "note": "Illustrative accepted evaluation; not regulatory evidence."
          },
          "R06": {
            "status": "Not satisfied",
            "reviewer": "Ifeoma Nwosu",
            "date": "2026-11-10",
            "note": "Illustrative accepted evaluation; not regulatory evidence."
          },
          "R07": {
            "status": "Not assessed",
            "reviewer": "Ifeoma Nwosu",
            "date": "2026-11-10",
            "note": "Illustrative accepted evaluation; not regulatory evidence."
          },
          "R08": {
            "status": "Satisfied",
            "reviewer": "Ifeoma Nwosu",
            "date": "2026-11-10",
            "note": "Illustrative accepted evaluation; not regulatory evidence."
          }
        }
      },
      "created": "2026-10-01",
      "activity": []
    },
    {
      "id": "PRG-105",
      "name": "Atlas hosting assurance",
      "frameworkId": "FW-VENDOR",
      "frameworkVersion": "1.0",
      "scopeType": "vendor",
      "targets": [
        "ENG-01"
      ],
      "owner": "Sarah Adams",
      "reviewer": "Ifeoma Nwosu",
      "respondents": [
        "Vendor Management",
        "Information Security"
      ],
      "due": "2026-11-25",
      "cadence": "Quarterly",
      "reqs": [
        "R03",
        "R04",
        "R05",
        "R06",
        "R07"
      ],
      "excluded": {},
      "overrides": {},
      "rules": {},
      "assignments": {},
      "evaluations": {
        "ENG-01": {
          "R03": {
            "status": "Satisfied",
            "date": "2026-11-10",
            "reviewer": "Ifeoma Nwosu"
          },
          "R05": {
            "status": "Partially satisfied",
            "date": "2026-11-10",
            "reviewer": "Ifeoma Nwosu"
          }
        }
      },
      "created": "2026-10-10",
      "activity": []
    }
  ],
  "vendors": [
    {
      "id": "VEN-01",
      "name": "Atlas Cloud Services",
      "country": "Nigeria",
      "contact": "assurance@atlas.example"
    },
    {
      "id": "VEN-02",
      "name": "Verity Payments",
      "country": "South Africa",
      "contact": "security@verity.example"
    },
    {
      "id": "VEN-03",
      "name": "Cedar Data Services",
      "country": "Nigeria",
      "contact": "risk@cedar.example"
    },
    {
      "id": "VEN-04",
      "name": "Northstar Facilities",
      "country": "Nigeria",
      "contact": "ops@northstar.example"
    }
  ],
  "engagements": [
    {
      "id": "ENG-01",
      "vendorId": "VEN-01",
      "service": "Core banking hosting",
      "unit": "Technology",
      "owner": "Sarah Adams",
      "criticality": "Critical",
      "status": "Due diligence",
      "data": "Customer financial data",
      "renewal": "2027-03-31"
    },
    {
      "id": "ENG-02",
      "vendorId": "VEN-01",
      "service": "Analytics sandbox",
      "unit": "Technology",
      "owner": "Chidi Okafor",
      "criticality": "Medium",
      "status": "Active",
      "data": "De-identified test data",
      "renewal": "2027-06-30"
    },
    {
      "id": "ENG-03",
      "vendorId": "VEN-02",
      "service": "Payment switching",
      "unit": "Payments",
      "owner": "Daniel Mensah",
      "criticality": "Critical",
      "status": "Active",
      "data": "Transaction data",
      "renewal": "2027-01-31"
    },
    {
      "id": "ENG-04",
      "vendorId": "VEN-03",
      "service": "Document processing",
      "unit": "Retail Operations",
      "owner": "Sarah Adams",
      "criticality": "High",
      "status": "Remediation",
      "data": "Customer identity documents",
      "renewal": "2027-04-30"
    },
    {
      "id": "ENG-05",
      "vendorId": "VEN-04",
      "service": "Branch facilities",
      "unit": "Retail Operations",
      "owner": "Daniel Mensah",
      "criticality": "Low",
      "status": "Onboarding",
      "data": "Staff contact details",
      "renewal": "2027-07-31"
    }
  ],
  "controls": [
    {
      "id": "CTL-01",
      "title": "Privileged-access governance",
      "owner": "Chidi Okafor",
      "unit": "Information Security",
      "effectiveness": "Partially effective",
      "reqs": [
        "R03",
        "R04"
      ],
      "risks": [
        "RSK-01"
      ]
    },
    {
      "id": "CTL-02",
      "title": "Service recovery exercises",
      "owner": "Daniel Mensah",
      "unit": "Technology",
      "effectiveness": "Ineffective",
      "reqs": [
        "R06"
      ],
      "risks": [
        "RSK-02",
        "RSK-03"
      ]
    },
    {
      "id": "CTL-03",
      "title": "Dual payment authorization",
      "owner": "Amina Bello",
      "unit": "Payments",
      "effectiveness": "Effective",
      "reqs": [
        "R03"
      ],
      "risks": [
        "RSK-04"
      ]
    },
    {
      "id": "CTL-04",
      "title": "Vendor assurance review",
      "owner": "Sarah Adams",
      "unit": "Vendor Management",
      "effectiveness": "Partially effective",
      "reqs": [
        "R07",
        "R05"
      ],
      "risks": [
        "RSK-05"
      ]
    },
    {
      "id": "CTL-05",
      "title": "Daily reconciliation",
      "owner": "Daniel Mensah",
      "unit": "Retail Operations",
      "effectiveness": "Effective",
      "reqs": [
        "R08"
      ],
      "risks": [
        "RSK-06"
      ]
    }
  ],
  "risks": [
    {
      "id": "RSK-01",
      "title": "Unauthorized privileged access",
      "unit": "Technology",
      "category": "Information security",
      "owner": "Chidi Okafor",
      "cause": "Inconsistent privileged-account controls",
      "event": "Unauthorized administrative changes",
      "impact": "Service disruption and data exposure",
      "inherent": [
        4,
        5
      ],
      "residual": [
        3,
        4
      ],
      "appetite": 9,
      "control": "CTL-01",
      "status": "Treat"
    },
    {
      "id": "RSK-02",
      "title": "Core banking service interruption",
      "unit": "Technology",
      "category": "Service continuity",
      "owner": "Daniel Mensah",
      "cause": "Recovery arrangements are not fully tested",
      "event": "Primary hosting service fails",
      "impact": "Customer transactions become unavailable",
      "inherent": [
        4,
        5
      ],
      "residual": [
        3,
        5
      ],
      "appetite": 9,
      "control": "CTL-02",
      "status": "Treat"
    },
    {
      "id": "RSK-03",
      "title": "Delayed payment recovery",
      "unit": "Payments",
      "category": "Service continuity",
      "owner": "Daniel Mensah",
      "cause": "Dependence on a single switching service",
      "event": "Switching service outage",
      "impact": "Payment backlog and customer complaints",
      "inherent": [
        4,
        4
      ],
      "residual": [
        2,
        4
      ],
      "appetite": 9,
      "control": "CTL-02",
      "status": "Monitor"
    },
    {
      "id": "RSK-04",
      "title": "Incorrect payment authorization",
      "unit": "Payments",
      "category": "Execution",
      "owner": "Amina Bello",
      "cause": "Manual payment approval exceptions",
      "event": "Unauthorized payment is processed",
      "impact": "Direct financial loss",
      "inherent": [
        4,
        4
      ],
      "residual": [
        2,
        3
      ],
      "appetite": 9,
      "control": "CTL-03",
      "status": "Monitor"
    },
    {
      "id": "RSK-05",
      "title": "Third-party data exposure",
      "unit": "Retail Operations",
      "category": "Third party",
      "owner": "Sarah Adams",
      "cause": "Incomplete service-specific assurance",
      "event": "Vendor exposes customer documents",
      "impact": "Privacy harm and response costs",
      "inherent": [
        4,
        5
      ],
      "residual": [
        3,
        4
      ],
      "appetite": 9,
      "control": "CTL-04",
      "status": "Treat"
    },
    {
      "id": "RSK-06",
      "title": "Reconciliation exceptions",
      "unit": "Retail Operations",
      "category": "Execution",
      "owner": "Daniel Mensah",
      "cause": "Delayed exception resolution",
      "event": "Posting error is not corrected",
      "impact": "Unrecovered customer reimbursements",
      "inherent": [
        3,
        4
      ],
      "residual": [
        2,
        3
      ],
      "appetite": 9,
      "control": "CTL-05",
      "status": "Monitor"
    }
  ],
  "losses": [
    {
      "id": "LOS-201",
      "title": "Payment settlement error",
      "risk": "RSK-04",
      "unit": "Payments",
      "date": "2026-11-08",
      "type": "Actual loss",
      "gross": 850000000,
      "recoveries": [
        {
          "amount": 300000000,
          "date": "2026-11-10",
          "note": "Counterparty recovery"
        }
      ],
      "owner": "Amina Bello",
      "status": "Open"
    },
    {
      "id": "LOS-202",
      "title": "Hosting outage response",
      "risk": "RSK-02",
      "unit": "Technology",
      "date": "2026-10-22",
      "type": "Actual loss",
      "gross": 1800000000,
      "recoveries": [
        {
          "amount": 600000000,
          "date": "2026-11-01",
          "note": "Service credit received"
        }
      ],
      "owner": "Daniel Mensah",
      "status": "Open"
    },
    {
      "id": "LOS-203",
      "title": "Document exposure response",
      "risk": "RSK-05",
      "unit": "Retail Operations",
      "date": "2026-10-11",
      "type": "Actual loss",
      "gross": 600000000,
      "recoveries": [],
      "owner": "Sarah Adams",
      "status": "Open"
    },
    {
      "id": "LOS-204",
      "title": "Branch reconciliation correction",
      "risk": "RSK-06",
      "unit": "Retail Operations",
      "date": "2026-09-28",
      "type": "Actual loss",
      "gross": 400000000,
      "recoveries": [
        {
          "amount": 300000000,
          "date": "2026-10-05",
          "note": "Balance recovered"
        }
      ],
      "owner": "Daniel Mensah",
      "status": "Open"
    },
    {
      "id": "LOS-205",
      "title": "Duplicate payment prevented",
      "risk": "RSK-04",
      "unit": "Payments",
      "date": "2026-11-11",
      "type": "Near miss",
      "gross": 0,
      "recoveries": [],
      "owner": "Amina Bello",
      "status": "Reviewed"
    }
  ],
  "indicators": [
    {
      "id": "KRI-01",
      "title": "Privileged accounts without MFA",
      "risk": "RSK-01",
      "type": "KRI",
      "value": 4,
      "unit": "accounts",
      "threshold": 0,
      "warning": 0,
      "direction": "above",
      "observed": "2026-11-12",
      "freshDays": 7,
      "history": [
        8,
        6,
        5,
        4
      ]
    },
    {
      "id": "KRI-02",
      "title": "Core service unavailability",
      "risk": "RSK-02",
      "type": "KRI",
      "value": 38,
      "unit": "minutes",
      "threshold": 30,
      "warning": 20,
      "direction": "above",
      "observed": "2026-11-12",
      "freshDays": 7,
      "history": [
        10,
        14,
        22,
        38
      ]
    },
    {
      "id": "KCI-03",
      "title": "Payment dual-approval coverage",
      "risk": "RSK-04",
      "type": "KCI",
      "value": 99.8,
      "unit": "%",
      "threshold": 99,
      "warning": 99.5,
      "direction": "below",
      "observed": "2026-11-12",
      "freshDays": 7,
      "history": [
        98.4,
        99.0,
        99.6,
        99.8
      ]
    },
    {
      "id": "KCI-04",
      "title": "Critical vendor evidence coverage",
      "risk": "RSK-05",
      "type": "KCI",
      "value": 80,
      "unit": "%",
      "threshold": 95,
      "warning": 98,
      "direction": "below",
      "observed": "2026-10-01",
      "freshDays": 30,
      "history": [
        88,
        82,
        80,
        80
      ]
    },
    {
      "id": "KRI-05",
      "title": "Unresolved reconciliation items",
      "risk": "RSK-06",
      "type": "KRI",
      "value": null,
      "unit": "items",
      "threshold": 10,
      "warning": 7,
      "direction": "above",
      "observed": null,
      "freshDays": 7,
      "history": []
    }
  ],
  "cycles": [
    {
      "id": "RCSA-01",
      "name": "Technology · Q4 2026",
      "unit": "Technology",
      "owner": "Chidi Okafor",
      "reviewer": "Ifeoma Nwosu",
      "due": "2026-11-25",
      "risks": [
        "RSK-01",
        "RSK-02"
      ],
      "reviews": {}
    },
    {
      "id": "RCSA-02",
      "name": "Payments · Q4 2026",
      "unit": "Payments",
      "owner": "Amina Bello",
      "reviewer": "Ifeoma Nwosu",
      "due": "2026-11-28",
      "risks": [
        "RSK-03",
        "RSK-04"
      ],
      "reviews": {
        "RSK-03": {
          "status": "Submitted",
          "effectiveness": "Partially effective",
          "likelihood": 2,
          "impact": 4,
          "rationale": "Recovery test completed with two open actions.",
          "evidence": "EV-03",
          "submittedBy": "Amina Bello",
          "submitted": "2026-11-11"
        }
      }
    },
    {
      "id": "RCSA-03",
      "name": "Retail Operations · Q4 2026",
      "unit": "Retail Operations",
      "owner": "Daniel Mensah",
      "reviewer": "Ifeoma Nwosu",
      "due": "2026-11-30",
      "risks": [
        "RSK-05",
        "RSK-06"
      ],
      "reviews": {
        "RSK-06": {
          "status": "Approved",
          "effectiveness": "Effective",
          "likelihood": 2,
          "impact": 3,
          "rationale": "Reconciliation sampling is complete.",
          "evidence": "EV-05",
          "submittedBy": "Daniel Mensah",
          "submitted": "2026-11-09",
          "reviewedBy": "Ifeoma Nwosu",
          "reviewed": "2026-11-10"
        }
      }
    }
  ],
  "evidence": [
    {
      "id": "EV-01",
      "name": "Privileged-account extract.csv",
      "target": "Meridian Bank",
      "program": "PRG-104",
      "question": "Q05",
      "observed": "2026-11-10",
      "expires": "2026-12-10",
      "status": "Accepted",
      "owner": "Chidi Okafor",
      "size": 18500
    },
    {
      "id": "EV-02",
      "name": "Access review sign-off.pdf",
      "target": "Meridian Bank",
      "program": "PRG-104",
      "question": "Q07",
      "observed": "2026-07-01",
      "expires": "2026-09-29",
      "status": "Accepted",
      "owner": "Chidi Okafor",
      "size": 82000
    },
    {
      "id": "EV-03",
      "name": "Service recovery test.pdf",
      "target": "Meridian Bank",
      "program": "PRG-104",
      "question": "Q10",
      "observed": "2026-10-27",
      "expires": "2027-04-25",
      "status": "Accepted",
      "owner": "Daniel Mensah",
      "size": 240000
    },
    {
      "id": "EV-04",
      "name": "Atlas hosting assurance.pdf",
      "target": "ENG-01",
      "program": "PRG-105",
      "question": "Q13",
      "observed": "2026-10-15",
      "expires": "2027-01-13",
      "status": "Submitted",
      "owner": "Sarah Adams",
      "size": 580000
    },
    {
      "id": "EV-05",
      "name": "Reconciliation sample.csv",
      "target": "Meridian Bank",
      "program": "PRG-104",
      "question": "Q15",
      "observed": "2026-11-10",
      "expires": "2026-12-10",
      "status": "Accepted",
      "owner": "Daniel Mensah",
      "size": 25000
    },
    {
      "id": "EV-06",
      "name": "Cedar data handling policy.pdf",
      "target": "ENG-04",
      "program": "",
      "question": "Q09",
      "observed": "2026-06-01",
      "expires": "2026-10-01",
      "status": "Submitted",
      "owner": "Sarah Adams",
      "size": 92000
    }
  ],
  "tasks": [
    {
      "id": "ACT-01",
      "title": "Resolve privileged accounts without MFA",
      "kind": "Remediation",
      "owner": "Chidi Okafor",
      "due": "2026-11-15",
      "status": "Open",
      "route": "risks",
      "ref": "RSK-01"
    },
    {
      "id": "ACT-02",
      "title": "Review Atlas hosting assurance",
      "kind": "Evidence review",
      "owner": "Ifeoma Nwosu",
      "due": "2026-11-16",
      "status": "Open",
      "route": "evidence",
      "ref": "EV-04"
    },
    {
      "id": "ACT-03",
      "title": "Refresh expired access-review evidence",
      "kind": "Evidence refresh",
      "owner": "Chidi Okafor",
      "due": "2026-11-10",
      "status": "Open",
      "route": "evidence",
      "ref": "EV-02"
    }
  ],
  "processing": [
    {
      "id": "ROPA-01",
      "name": "Digital account opening",
      "owner": "Sarah Adams",
      "unit": "Retail Operations",
      "purpose": "Customer onboarding",
      "data": "Identity and contact details",
      "vendor": "Cedar Data Services",
      "retention": "5 years — illustrative internal policy",
      "dpia": "In review"
    },
    {
      "id": "ROPA-02",
      "name": "Transaction monitoring",
      "owner": "Amina Bello",
      "unit": "Payments",
      "purpose": "Fraud monitoring",
      "data": "Transaction metadata",
      "vendor": "Internal",
      "retention": "7 years — illustrative internal policy",
      "dpia": "Approved"
    }
  ],
  "assessments": {},
  "activity": [],
  "fixtureRuns": [],
  "draft": null,
  "theme": "light"
};
