//go:build postgres && postgresintegration

package metricview

import (
	"context"
	"os"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestLossPeriodProjectionPreservesCurrencyRecoveryScopeAndExactMembers(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	tenantID := mustLossPeriodID(t)
	entityID := mustLossPeriodID(t)
	childScopeID := mustLossPeriodID(t)
	siblingScopeID := mustLossPeriodID(t)
	ownerID := mustLossPeriodID(t)
	childLossID := mustLossPeriodID(t)
	unattributedLossID := mustLossPeriodID(t)
	siblingLossID := mustLossPeriodID(t)
	oldLossID := mustLossPeriodID(t)
	voidedLossID := mustLossPeriodID(t)

	now := time.Now().UTC().Truncate(time.Second)
	periodStart := now.Add(-7 * 24 * time.Hour)
	periodEnd := now.Add(-time.Second)

	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, execErr := pool.Exec(ctx, query, args...); execErr != nil {
			t.Fatal(execErr)
		}
	}
	mustExec(`INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'Loss Period Metrics')`,
		tenantID, "loss-period-"+tenantID[len(tenantID)-8:])
	mustExec(`
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from)
		VALUES($1::uuid,$2::uuid,$3,'Loss Period Nigeria','NG',$4)`,
		entityID, tenantID, "LP-"+entityID[len(entityID)-8:], periodStart.Add(-365*24*time.Hour))
	mustExec(`
		INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from)
		VALUES($1::uuid,$2::uuid,'PERSON','Loss owner','ACTIVE',$3)`,
		ownerID, tenantID, periodStart.Add(-24*time.Hour))
	mustExec(`
		INSERT INTO organization_scopes(
			id,tenant_id,legal_entity_id,code,name,kind,department_path,origin,status,valid_from
		) VALUES
		  ($1::uuid,$3::uuid,$4::uuid,'OPS','Operations','DEPARTMENT',ARRAY['OPS'],'MANAGED','ACTIVE',$5),
		  ($2::uuid,$3::uuid,$4::uuid,'FIN','Finance','DEPARTMENT',ARRAY['FIN'],'MANAGED','ACTIVE',$5)`,
		childScopeID, siblingScopeID, tenantID, entityID, periodStart.Add(-30*24*time.Hour))

	insertLoss := func(lossID, scopeID, code, currency string, gross int64, occurred time.Time, status string, version int64) {
		t.Helper()
		mustExec(`
			INSERT INTO operational_losses(
				id,tenant_id,legal_entity_id,organization_scope_id,code,title,event_type,cause,description,
				gross_amount_minor,currency,occurred_at,discovered_at,owner_principal_id,
				status,version,created_at,updated_at
			) VALUES(
				$1::uuid,$2::uuid,$3::uuid,NULLIF($4,'')::uuid,$5,$5,'OTHER','Period test','',
				$6,$7,$8,$9,$10::uuid,$11,$12,$13,$13
			)`,
			lossID, tenantID, entityID, scopeID, code, gross, currency,
			occurred, occurred.Add(time.Minute), ownerID, status, version, occurred.Add(time.Minute))
	}
	insertLoss(childLossID, childScopeID, "LOSS-CHILD", "NGN", 1000, periodStart.Add(time.Hour), "ACTIVE", 3)
	insertLoss(unattributedLossID, "", "LOSS-UNATTRIBUTED", "NGN", 500, periodStart.Add(2*time.Hour), "ACTIVE", 1)
	insertLoss(siblingLossID, siblingScopeID, "LOSS-SIBLING", "USD", 200, periodStart.Add(3*time.Hour), "ACTIVE", 1)
	insertLoss(oldLossID, childScopeID, "LOSS-OLD", "NGN", 1000, periodStart.Add(-24*time.Hour), "ACTIVE", 2)
	insertLoss(voidedLossID, childScopeID, "LOSS-VOID", "NGN", 999, periodStart.Add(4*time.Hour), "VOIDED", 1)

	insertRecovery := func(lossID, currency, kind string, amount int64, version int64, recoveredAt time.Time) {
		t.Helper()
		mustExec(`
			INSERT INTO operational_loss_recoveries(
				id,tenant_id,legal_entity_id,loss_id,loss_version,kind,amount_minor,currency,
				reference,recovered_at,actor_id,created_at
			) VALUES(
				$1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,$8,'period test',$9,$10::uuid,$9
			)`,
			mustLossPeriodID(t), tenantID, entityID, lossID, version, kind, amount, currency, recoveredAt, ownerID)
	}
	insertRecovery(childLossID, "NGN", "RECOVERY", 200, 2, periodStart.Add(4*time.Hour))
	insertRecovery(childLossID, "NGN", "REVERSAL", 50, 3, periodStart.Add(5*time.Hour))
	insertRecovery(oldLossID, "NGN", "RECOVERY", 300, 2, periodStart.Add(6*time.Hour))

	reader := NewLossPeriodRepository(pool)
	legal, err := reader.CurrentLossPeriod(
		ctx, tenantID, entityID, "", nil, periodStart, periodEnd, now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if legal.ScopeKind != "LEGAL_ENTITY" || legal.ScopeID != entityID || legal.EventCount != 3 ||
		legal.ContributingLossCount != 4 || legal.UnattributedEventCount != 1 || !legal.MixedCurrencies ||
		legal.NetLoss != nil || len(legal.Currencies) != 2 {
		t.Fatalf("legal bundle=%#v", legal)
	}
	assertLossCurrencyFlow(t, legal.Currencies, "NGN", "1500", "500", "50", "1050", 2, 2, 1)
	assertLossCurrencyFlow(t, legal.Currencies, "USD", "200", "0", "0", "200", 1, 0, 0)

	if len(legal.OrganizationBreakdown) != 3 {
		t.Fatalf("legal breakdown=%#v", legal.OrganizationBreakdown)
	}
	assertLossOrganizationFlow(t, legal.OrganizationBreakdown, "scope:"+childScopeID, childScopeID, "Operations", "ORGANIZATION_SCOPE", 1, 2, false, "NGN", "550")
	assertLossOrganizationFlow(t, legal.OrganizationBreakdown, "scope:"+siblingScopeID, siblingScopeID, "Finance", "ORGANIZATION_SCOPE", 1, 1, false, "USD", "200")
	assertLossOrganizationFlow(t, legal.OrganizationBreakdown, "unattributed", "", "Unattributed", "UNATTRIBUTED", 1, 1, false, "NGN", "500")

	if legal.FlowResolution != LossFlowResolutionDay || len(legal.FlowPoints) < 7 {
		t.Fatalf("legal flow points=%#v", legal.FlowPoints)
	}
	if legal.Comparison.EventCount != 1 || legal.Comparison.EventDelta != 2 ||
		legal.Comparison.MixedCurrencies || legal.Comparison.NetLoss == nil ||
		legal.Comparison.NetLoss.Currency != "NGN" || legal.Comparison.NetLoss.MinorUnits != "1000" ||
		legal.Comparison.NetDelta != nil || legal.Comparison.Direction != TrendUnknown ||
		legal.Comparison.ComparisonQuality != ComparisonLimited {
		t.Fatalf("legal comparison=%#v", legal.Comparison)
	}
	assertLossFlowPointTotals(t, legal.FlowPoints, 3, "NGN", "1050", "USD", "200")

	members := NewMembershipRepository(pool)
	eventPage, err := members.ListSnapshotMembers(
		ctx, tenantID, entityID, "", legal.SourceID, LossPeriodMetricEvents,
		LossPeriodDefinitionRevision, ownerID, "", 20,
	)
	if err != nil {
		t.Fatal(err)
	}
	if eventPage.Count != 3 {
		t.Fatalf("event members=%#v", eventPage)
	}
	assertLossMemberTargets(t, eventPage, childLossID, unattributedLossID, siblingLossID)

	netPage, err := members.ListSnapshotMembers(
		ctx, tenantID, entityID, "", legal.SourceID, LossPeriodMetricNet,
		LossPeriodDefinitionRevision, ownerID, "", 20,
	)
	if err != nil {
		t.Fatal(err)
	}
	if netPage.Count != 4 {
		t.Fatalf("net members=%#v", netPage)
	}
	assertLossMemberTargets(t, netPage, childLossID, unattributedLossID, siblingLossID, oldLossID)

	scoped, err := reader.CurrentLossPeriod(
		ctx, tenantID, entityID, childScopeID, []string{childScopeID}, periodStart, periodEnd, now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if scoped.ScopeKind != "ORGANIZATION_SCOPE" || scoped.ScopeID != childScopeID ||
		scoped.EventCount != 1 || scoped.ContributingLossCount != 2 || scoped.UnattributedEventCount != 0 ||
		scoped.MixedCurrencies || scoped.NetLoss == nil || scoped.NetLoss.Currency != "NGN" ||
		scoped.NetLoss.MinorUnits != "550" || len(scoped.Currencies) != 1 {
		t.Fatalf("scoped bundle=%#v", scoped)
	}

	if len(scoped.OrganizationBreakdown) != 1 {
		t.Fatalf("scoped breakdown=%#v", scoped.OrganizationBreakdown)
	}
	assertLossOrganizationFlow(t, scoped.OrganizationBreakdown, "direct", "", "Direct", "DIRECT", 1, 2, false, "NGN", "550")

	if scoped.FlowResolution != LossFlowResolutionDay || len(scoped.FlowPoints) < 7 {
		t.Fatalf("scoped flow points=%#v", scoped.FlowPoints)
	}
	if scoped.Comparison.EventCount != 1 || scoped.Comparison.EventDelta != 0 ||
		scoped.Comparison.MixedCurrencies || scoped.Comparison.NetLoss == nil ||
		scoped.Comparison.NetLoss.Currency != "NGN" || scoped.Comparison.NetLoss.MinorUnits != "1000" ||
		scoped.Comparison.NetDelta == nil || scoped.Comparison.NetDelta.Currency != "NGN" ||
		scoped.Comparison.NetDelta.MinorUnits != "-450" || scoped.Comparison.Direction != TrendImproved ||
		scoped.Comparison.ComparisonQuality != ComparisonComplete {
		t.Fatalf("scoped comparison=%#v", scoped.Comparison)
	}
	assertLossFlowPointTotals(t, scoped.FlowPoints, 1, "NGN", "550", "", "")
	scopedEventPage, err := members.ListSnapshotMembers(
		ctx, tenantID, entityID, childScopeID, scoped.SourceID, LossPeriodMetricEvents,
		LossPeriodDefinitionRevision, ownerID, "", 20,
	)
	if err != nil {
		t.Fatal(err)
	}
	if scopedEventPage.Count != 1 {
		t.Fatalf("scoped event members=%#v", scopedEventPage)
	}
	assertLossMemberTargets(t, scopedEventPage, childLossID)
	scopedNetPage, err := members.ListSnapshotMembers(
		ctx, tenantID, entityID, childScopeID, scoped.SourceID, LossPeriodMetricNet,
		LossPeriodDefinitionRevision, ownerID, "", 20,
	)
	if err != nil {
		t.Fatal(err)
	}
	if scopedNetPage.Count != 2 {
		t.Fatalf("scoped net members=%#v", scopedNetPage)
	}
	assertLossMemberTargets(t, scopedNetPage, childLossID, oldLossID)

	if legal.SourceID == scoped.SourceID {
		t.Fatal("legal-entity and scoped Loss period snapshots must not share source identity")
	}
}

func assertLossCurrencyFlow(
	t *testing.T,
	values []LossCurrencyFlow,
	currency string,
	gross string,
	recovery string,
	reversal string,
	net string,
	lossEvents int,
	recoveryEvents int,
	reversalEvents int,
) {
	t.Helper()
	for _, value := range values {
		if value.Currency != currency {
			continue
		}
		if value.Gross.MinorUnits != gross || value.Recovery.MinorUnits != recovery ||
			value.Reversal.MinorUnits != reversal || value.Net.MinorUnits != net ||
			value.LossEventCount != lossEvents || value.RecoveryEventCount != recoveryEvents ||
			value.ReversalEventCount != reversalEvents {
			t.Fatalf("%s flow=%#v", currency, value)
		}
		return
	}
	t.Fatalf("currency %s not found in %#v", currency, values)
}

func assertLossFlowPointTotals(
	t *testing.T,
	points []LossFlowPoint,
	wantEvents int,
	currencyA string,
	netA string,
	currencyB string,
	netB string,
) {
	t.Helper()
	eventCount := 0
	zeroBuckets := 0
	netByCurrency := make(map[string]int64)
	for _, point := range points {
		eventCount += point.LossEventCount
		if point.LossEventCount == 0 && len(point.Currencies) == 0 {
			zeroBuckets++
		}
		for _, flow := range point.Currencies {
			net, err := parseMoneyMinorUnits(flow.Net)
			if err != nil {
				t.Fatal(err)
			}
			netByCurrency[flow.Currency] += net
		}
	}
	if eventCount != wantEvents || zeroBuckets == 0 {
		t.Fatalf("flow event count=%d zero buckets=%d points=%#v", eventCount, zeroBuckets, points)
	}
	if currencyA != "" {
		want, err := strconv.ParseInt(netA, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		if netByCurrency[currencyA] != want {
			t.Fatalf("%s flow net=%d want=%d", currencyA, netByCurrency[currencyA], want)
		}
	}
	if currencyB != "" {
		want, err := strconv.ParseInt(netB, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		if netByCurrency[currencyB] != want {
			t.Fatalf("%s flow net=%d want=%d", currencyB, netByCurrency[currencyB], want)
		}
	}
}

func assertLossOrganizationFlow(
	t *testing.T,
	values []LossOrganizationFlow,
	key string,
	scopeID string,
	label string,
	kind string,
	lossEvents int,
	contributors int,
	mixed bool,
	currency string,
	net string,
) {
	t.Helper()
	for _, value := range values {
		if value.Key != key {
			continue
		}
		if value.ScopeID != scopeID || value.Label != label || value.Kind != kind ||
			value.LossEventCount != lossEvents || value.ContributingLossCount != contributors ||
			value.MixedCurrencies != mixed || len(value.Currencies) != 1 || value.NetLoss == nil ||
			value.NetLoss.Currency != currency || value.NetLoss.MinorUnits != net {
			t.Fatalf("organization flow %q=%#v", key, value)
		}
		return
	}
	t.Fatalf("organization flow %q not found in %#v", key, values)
}

func assertLossMemberTargets(t *testing.T, page MemberPage, want ...string) {
	t.Helper()
	got := make([]string, 0, len(page.Items))
	for _, item := range page.Items {
		if !item.Accessible || item.TargetType != "LOSS" || item.TargetID == "" {
			t.Fatalf("member=%#v", item)
		}
		got = append(got, item.TargetID)
	}
	sort.Strings(got)
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("targets=%v want=%v", got, want)
	}
	for index := range got {
		if got[index] != want[index] {
			t.Fatalf("targets=%v want=%v", got, want)
		}
	}
}

func mustLossPeriodID(t *testing.T) string {
	t.Helper()
	value, err := id.NewUUIDv7()
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestLossPeriodComparisonTreatsRecoveryOnlyImprovementAsSignedMoney(t *testing.T) {
	current := lossPeriodAggregate{Currencies: []lossCurrencyAggregate{{
		Currency: "NGN", RecoveryMinor: 700, RecoveryEventCount: 1,
	}}}
	previous := lossPeriodAggregate{Currencies: []lossCurrencyAggregate{{
		Currency: "NGN", GrossMinor: 500, LossEventCount: 1,
	}}}
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 7, 23, 59, 59, 999999999, time.UTC)

	comparison, err := buildLossPeriodComparison(current, previous, start, end)
	if err != nil {
		t.Fatal(err)
	}
	if comparison.NetDelta == nil || comparison.NetDelta.Currency != "NGN" ||
		comparison.NetDelta.MinorUnits != "-1200" ||
		comparison.Direction != TrendImproved ||
		comparison.ComparisonQuality != ComparisonComplete {
		t.Fatalf("comparison=%#v", comparison)
	}
}
