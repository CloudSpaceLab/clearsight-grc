package today

import (
	"sort"
	"time"
)

type Service struct{ items []AttentionItem }

func NewService(items []AttentionItem) *Service {
	return &Service{items: append([]AttentionItem(nil), items...)}
}

func (s *Service) List() []AttentionItem {
	items := append([]AttentionItem(nil), s.items...)
	sort.Slice(items, func(i, j int) bool { return items[i].DueAt.Before(items[j].DueAt) })
	return items
}

func DemoItems() []AttentionItem {
	now := time.Now().UTC()
	return []AttentionItem{
		{ID: "matter_vendor_certificate", Type: "MATTER", Title: "Payment processor certificate expires in 12 days", WhyNow: "The current certificate is the final unresolved proof for the third-party assurance conclusion.", Scope: "Retail Payments · Bank NG", State: "Evidence becoming stale", Evidence: "Current until 17 Aug 2026", Owner: "Third-Party Risk", DueAt: now.Add(12 * 24 * time.Hour), PrimaryAction: "Request current certificate"},
		{ID: "matter_cbn_change", Type: "REGULATORY_CHANGE", Title: "Review proposed CBN digital-channel obligations", WhyNow: "Seven source-linked provisions may affect mobile banking and two payment vendors.", Scope: "Digital Channels · Bank NG", State: "Applicability review", Evidence: "Official source verified", Owner: "Regulatory Compliance", DueAt: now.Add(3 * 24 * time.Hour), PrimaryAction: "Review seven proposed obligations"},
		{ID: "matter_access_review", Type: "MATTER", Title: "Resolve four privileged-access exceptions", WhyNow: "IAM and HR evidence resolved 1,246 accounts; four still lack current business-need evidence.", Scope: "Treasury Operations · July 2026", State: "Waiting for focused response", Evidence: "99.7% population resolved", Owner: "Treasury Technology", DueAt: now.Add(36 * time.Hour), PrimaryAction: "Confirm four account owners"},
	}
}
