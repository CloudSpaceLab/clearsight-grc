package aigateway

import (
	"fmt"
	"sync/atomic"
	"time"
)

type routeRuntime struct {
	Route
	provider *providerRuntime
	breaker  *circuitBreaker
}

type aliasRuntime struct {
	alias       string
	routes      []*routeRuntime
	totalWeight uint64
	counter     atomic.Uint64
}

type router struct {
	aliases map[string]*aliasRuntime
}

func newRouter(config RuntimeConfig, providers map[string]*providerRuntime) (*router, error) {
	result := &router{aliases: make(map[string]*aliasRuntime, len(config.Models))}
	for _, model := range config.Models {
		alias := &aliasRuntime{alias: model.Alias}
		for _, input := range model.Routes {
			provider := providers[input.ProviderID]
			if provider == nil {
				return nil, fmt.Errorf("route %s provider %s is unavailable", input.ID, input.ProviderID)
			}
			route := &routeRuntime{Route: Route{
				ID: input.ID, ProviderID: input.ProviderID, Model: input.Model, Weight: input.Weight,
				Price: TokenPrice{InputPerMillion: input.InputMicroUSDPerMillionTokens, OutputPerMillion: input.OutputMicroUSDPerMillionTokens},
			}, provider: provider, breaker: newCircuitBreaker(config.CircuitBreaker)}
			alias.routes = append(alias.routes, route)
			alias.totalWeight += uint64(input.Weight)
		}
		result.aliases[model.Alias] = alias
	}
	return result, nil
}

func (r *router) candidates(workload Workload, alias string) ([]*routeRuntime, TokenPrice, error) {
	return r.candidatesFor(workload, alias, "")
}

func (r *router) candidatesFor(workload Workload, alias, routeID string) ([]*routeRuntime, TokenPrice, error) {
	if _, allowed := workload.AllowedModels[alias]; !allowed {
		return nil, TokenPrice{}, ErrModelNotFound
	}
	model := r.aliases[alias]
	if model == nil || len(model.routes) == 0 || model.totalWeight == 0 {
		return nil, TokenPrice{}, ErrModelNotFound
	}
	if routeID != "" {
		for _, route := range model.routes {
			if route.ID == routeID {
				return []*routeRuntime{route}, route.Price, nil
			}
		}
		return nil, TokenPrice{}, ErrModelNotFound
	}
	point := model.counter.Add(1) - 1
	point %= model.totalWeight
	chosen := 0
	var cumulative uint64
	for index, route := range model.routes {
		cumulative += uint64(route.Weight)
		if point < cumulative {
			chosen = index
			break
		}
	}
	ordered := make([]*routeRuntime, 0, len(model.routes))
	ordered = append(ordered, model.routes[chosen])
	for offset := 1; offset < len(model.routes); offset++ {
		ordered = append(ordered, model.routes[(chosen+offset)%len(model.routes)])
	}
	var highest TokenPrice
	for _, route := range ordered {
		if route.Price.InputPerMillion > highest.InputPerMillion {
			highest.InputPerMillion = route.Price.InputPerMillion
		}
		if route.Price.OutputPerMillion > highest.OutputPerMillion {
			highest.OutputPerMillion = route.Price.OutputPerMillion
		}
	}
	return ordered, highest, nil
}

func (r *router) ready(now time.Time) bool {
	if len(r.aliases) == 0 {
		return false
	}
	for _, alias := range r.aliases {
		available := false
		for _, route := range alias.routes {
			if route.breaker.available(now) {
				available = true
				break
			}
		}
		if !available {
			return false
		}
	}
	return true
}
