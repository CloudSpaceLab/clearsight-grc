package ropa

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
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
		description:            "Collects and verifies identity and contact details when a customer opens a personal account.",
		purpose:                "Open and verify customer accounts",
		lawfulBasis:            "Contract",
		processor:              "Retail Banking Operations",
		dataSubjectCategories:  "Customers; Prospective customers",
		personalDataCategories: "Name; Date of birth; Address; Identification document",
		retentionPeriod:        "7 years after account closure",
		securityMeasures:       "Encryption at rest and in transit; role-based access",
		owner:                  DemoOwnerPrincipalID,
		authority:              DemoRequiredAuthorityID,
		status:                 StatusOpen,
		nextReviewOffsetDays:   120,
		hasNextReviewDate:      true,
		dataCategories: []DataCategory{
			{Category: "Identity", Sensitivity: "DIRECT_PERSONAL"},
			{Category: "Contact", Sensitivity: "DIRECT_PERSONAL"},
		},
		systems: []System{
			{SystemName: "Customer onboarding portal", SystemKind: "APPLICATION"},
		},
	},
	{
		code:                   "PA-LOAN-APPLICATION",
		name:                   "Loan application assessment",
		description:            "Collects applicant financial information so the credit team can assess a loan application and proposed terms.",
		purpose:                "Assess personal and business loan applications",
		lawfulBasis:            "",
		processor:              "Credit Risk Operations",
		dataSubjectCategories:  "Customers; Prospective customers",
		personalDataCategories: "Name; Employment history; Income; Credit history",
		retentionPeriod:        "7 years after loan closure or withdrawal",
		securityMeasures:       "Encryption at rest; restricted credit-team access",
		owner:                  DemoOwnerPrincipalID,
		authority:              DemoRequiredAuthorityID,
		status:                 StatusOpen,
		nextReviewOffsetDays:   90,
		hasNextReviewDate:      true,
		dataCategories: []DataCategory{
			{Category: "Financial profile", Sensitivity: "SENSITIVE_BY_NATURE"},
			{Category: "Identity", Sensitivity: "DIRECT_PERSONAL"},
		},
		systems: []System{
			{SystemName: "Credit decisioning platform", SystemKind: "APPLICATION"},
		},
	},
	{
		code:                   "PA-CUSTOMER-SERVICE-CHANNEL",
		name:                   "Customer service channel monitoring",
		description:            "Records customer contact details and service interactions to investigate complaints and support account servicing.",
		purpose:                "Support customer servicing and complaint investigation",
		lawfulBasis:            "Legitimate interests",
		processor:              "Customer Experience Operations",
		dataSubjectCategories:  "Customers",
		personalDataCategories: "Name; Telephone number; Email address; Contact history",
		retentionPeriod:        "5 years after the last customer interaction",
		securityMeasures:       "Encryption at rest; masked contact details in reports",
		owner:                  DemoOwnerPrincipalID,
		authority:              DemoRequiredAuthorityID,
		status:                 StatusOpen,
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
			{SystemName: "Contact centre recording archive", SystemKind: "DATABASE"},
		},
	},
	{
		code:                   "PA-ARCHIVED-CUSTOMER-RECORDS",
		name:                   "Archived customer records migration",
		description:            "Retains a historical customer-record extract for controlled retrieval after the migration and routine processing have ended.",
		purpose:                "Controlled retrieval of historic customer records",
		lawfulBasis:            "Legal obligation",
		processor:              "Records Management",
		dataSubjectCategories:  "Customers",
		personalDataCategories: "Name; Account number; Transaction history; Address",
		retentionPeriod:        "10 years after the applicable records period",
		securityMeasures:       "Read-only archive storage; restricted retrieval approval",
		owner:                  DemoOwnerPrincipalID,
		authority:              DemoRequiredAuthorityID,
		status:                 StatusClosed,
		endDateOffsetMonths:    -1,
		dataCategories: []DataCategory{
			{Category: "Account identifiers", Sensitivity: "DIRECT_PERSONAL"},
			{Category: "Transaction history", Sensitivity: "SENSITIVE_BY_NATURE"},
		},
		systems: []System{
			{SystemName: "Legacy customer records archive", SystemKind: "FILE"},
		},
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
	if seed.hasNextReviewDate {
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
		Controller:                   "Clear Bank Nigeria",
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
