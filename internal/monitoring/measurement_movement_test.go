package monitoring

import "testing"

func TestCompareNativeMeasurementsKeepsExactSignedMovement(t *testing.T) {
	tests := []struct {
		name      string
		current   NativeMeasurement
		previous  NativeMeasurement
		direction MeasurementMovementDirection
		delta     string
	}{
		{
			name: "percentage points up",
			current: NativeMeasurement{Field: "availability", Unit: MeasurementPercent, Precision: 2, Value: "99.70"},
			previous: NativeMeasurement{Field: "availability", Unit: MeasurementPercent, Precision: 2, Value: "99.50"},
			direction: MeasurementMovementUp,
			delta: "+0.20",
		},
		{
			name: "count down",
			current: NativeMeasurement{Field: "exceptions", Unit: MeasurementCount, Value: "7"},
			previous: NativeMeasurement{Field: "exceptions", Unit: MeasurementCount, Value: "10"},
			direction: MeasurementMovementDown,
			delta: "-3",
		},
		{
			name: "exact flat",
			current: NativeMeasurement{Field: "loss", Unit: MeasurementMoney, Currency: "NGN", Precision: 2, Value: "1000000.00"},
			previous: NativeMeasurement{Field: "loss", Unit: MeasurementMoney, Currency: "NGN", Precision: 2, Value: "1000000.00"},
			direction: MeasurementMovementFlat,
			delta: "0.00",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := CompareNativeMeasurements(&tt.current, &tt.previous)
			if !ok || got.Direction != tt.direction || got.Delta != tt.delta {
				t.Fatalf("movement=%#v ok=%v", got, ok)
			}
		})
	}
}

func TestCompareNativeMeasurementsRejectsIncompatibleDefinitions(t *testing.T) {
	current := NativeMeasurement{Field: "availability", Unit: MeasurementPercent, Value: "99.5"}
	for name, previous := range map[string]NativeMeasurement{
		"field": {Field: "latency", Unit: MeasurementPercent, Value: "99.4"},
		"unit": {Field: "availability", Unit: MeasurementCount, Value: "99"},
		"missing value": {Field: "availability", Unit: MeasurementPercent},
	} {
		t.Run(name, func(t *testing.T) {
			if got, ok := CompareNativeMeasurements(&current, &previous); ok {
				t.Fatalf("unexpected movement %#v", got)
			}
		})
	}
}
