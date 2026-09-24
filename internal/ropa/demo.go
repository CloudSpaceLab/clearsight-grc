package ropa

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

// The demo scope is shared with the existing non-production bank installer. The
// identifiers are deliberately kept on the surface-facing scope rather than in
// activity names or codes, so the records remain ordinary operating language
// while a surface can label the population with IsDemoScope.
const (
	DemoTenant              = "bank-demo"
	DemoLegalEntity         = "bank-ng"
	DemoOwnerPrincipalID    = "owner-demo"
	DemoReviewerPrincipalID = "reviewer-demo"
	DemoActorPrincipalID    = "user-demo"
	DemoRequiredAuthorityID = "role-cro"
	DemoCISOPrincipalID     = "role-ciso"
)

const (
	demoSystemFinacleTreasury      = "Finacle Treasury"
	demoSystemFincoreColigo        = "Fincore/Coligo"
	demoSystemRingoSMS             = "Ringo sms"
	demoSystemBVN                  = "BVN Link Portal/Matching System"
	demoSystemEntrust              = "Entrust/Entrust Middleware (Credential Security)"
	demoSystemFortiProxy           = "FortiProxy, Analyzer, Manager, Gate (Firewall Management)"
	demoSystemFalcon               = "Falcon (AD Security)"
	demoSystemCheckmarx            = "Checkmarx (Software testing/code scanning)"
	demoSystemCardManagement       = "Card Management Portal/Instant card"
	demoSystemSoftToken            = "Soft Token"
	demoSystemQradar               = "Log management review (Qradar)"
	demoSystemVirusScan            = "Virus Scan and Update"
	demoSystemFIMReview            = "File Integrity Monitoring Review"
	demoSystemPatching             = "Patching Process"
	demoSystemAccessReview         = "User Access Control Review"
	demoSystemPenetrationTesting   = "Internal & External Penetration Testing"
	demoSystemCloudspacePOS        = "Cloudspace OEM — POS Support/PTSP"
	demoSystemCloudspaceMontgomery = "Cloudspace OEM — Montgomery Vault Services"
)

type demoSeed struct {
	code                   string
	name                   string
	description            string
	purpose                string
	lawfulBasis            string
	processor              string
	dataSubjectCategories  string
	personalDataCategories string
	retentionPeriod        string
	securityMeasures       string
	owner                  string
	authority              string
	status                 Status
	startDate              time.Time
	endDateOffsetMonths    int
	nextReviewDate         time.Time
	nextReviewOffsetDays   int
	hasNextReviewDate      bool
	dataCategories         []DataCategory
	recipients             []Recipient
	systems                []System
	review                 Review
}

var demoSeeds = []demoSeed{
	{
		code:                   "PA-CUSTOMER-ACCOUNT-OPENING",
		name:                   "Customer account opening",
		description:            "Sample data: collect and verify identity and contact details when a customer opens a personal account. The record is linked to the Digital Identity & KYC reference obligation; confirm the current applicability with the privacy office.",
		purpose:                "Open and verify customer accounts and digital identity records",
		lawfulBasis:            "Contract",
		processor:              "Retail Banking Operations",
		dataSubjectCategories:  "Customers; Prospective customers",
		personalDataCategories: "Name; Date of birth; Address; Identification document",
		retentionPeriod:        "7 years after account closure",
		securityMeasures:       "Encryption at rest and in transit; role-based access; BVN matching and soft-token authentication",
		owner:                  demoSourcePrincipalID("Somto"),
		authority:              DemoRequiredAuthorityID,
		status:                 StatusOpen,
		startDate:              time.Date(2024, 2, 12, 0, 0, 0, 0, time.UTC),
		nextReviewOffsetDays:   120,
		hasNextReviewDate:      true,
		dataCategories: []DataCategory{
			{Category: "Identity", Sensitivity: "DIRECT_PERSONAL"},
			{Category: "Contact", Sensitivity: "DIRECT_PERSONAL"},
		},
		systems: []System{
			{SystemName: demoSystemBVN, SystemKind: "APPLICATION"},
			{SystemName: demoSystemSoftToken, SystemKind: "APPLICATION"},
		},
	},
	{
		code:                   "PA-LOAN-APPLICATION",
		name:                   "Loan application assessment",
		description:            "Sample data: collect applicant financial information so the credit team can assess a loan application and proposed terms. The lawful basis remains open for confirmation in the supplied reference material.",
		purpose:                "Assess personal and business loan applications",
		lawfulBasis:            "",
		processor:              "Credit Risk Operations",
		dataSubjectCategories:  "Customers; Prospective customers",
		personalDataCategories: "Name; Employment history; Income; Credit history",
		retentionPeriod:        "7 years after loan closure or withdrawal",
		securityMeasures:       "Encryption at rest; restricted credit-team access; BVN matching and soft-token authentication",
		owner:                  demoSourcePrincipalID("Godspower"),
		authority:              DemoRequiredAuthorityID,
		status:                 StatusOpen,
		startDate:              time.Date(2024, 5, 6, 0, 0, 0, 0, time.UTC),
		nextReviewOffsetDays:   90,
		hasNextReviewDate:      true,
		dataCategories: []DataCategory{
			{Category: "Financial profile", Sensitivity: "SENSITIVE_BY_NATURE"},
			{Category: "Identity", Sensitivity: "DIRECT_PERSONAL"},
		},
		systems: []System{
			{SystemName: demoSystemFincoreColigo, SystemKind: "APPLICATION"},
			{SystemName: demoSystemBVN, SystemKind: "APPLICATION"},
			{SystemName: demoSystemSoftToken, SystemKind: "APPLICATION"},
		},
	},
	{
		code:                   "PA-CUSTOMER-SERVICE-CHANNEL",
		name:                   "Customer service channel monitoring",
		description:            "Sample data: record customer contact details and service interactions to investigate complaints and support account servicing. Ringo sms carries customer notifications, while the card service is used when the interaction concerns a card.",
		purpose:                "Support customer servicing, card servicing and complaint investigation",
		lawfulBasis:            "Legitimate interests",
		processor:              "Customer Experience Operations",
		dataSubjectCategories:  "Customers",
		personalDataCategories: "Name; Telephone number; Email address; Contact and card-service history",
		retentionPeriod:        "5 years after the last customer interaction",
		securityMeasures:       "Encryption at rest; masked contact details in reports; role-based access",
		owner:                  demoSourcePrincipalID("Tobi"),
		authority:              DemoRequiredAuthorityID,
		status:                 StatusOpen,
		startDate:              time.Date(2023, 11, 20, 0, 0, 0, 0, time.UTC),
		nextReviewOffsetDays:   -7,
		hasNextReviewDate:      true,
		dataCategories: []DataCategory{
			{Category: "Contact details", Sensitivity: "DIRECT_PERSONAL"},
			{Category: "Interaction history", Sensitivity: "INDIRECT_PERSONAL"},
		},
		recipients: []Recipient{
			{Recipient: "Customer Experience Operations", RecipientKind: "INTERNAL", IsCrossBorder: false, TransferBasis: TransferBasisNotApplicable},
		},
		systems: []System{
			{SystemName: demoSystemRingoSMS, SystemKind: "APPLICATION"},
			{SystemName: demoSystemCardManagement, SystemKind: "APPLICATION"},
		},
	},
	{
		code:                   "PA-ARCHIVED-CUSTOMER-RECORDS",
		name:                   "Archived customer records migration",
		description:            "Sample data: retain a historical customer-record extract from Finacle Treasury for controlled retrieval after routine processing and migration ended.",
		purpose:                "Controlled retrieval of historic customer records",
		lawfulBasis:            "Legal obligation",
		processor:              "Records Management",
		dataSubjectCategories:  "Customers",
		personalDataCategories: "Name; Account number; Transaction history; Address",
		retentionPeriod:        "10 years after the applicable records period",
		securityMeasures:       "Read-only archive storage; restricted retrieval approval",
		owner:                  demoSourcePrincipalID("Tobi"),
		authority:              DemoRequiredAuthorityID,
		status:                 StatusClosed,
		startDate:              time.Date(2022, 3, 1, 0, 0, 0, 0, time.UTC),
		endDateOffsetMonths:    -1,
		dataCategories: []DataCategory{
			{Category: "Account identifiers", Sensitivity: "DIRECT_PERSONAL"},
			{Category: "Transaction history", Sensitivity: "SENSITIVE_BY_NATURE"},
		},
		systems: []System{
			{SystemName: demoSystemFinacleTreasury, SystemKind: "APPLICATION"},
		},
	},
	{
		code:                   "PA-PAYMENTS-TREASURY-OPERATIONS",
		name:                   "Payments and treasury operations",
		description:            "Sample data: process POS, card and treasury instructions through the bank's Nigerian payment estate. Cloudspace OEM provides POS Support/PTSP and Montgomery Vault Services. The open third-party register findings are no adopted information security management standard such as ISO 27001, no VAPT, no right-to-audit clause in the SLA, and no certificate of compliance with ISO 27001 and ISO 22301 for the PTSP. The workplan names Ebube for POS Support/PTSP and Hakeem for the register action. Review the open evidence and contract gaps before relying on this service.",
		purpose:                "Process and monitor domestic card, POS and treasury transactions for Payment Systems, Instant Payments, Outsourcing Governance and Operational Resilience reference obligations",
		lawfulBasis:            "Contract",
		processor:              "Cloudspace OEM / POS Business",
		dataSubjectCategories:  "Customers; Cardholders; Service providers",
		personalDataCategories: "Account and card identifiers; Transaction data; Merchant and terminal identifiers",
		retentionPeriod:        "7 years after the applicable payment record period",
		securityMeasures:       "Tokenisation; encryption in transit and at rest; role-based access; domestic processor oversight",
		owner:                  demoSourcePrincipalID("Hakeem"),
		authority:              DemoRequiredAuthorityID,
		status:                 StatusOpen,
		startDate:              time.Date(2023, 4, 1, 0, 0, 0, 0, time.UTC),
		nextReviewDate:         time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC),
		hasNextReviewDate:      true,
		dataCategories: []DataCategory{
			{Category: "Payment transaction data", Sensitivity: "SENSITIVE_BY_NATURE"},
			{Category: "Account and card identifiers", Sensitivity: "DIRECT_PERSONAL"},
			{Category: "Merchant and terminal identifiers", Sensitivity: "INDIRECT_PERSONAL"},
		},
		recipients: []Recipient{
			// CountryCode is intentionally empty because the current ROPA
			// invariant reserves that field for cross-border recipients; the
			// Nigerian operating context is recorded in the activity facts.
			{Recipient: "Cloudspace OEM", RecipientKind: "EXTERNAL", IsCrossBorder: false, TransferBasis: TransferBasisNotApplicable},
		},
		systems: []System{
			{SystemName: demoSystemFinacleTreasury, SystemKind: "APPLICATION"},
			{SystemName: demoSystemFincoreColigo, SystemKind: "APPLICATION"},
			{SystemName: demoSystemCardManagement, SystemKind: "APPLICATION"},
			{SystemName: demoSystemCloudspacePOS, SystemKind: "THIRD_PARTY"},
			{SystemName: demoSystemCloudspaceMontgomery, SystemKind: "THIRD_PARTY"},
		},
		review: openDemoReview(
			"PA-PAYMENTS-TREASURY-OPERATIONS-review",
			time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC),
		),
	},
	{
		code:                   "PA-SECURITY-MONITORING-CONTROL-REVIEW",
		name:                   "Security monitoring and control review",
		description:            "Sample data: review security events, vulnerability results, access decisions and infrastructure changes. The activity is supported by the Qradar, Checkmarx, Falcon, FortiProxy and Entrust controls listed in the supplied IT risk workplan. The source workplan names Sikiru for Qradar and file-integrity review, Ivason for virus, patching and access review, Ese for Entrust, Fawaz for FortiProxy, Adetutu for Falcon and Ginika for Checkmarx.",
		purpose:                "Monitor security events and review infrastructure and application safeguards under Cybersecurity and User Access Control reference obligations",
		lawfulBasis:            "Legitimate interests",
		processor:              "Information Security",
		dataSubjectCategories:  "Staff; Contractors; Customers",
		personalDataCategories: "Security and access events; Employee and device identifiers; Vulnerability findings",
		retentionPeriod:        "7 years after the applicable security record period",
		securityMeasures:       "Central log review; vulnerability scanning; privileged-access review; firewall and credential monitoring",
		owner:                  demoSourcePrincipalID("Sikiru"),
		authority:              DemoRequiredAuthorityID,
		status:                 StatusOpen,
		startDate:              time.Date(2024, 9, 24, 0, 0, 0, 0, time.UTC),
		nextReviewOffsetDays:   60,
		hasNextReviewDate:      true,
		dataCategories: []DataCategory{
			{Category: "Security and access events", Sensitivity: "INDIRECT_PERSONAL"},
			{Category: "Employee and device identifiers", Sensitivity: "DIRECT_PERSONAL"},
			{Category: "Vulnerability findings", Sensitivity: "SENSITIVE_BY_NATURE"},
		},
		systems: []System{
			{SystemName: demoSystemQradar, SystemKind: "APPLICATION"},
			{SystemName: demoSystemVirusScan, SystemKind: "APPLICATION"},
			{SystemName: demoSystemFIMReview, SystemKind: "APPLICATION"},
			{SystemName: demoSystemPatching, SystemKind: "APPLICATION"},
			{SystemName: demoSystemAccessReview, SystemKind: "MANUAL"},
			{SystemName: demoSystemPenetrationTesting, SystemKind: "THIRD_PARTY"},
			{SystemName: demoSystemCheckmarx, SystemKind: "APPLICATION"},
			{SystemName: demoSystemFortiProxy, SystemKind: "APPLICATION"},
			{SystemName: demoSystemFalcon, SystemKind: "APPLICATION"},
			{SystemName: demoSystemEntrust, SystemKind: "APPLICATION"},
		},
	},
	{
		code:                   "PA-AZURE-USER-ACCESS-MANAGEMENT",
		name:                   "Azure user access management",
		description:            "Sample data: Risk ID 72 from Sample IT Risk Exception Register (1).xlsx. The open finding records 160 stale staff and guest accounts active for over three months on the Azure portal, affecting 28 guest users and 133 staff members. The source record was published 29 October 2025, targeted 31 January 2026, and remains OPEN; the closure timeline is exceeded. Control: ISO 27002:2022 A.5.18. This is sample reference data, not legal advice.",
		purpose:                "Review and remediate user and guest access on the Azure portal",
		lawfulBasis:            "",
		processor:              "CISO / Security Engineering / Technology Vendor Management",
		dataSubjectCategories:  "Staff; Guests; Prospective customers",
		personalDataCategories: "Staff and guest account records; Access and activity logs",
		retentionPeriod:        "7 years after the applicable access-audit period",
		securityMeasures:       "Conditional access; privileged-access review; activity logging; credential security",
		owner:                  DemoCISOPrincipalID,
		authority:              DemoRequiredAuthorityID,
		status:                 StatusOpen,
		startDate:              time.Date(2025, 10, 29, 0, 0, 0, 0, time.UTC),
		nextReviewDate:         time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC),
		hasNextReviewDate:      true,
		dataCategories: []DataCategory{
			{Category: "Staff and guest account records", Sensitivity: "DIRECT_PERSONAL"},
			{Category: "Access and activity logs", Sensitivity: "INDIRECT_PERSONAL"},
		},
		systems: []System{
			{SystemName: "Azure portal", SystemKind: "APPLICATION"},
			{SystemName: demoSystemFalcon, SystemKind: "APPLICATION"},
			{SystemName: demoSystemEntrust, SystemKind: "APPLICATION"},
		},
		review: openDemoReview(
			"PA-AZURE-USER-ACCESS-MANAGEMENT-review",
			time.Date(2025, 10, 29, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC),
		),
	},
	{
		code:                   "PA-AZURE-DEVICE-COMPLIANCE",
		name:                   "Azure device compliance management",
		description:            "Sample data: Risk ID 82 from Sample IT Risk Exception Register (1).xlsx. The open finding records 1,315 stale devices, 6,352 uncompliant devices and 8,693 unmanaged devices on the Microsoft Entra device dashboard. The source record was published 29 October 2025, targeted 31 January 2026, and remains OPEN; the closure timeline is exceeded. Owner: CISO / Security Engr / TVM. Control: ISO 27001 A.8.1.1. This is sample reference data, not legal advice.",
		purpose:                "Review and remediate device compliance on the Azure portal",
		lawfulBasis:            "Legitimate interests",
		processor:              "CISO / Security Engineering / Technology Vendor Management",
		dataSubjectCategories:  "Staff; Guests; Service providers",
		personalDataCategories: "Device and user identifiers; Device compliance status; Device activity logs",
		retentionPeriod:        "7 years after the applicable device-audit period",
		securityMeasures:       "Device compliance policy; conditional access; endpoint monitoring; firewall and credential review",
		owner:                  DemoCISOPrincipalID,
		authority:              DemoRequiredAuthorityID,
		status:                 StatusOpen,
		startDate:              time.Date(2025, 10, 29, 0, 0, 0, 0, time.UTC),
		nextReviewDate:         time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC),
		hasNextReviewDate:      true,
		dataCategories: []DataCategory{
			{Category: "Device and user identifiers", Sensitivity: "DIRECT_PERSONAL"},
			{Category: "Device compliance status", Sensitivity: "INDIRECT_PERSONAL"},
			{Category: "Device activity logs", Sensitivity: "INDIRECT_PERSONAL"},
		},
		systems: []System{
			{SystemName: "Azure portal", SystemKind: "APPLICATION"},
			{SystemName: demoSystemFortiProxy, SystemKind: "APPLICATION"},
			{SystemName: demoSystemFalcon, SystemKind: "APPLICATION"},
			{SystemName: demoSystemEntrust, SystemKind: "APPLICATION"},
		},
		review: openDemoReview(
			"PA-AZURE-DEVICE-COMPLIANCE-review",
			time.Date(2025, 10, 29, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC),
		),
	},
}

func InstallDemo(ctx context.Context, service *Service) error {
	if service == nil || service.repository == nil {
		return ErrInvalid
	}

	now := service.now()
	for _, seed := range demoSeeds {
		input := demoCreateInput(seed, now)
		activity, err := service.CreateActivity(ctx, input)
		if err != nil {
			if errors.Is(err, ErrDuplicate) {
				// Stable demo codes are the idempotency key. A repeat install
				// must not recreate a row or rewrite its governed history.
				continue
			}
			return fmt.Errorf("create demo processing activity %s: %w", seed.code, err)
		}

		if seed.status != StatusNew {
			_, err = service.TransitionActivity(ctx, TransitionActivityInput{
				TenantID:        activity.TenantID,
				LegalEntityID:   activity.LegalEntityID,
				ActivityID:      activity.ID,
				ExpectedVersion: activity.Version,
				To:              seed.status,
				EndDate:         input.EndDate,
				ActorID:         DemoActorPrincipalID,
			})
			if err != nil {
				return fmt.Errorf("transition demo processing activity %s to %s: %w", seed.code, seed.status, err)
			}
		}
	}
	return nil
}

// IsDemoScope reports whether a read belongs to the explicitly labelled sample
// population. It does not infer a label from a row's business name or code.
func IsDemoScope(tenantID, legalEntityID string) bool {
	return strings.TrimSpace(tenantID) == DemoTenant && strings.TrimSpace(legalEntityID) == DemoLegalEntity
}

func demoCreateInput(seed demoSeed, now time.Time) CreateActivityInput {
	startDate := seed.startDate
	if startDate.IsZero() {
		startDate = now.AddDate(-2, 0, 0)
	}
	var endDate *time.Time
	if seed.endDateOffsetMonths != 0 {
		value := now.AddDate(0, seed.endDateOffsetMonths, 0)
		endDate = &value
	}
	var nextReviewDate *time.Time
	if !seed.nextReviewDate.IsZero() {
		value := seed.nextReviewDate.UTC()
		nextReviewDate = &value
	} else if seed.hasNextReviewDate {
		value := now.AddDate(0, 0, seed.nextReviewOffsetDays)
		nextReviewDate = &value
	}
	review := seed.review
	if review.ID == "" {
		review = demoReview(now, seed.code+"-review")
	}
	return CreateActivityInput{
		TenantID:                     DemoTenant,
		LegalEntityID:                DemoLegalEntity,
		Code:                         seed.code,
		Name:                         seed.name,
		Description:                  seed.description,
		Purpose:                      seed.purpose,
		LawfulBasis:                  seed.lawfulBasis,
		Controller:                   "Meridian Trust Bank",
		Processor:                    seed.processor,
		DataSubjectCategories:        seed.dataSubjectCategories,
		PersonalDataCategories:       seed.personalDataCategories,
		SecurityMeasures:             seed.securityMeasures,
		RetentionPeriod:              seed.retentionPeriod,
		StartDate:                    &startDate,
		EndDate:                      endDate,
		NextReviewDate:               nextReviewDate,
		OwnerPrincipalID:             seed.owner,
		RequiredAuthorityPrincipalID: seed.authority,
		DataCategories:               seed.dataCategories,
		Recipients:                   seed.recipients,
		Systems:                      seed.systems,
		Reviews:                      []Review{review},
		ActorID:                      DemoActorPrincipalID,
	}
}

func demoReview(now time.Time, id string) Review {
	dueDate := now.AddDate(0, 0, -30)
	createdAt := dueDate.AddDate(0, 0, -30)
	completedAt := dueDate.AddDate(0, 0, -7)
	return Review{
		ID:                  id,
		CreatedAt:           createdAt,
		DueDate:             dueDate,
		CompletedAt:         &completedAt,
		Outcome:             "CONFIRMED",
		ReviewerPrincipalID: DemoReviewerPrincipalID,
	}
}

func openDemoReview(id string, createdAt, dueDate time.Time) Review {
	return Review{
		ID:                  id,
		CreatedAt:           createdAt.UTC(),
		DueDate:             dueDate.UTC(),
		ReviewerPrincipalID: DemoReviewerPrincipalID,
	}
}

// demoSourcePrincipalID reuses the deterministic IDs provisioned by the
// source-directory seed; a display name is never written into the principal
// reference field.
func demoSourcePrincipalID(displayName string) string {
	return identity.DemoSourceEmployeePrincipalID(displayName)
}
