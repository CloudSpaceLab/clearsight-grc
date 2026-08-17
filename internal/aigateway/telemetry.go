package aigateway

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type telemetryKey struct {
	modelAlias string
	provider   string
	outcome    string
	stream     bool
}

type telemetryValue struct {
	requests          uint64
	inputTokens       uint64
	cachedInputTokens uint64
	outputTokens      uint64
	costMicroUSD      uint64
	durationSeconds   float64
	ttftSeconds       float64
}

type telemetry struct {
	mu      sync.Mutex
	started time.Time
	values  map[telemetryKey]telemetryValue
}

func newTelemetry(started time.Time) *telemetry {
	return &telemetry{started: started.UTC(), values: make(map[telemetryKey]telemetryValue)}
}

func (t *telemetry) record(modelAlias, provider, outcome string, stream bool, usage Usage, cost int64, duration, ttft time.Duration) {
	key := telemetryKey{
		modelAlias: metricLabel(modelAlias),
		provider:   metricLabel(provider),
		outcome:    metricLabel(outcome),
		stream:     stream,
	}
	t.mu.Lock()
	value := t.values[key]
	value.requests++
	value.inputTokens = saturatingUint64Add(value.inputTokens, usage.InputTokens)
	value.cachedInputTokens = saturatingUint64Add(value.cachedInputTokens, usage.CachedInputTokens)
	value.outputTokens = saturatingUint64Add(value.outputTokens, usage.OutputTokens)
	value.costMicroUSD = saturatingUint64Add(value.costMicroUSD, cost)
	if duration > 0 {
		value.durationSeconds += duration.Seconds()
	}
	if ttft > 0 {
		value.ttftSeconds += ttft.Seconds()
	}
	t.values[key] = value
	t.mu.Unlock()
}

func (t *telemetry) writePrometheus(writer io.Writer) error {
	t.mu.Lock()
	keys := make([]telemetryKey, 0, len(t.values))
	values := make(map[telemetryKey]telemetryValue, len(t.values))
	for key, value := range t.values {
		keys = append(keys, key)
		values[key] = value
	}
	started := t.started
	t.mu.Unlock()
	sort.Slice(keys, func(i, j int) bool {
		left, right := keys[i], keys[j]
		if left.modelAlias != right.modelAlias {
			return left.modelAlias < right.modelAlias
		}
		if left.provider != right.provider {
			return left.provider < right.provider
		}
		if left.outcome != right.outcome {
			return left.outcome < right.outcome
		}
		return !left.stream && right.stream
	})
	if _, err := fmt.Fprintln(writer, "# HELP clearsight_ai_gateway_info Stateless AI gateway process information."); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(writer, "# TYPE clearsight_ai_gateway_info gauge"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "clearsight_ai_gateway_info{started_at_unix=\"%d\"} 1\n", started.Unix()); err != nil {
		return err
	}
	families := []struct {
		help, typ, name string
		value           func(telemetryValue) string
	}{
		{"Completed gateway request attempts.", "counter", "clearsight_ai_gateway_requests_total", func(v telemetryValue) string { return strconv.FormatUint(v.requests, 10) }},
		{"Provider-reported input tokens.", "counter", "clearsight_ai_gateway_input_tokens_total", func(v telemetryValue) string { return strconv.FormatUint(v.inputTokens, 10) }},
		{"Provider-reported cached input tokens.", "counter", "clearsight_ai_gateway_cached_input_tokens_total", func(v telemetryValue) string { return strconv.FormatUint(v.cachedInputTokens, 10) }},
		{"Provider-reported output tokens.", "counter", "clearsight_ai_gateway_output_tokens_total", func(v telemetryValue) string { return strconv.FormatUint(v.outputTokens, 10) }},
		{"Calculated provider cost in micro-US-dollars.", "counter", "clearsight_ai_gateway_cost_microusd_total", func(v telemetryValue) string { return strconv.FormatUint(v.costMicroUSD, 10) }},
		{"Total gateway request duration in seconds.", "counter", "clearsight_ai_gateway_duration_seconds_total", func(v telemetryValue) string { return strconv.FormatFloat(v.durationSeconds, 'g', -1, 64) }},
		{"Total streaming time-to-first-token in seconds.", "counter", "clearsight_ai_gateway_ttft_seconds_total", func(v telemetryValue) string { return strconv.FormatFloat(v.ttftSeconds, 'g', -1, 64) }},
	}
	for _, family := range families {
		if _, err := fmt.Fprintf(writer, "# HELP %s %s\n# TYPE %s %s\n", family.name, family.help, family.name, family.typ); err != nil {
			return err
		}
		for _, key := range keys {
			labels := metricLabels(key)
			if _, err := fmt.Fprintf(writer, "%s%s %s\n", family.name, labels, family.value(values[key])); err != nil {
				return err
			}
		}
	}
	return nil
}

func metricLabels(key telemetryKey) string {
	return `{model_alias="` + escapePrometheusLabel(key.modelAlias) + `",provider="` + escapePrometheusLabel(key.provider) + `",outcome="` + escapePrometheusLabel(key.outcome) + `",stream="` + strconv.FormatBool(key.stream) + `"}`
}

func metricLabel(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 256 {
		return "unknown"
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return "unknown"
		}
	}
	return value
}

func escapePrometheusLabel(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	return strings.ReplaceAll(value, `"`, `\"`)
}

func saturatingUint64Add(current uint64, delta int64) uint64 {
	if delta <= 0 {
		return current
	}
	add := uint64(delta)
	if ^uint64(0)-current < add {
		return ^uint64(0)
	}
	return current + add
}
