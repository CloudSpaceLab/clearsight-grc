package metricview

import "testing"

func TestValidObservationMeasureKeepsCountAndMoneyContractsSeparate(t *testing.T) {
	memberCount := int64(2)
	cases := []struct {
		name       string
		observation Observation
		definition Definition
		want       bool
	}{
		{
			name: "count",
			observation: Observation{Value: 2},
			definition: Definition{Unit: MetricUnitCount},
			want: true,
		},
		{
			name: "count rejects currency",
			observation: Observation{Value: 2, Currency: "NGN"},
			definition: Definition{Unit: MetricUnitCount},
		},
		{
			name: "money",
			observation: Observation{Value: 250000, Currency: "NGN", MemberCount: &memberCount},
			definition: Definition{Unit: MetricUnitMoney},
			want: true,
		},
		{
			name: "money requires currency",
			observation: Observation{Value: 250000, MemberCount: &memberCount},
			definition: Definition{Unit: MetricUnitMoney},
		},
		{
			name: "money requires member count",
			observation: Observation{Value: 250000, Currency: "NGN"},
			definition: Definition{Unit: MetricUnitMoney},
		},
		{
			name: "negative value rejected",
			observation: Observation{Value: -1, Currency: "NGN", MemberCount: &memberCount},
			definition: Definition{Unit: MetricUnitMoney},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := validObservationMeasure(tc.observation, tc.definition); got != tc.want {
				t.Fatalf("got=%v want=%v observation=%#v definition=%#v", got, tc.want, tc.observation, tc.definition)
			}
		})
	}
}
