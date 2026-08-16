package aigateway

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

type budgetManager struct {
	mu     sync.Mutex
	states map[string]*workloadBudgetState
}

type workloadBudgetState struct {
	mu          sync.Mutex
	concurrent  int64
	minuteUsage map[int64]*minuteUsage
}

type minuteUsage struct {
	requests int64
	tokens   int64
	cost     int64
}

type reservation struct {
	state          *workloadBudgetState
	bucket         *minuteUsage
	reservedTokens int64
	reservedCost   int64
	finished       atomic.Bool
}

func newBudgetManager() *budgetManager {
	return &budgetManager{states: make(map[string]*workloadBudgetState)}
}

func (m *budgetManager) reserve(now time.Time, workload Workload, request Request, highestPrice TokenPrice) (*reservation, error) {
	state := m.state(workload.ID)
	minute := now.UTC().Unix() / 60
	reservedTokens, err := estimatedTokens(request)
	if err != nil {
		return nil, err
	}
	reservedCost, err := tokenCost(reservedTokens-request.MaxOutputTokens, request.MaxOutputTokens, highestPrice)
	if err != nil {
		return nil, withCause(ErrBudgetExceeded, err)
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.concurrent >= workload.MaxConcurrent {
		return nil, ErrConcurrency
	}
	bucket := state.minuteUsage[minute]
	if bucket == nil {
		bucket = &minuteUsage{}
		state.minuteUsage[minute] = bucket
	}
	if exceedsLimit(bucket.requests, 1, workload.RequestsPerMinute) ||
		exceedsLimit(bucket.tokens, reservedTokens, workload.TokensPerMinute) ||
		exceedsLimit(bucket.cost, reservedCost, workload.CostMicroUSDPerMinute) {
		return nil, ErrBudgetExceeded
	}
	bucket.requests++
	bucket.tokens += reservedTokens
	bucket.cost += reservedCost
	state.concurrent++
	for key := range state.minuteUsage {
		if key < minute-2 {
			delete(state.minuteUsage, key)
		}
	}
	return &reservation{state: state, bucket: bucket, reservedTokens: reservedTokens, reservedCost: reservedCost}, nil
}

func (m *budgetManager) state(id string) *workloadBudgetState {
	m.mu.Lock()
	defer m.mu.Unlock()
	state := m.states[id]
	if state == nil {
		state = &workloadBudgetState{minuteUsage: make(map[int64]*minuteUsage)}
		m.states[id] = state
	}
	return state
}

func (r *reservation) finish(usage Usage, price TokenPrice, usageKnown bool) (int64, error) {
	if r == nil || !r.finished.CompareAndSwap(false, true) {
		return 0, nil
	}
	actualCost := r.reservedCost
	actualTokens := r.reservedTokens
	var err error
	if usageKnown {
		actualTokens = usage.TotalTokens()
		actualCost, err = tokenCost(usage.InputTokens, usage.OutputTokens, price)
		if err != nil {
			actualTokens = r.reservedTokens
			actualCost = r.reservedCost
		}
	}
	r.state.mu.Lock()
	r.bucket.tokens = saturatingAdd(r.bucket.tokens, actualTokens-r.reservedTokens)
	r.bucket.cost = saturatingAdd(r.bucket.cost, actualCost-r.reservedCost)
	if r.state.concurrent > 0 {
		r.state.concurrent--
	}
	r.state.mu.Unlock()
	return actualCost, err
}

func estimatedTokens(request Request) (int64, error) {
	// UTF-8 bytes are a deliberately conservative upper bound for ordinary model
	// tokenization and prevent a caller from reserving fewer tokens by changing script.
	var input int64
	add := func(value int) error {
		if value < 0 || input > math.MaxInt64-int64(value) {
			return fmt.Errorf("token estimate overflow")
		}
		input += int64(value)
		return nil
	}
	for _, message := range request.Messages {
		if err := add(len(message.Text) + 8); err != nil {
			return 0, withCause(ErrInvalidRequest, err)
		}
		for _, call := range message.ToolCalls {
			if err := add(len(call.Name) + len(call.Arguments) + 16); err != nil {
				return 0, withCause(ErrInvalidRequest, err)
			}
		}
	}
	for _, tool := range request.Tools {
		if err := add(len(tool.Name) + len(tool.Description) + len(tool.Parameters) + 16); err != nil {
			return 0, withCause(ErrInvalidRequest, err)
		}
	}
	if input == 0 {
		input = 1
	}
	if input > math.MaxInt64-request.MaxOutputTokens {
		return 0, withCause(ErrInvalidRequest, fmt.Errorf("token estimate overflow"))
	}
	return input + request.MaxOutputTokens, nil
}

func tokenCost(inputTokens, outputTokens int64, price TokenPrice) (int64, error) {
	if inputTokens < 0 || outputTokens < 0 || price.InputPerMillion < 0 || price.OutputPerMillion < 0 {
		return 0, fmt.Errorf("negative token or price value")
	}
	input, err := checkedProduct(inputTokens, price.InputPerMillion)
	if err != nil {
		return 0, err
	}
	output, err := checkedProduct(outputTokens, price.OutputPerMillion)
	if err != nil || input > math.MaxInt64-output {
		return 0, fmt.Errorf("token cost overflow")
	}
	combined := input + output
	if combined == 0 {
		return 0, nil
	}
	result := combined / 1_000_000
	if combined%1_000_000 != 0 {
		if result == math.MaxInt64 {
			return 0, fmt.Errorf("token cost overflow")
		}
		result++
	}
	return result, nil
}

func checkedProduct(left, right int64) (int64, error) {
	if left == 0 || right == 0 {
		return 0, nil
	}
	if left > math.MaxInt64/right {
		return 0, fmt.Errorf("token cost overflow")
	}
	return left * right, nil
}

func exceedsLimit(current, delta, limit int64) bool {
	if current < 0 || delta < 0 || limit < 0 || current > limit {
		return true
	}
	return delta > limit-current
}

func saturatingAdd(value, delta int64) int64 {
	if delta > 0 && value > math.MaxInt64-delta {
		return math.MaxInt64
	}
	if delta < 0 && value < -delta {
		return 0
	}
	return value + delta
}
