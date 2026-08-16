package aigateway

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"time"
)

// StreamSink translates canonical stream events to one OpenAI-compatible
// response protocol. Start is called only after a provider has produced a valid
// first event, preserving fallback before any downstream bytes are committed.
type StreamSink interface {
	Start(routeID string) error
	Emit(StreamEvent) error
	Fail(*Error) error
}

type Gateway struct {
	config    RuntimeConfig
	auth      *authenticator
	router    *router
	budgets   *budgetManager
	telemetry *telemetry
	logger    *slog.Logger
	now       func() time.Time
}

func NewGateway(config RuntimeConfig, logger *slog.Logger) (*Gateway, error) {
	providers := make(map[string]*providerRuntime, len(config.Providers))
	for _, providerConfig := range config.Providers {
		var provider Provider
		switch providerConfig.Kind {
		case ProviderKindOpenAI:
			provider = newOpenAIProvider(providerConfig, config.MaxProviderBodyBytes, config.MaxSSEEventBytes)
		case ProviderKindAnthropic:
			provider = newAnthropicProvider(providerConfig, config.MaxProviderBodyBytes, config.MaxSSEEventBytes)
		default:
			return nil, invalid("provider", "The provider adapter kind is not supported.")
		}
		providers[providerConfig.ID] = &providerRuntime{provider: provider, config: providerConfig}
	}
	return newGatewayWithProviders(config, providers, logger)
}

func newGatewayWithProviders(config RuntimeConfig, providers map[string]*providerRuntime, logger *slog.Logger) (*Gateway, error) {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	router, err := newRouter(config, providers)
	if err != nil {
		return nil, err
	}
	now := time.Now
	return &Gateway{
		config: config, auth: newAuthenticator(config.Workloads), router: router,
		budgets: newBudgetManager(), telemetry: newTelemetry(now()), logger: logger, now: now,
	}, nil
}

func (g *Gateway) Authenticate(header string) (*Workload, error) {
	return g.auth.authenticate(header)
}

func (g *Gateway) Ready() bool { return g.router.ready(g.now()) }

func (g *Gateway) Metrics(writer io.Writer) error { return g.telemetry.writePrometheus(writer) }

func (g *Gateway) Complete(ctx context.Context, workload Workload, request Request) (Response, string, error) {
	started := g.now()
	if err := ValidateRequest(request); err != nil {
		return Response{}, "", err
	}
	candidates, highestPrice, err := g.router.candidates(workload, request.ModelAlias)
	if err != nil {
		g.recordFailure(request, "none", asGatewayError(err), started, 0)
		return Response{}, "", err
	}
	reservation, err := g.budgets.reserve(started, workload, request, highestPrice)
	if err != nil {
		g.recordFailure(request, "none", asGatewayError(err), started, 0)
		return Response{}, "", err
	}
	requestCtx, cancel := context.WithTimeout(ctx, g.config.RequestTimeout)
	defer cancel()
	var lastError *Error
	attempted := false
	for _, route := range candidates {
		if !route.breaker.allow(g.now()) {
			continue
		}
		attempted = true
		response, callErr := route.provider.provider.Complete(requestCtx, ProviderRequest{Request: request, ProviderModel: route.Model})
		if callErr == nil {
			callErr = validateCanonicalResponse(response)
			if callErr == nil {
				route.breaker.success()
				response.ID = responseID(request.Protocol, request.ID)
				if response.CreatedAt.IsZero() {
					response.CreatedAt = g.now().UTC()
				}
				usageKnown := route.provider.config.RequireUsage || response.Usage.TotalTokens() > 0
				cost, costErr := reservation.finish(response.Usage, route.Price, usageKnown)
				if costErr != nil {
					g.logger.Error("ai gateway cost reconciliation failed", "request_id", request.ID, "workload_id", workload.ID, "model_alias", request.ModelAlias, "route_id", route.ID, "error_code", "cost_overflow")
				}
				g.telemetry.record(g.modelMetricAlias(request.ModelAlias), route.ProviderID, "success", false, response.Usage, cost, g.now().Sub(started), 0)
				g.logRequest("completed", workload, request, route, response.Usage, cost, started, 0, nil)
				return response, route.ID, nil
			}
		}
		gatewayErr := classifyContextError(requestCtx, callErr)
		lastError = gatewayErr
		if gatewayErr.Retriable {
			route.breaker.failure(g.now())
		} else {
			route.breaker.neutral()
		}
		if !gatewayErr.Retriable {
			cost, _ := reservation.finish(Usage{}, highestPrice, false)
			g.telemetry.record(g.modelMetricAlias(request.ModelAlias), route.ProviderID, gatewayErr.Code, false, Usage{}, cost, g.now().Sub(started), 0)
			g.logRequest("failed", workload, request, route, Usage{}, cost, started, 0, gatewayErr)
			return Response{}, route.ID, gatewayErr
		}
	}
	if lastError == nil {
		lastError = ErrUnavailable
	}
	usageKnown := false
	price := highestPrice
	if !attempted {
		usageKnown = true
		price = TokenPrice{}
	}
	cost, _ := reservation.finish(Usage{}, price, usageKnown)
	g.telemetry.record(g.modelMetricAlias(request.ModelAlias), "none", lastError.Code, false, Usage{}, cost, g.now().Sub(started), 0)
	g.logger.Warn("ai gateway request failed", "request_id", request.ID, "workload_id", workload.ID, "tenant_id", workload.TenantID, "model_alias", request.ModelAlias, "stream", false, "error_code", lastError.Code, "duration_ms", g.now().Sub(started).Milliseconds())
	return Response{}, "", lastError
}

func (g *Gateway) Stream(ctx context.Context, workload Workload, request Request, sink StreamSink) error {
	started := g.now()
	if err := ValidateRequest(request); err != nil {
		return err
	}
	candidates, highestPrice, err := g.router.candidates(workload, request.ModelAlias)
	if err != nil {
		g.recordFailure(request, "none", asGatewayError(err), started, 0)
		return err
	}
	reservation, err := g.budgets.reserve(started, workload, request, highestPrice)
	if err != nil {
		g.recordFailure(request, "none", asGatewayError(err), started, 0)
		return err
	}
	requestCtx, cancel := context.WithTimeout(ctx, g.config.RequestTimeout)
	defer cancel()
	var lastError *Error
	attempted := false
	for _, route := range candidates {
		if !route.breaker.allow(g.now()) {
			continue
		}
		attempted = true
		stream, openErr := route.provider.provider.Stream(requestCtx, ProviderRequest{Request: request, ProviderModel: route.Model})
		if openErr != nil {
			gatewayErr := classifyContextError(requestCtx, openErr)
			lastError = gatewayErr
			if gatewayErr.Retriable {
				route.breaker.failure(g.now())
			} else {
				route.breaker.neutral()
			}
			if gatewayErr.Retriable {
				continue
			}
			cost, _ := reservation.finish(Usage{}, highestPrice, false)
			g.telemetry.record(g.modelMetricAlias(request.ModelAlias), route.ProviderID, gatewayErr.Code, true, Usage{}, cost, g.now().Sub(started), 0)
			g.logRequest("failed", workload, request, route, Usage{}, cost, started, 0, gatewayErr)
			return gatewayErr
		}
		first, firstErr := stream.Recv()
		if firstErr != nil {
			_ = stream.Close()
			gatewayErr := classifyContextError(requestCtx, firstErr)
			lastError = gatewayErr
			if gatewayErr.Retriable {
				route.breaker.failure(g.now())
			} else {
				route.breaker.neutral()
			}
			if gatewayErr.Retriable {
				continue
			}
			cost, _ := reservation.finish(Usage{}, highestPrice, false)
			g.telemetry.record(g.modelMetricAlias(request.ModelAlias), route.ProviderID, gatewayErr.Code, true, Usage{}, cost, g.now().Sub(started), 0)
			g.logRequest("failed", workload, request, route, Usage{}, cost, started, 0, gatewayErr)
			return gatewayErr
		}
		validator := newCanonicalStreamValidator(route.provider.config.RequireUsage)
		if validationErr := validator.accept(first); validationErr != nil {
			_ = stream.Close()
			gatewayErr := classifyContextError(requestCtx, validationErr)
			lastError = gatewayErr
			if gatewayErr.Retriable {
				route.breaker.failure(g.now())
				continue
			}
			route.breaker.neutral()
			cost, _ := reservation.finish(Usage{}, highestPrice, false)
			g.telemetry.record(g.modelMetricAlias(request.ModelAlias), route.ProviderID, gatewayErr.Code, true, Usage{}, cost, g.now().Sub(started), 0)
			g.logRequest("failed", workload, request, route, Usage{}, cost, started, 0, gatewayErr)
			return gatewayErr
		}
		if err := sink.Start(route.ID); err != nil {
			_ = stream.Close()
			route.breaker.neutral()
			cost, _ := reservation.finish(Usage{}, highestPrice, false)
			g.telemetry.record(g.modelMetricAlias(request.ModelAlias), route.ProviderID, "client_disconnected", true, Usage{}, cost, g.now().Sub(started), 0)
			return err
		}
		ttft := g.now().Sub(started)
		usage, usageKnown, completed, streamErr := g.consumeStream(requestCtx, stream, first, sink, validator)
		_ = stream.Close()
		if completed {
			route.breaker.success()
			cost, costErr := reservation.finish(usage, route.Price, usageKnown)
			if costErr != nil {
				g.logger.Error("ai gateway stream cost reconciliation failed", "request_id", request.ID, "workload_id", workload.ID, "model_alias", request.ModelAlias, "route_id", route.ID, "error_code", "cost_overflow")
			}
			g.telemetry.record(g.modelMetricAlias(request.ModelAlias), route.ProviderID, "success", true, usage, cost, g.now().Sub(started), ttft)
			g.logRequest("completed", workload, request, route, usage, cost, started, ttft, nil)
			return nil
		}
		gatewayErr := classifyContextError(requestCtx, streamErr)
		if errors.Is(gatewayErr, ErrCanceled) || errors.Is(streamErr, io.ErrClosedPipe) {
			route.breaker.neutral()
			cost, _ := reservation.finish(usage, highestPrice, usageKnown)
			g.telemetry.record(g.modelMetricAlias(request.ModelAlias), route.ProviderID, "client_disconnected", true, usage, cost, g.now().Sub(started), ttft)
			return streamErr
		}
		if gatewayErr.Retriable {
			route.breaker.failure(g.now())
		} else {
			route.breaker.neutral()
		}
		cost, _ := reservation.finish(usage, highestPrice, usageKnown)
		g.telemetry.record(g.modelMetricAlias(request.ModelAlias), route.ProviderID, gatewayErr.Code, true, usage, cost, g.now().Sub(started), ttft)
		g.logRequest("failed", workload, request, route, usage, cost, started, ttft, gatewayErr)
		_ = sink.Fail(gatewayErr)
		return gatewayErr
	}
	if lastError == nil {
		lastError = ErrUnavailable
	}
	usageKnown := false
	price := highestPrice
	if !attempted {
		usageKnown = true
		price = TokenPrice{}
	}
	cost, _ := reservation.finish(Usage{}, price, usageKnown)
	g.telemetry.record(g.modelMetricAlias(request.ModelAlias), "none", lastError.Code, true, Usage{}, cost, g.now().Sub(started), 0)
	return lastError
}

func (g *Gateway) consumeStream(ctx context.Context, stream ProviderStream, first StreamEvent, sink StreamSink, validator *canonicalStreamValidator) (Usage, bool, bool, error) {
	var usage Usage
	usageKnown := false
	event := first
	for {
		if event.Type == StreamUsage && event.Usage != nil {
			usage = *event.Usage
			usageKnown = true
		}
		if err := sink.Emit(event); err != nil {
			return usage, usageKnown, false, err
		}
		if event.Type == StreamDone {
			return usage, usageKnown, true, nil
		}
		select {
		case <-ctx.Done():
			return usage, usageKnown, false, ctx.Err()
		default:
		}
		next, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				err = ErrStream
			}
			return usage, usageKnown, false, err
		}
		if err := validator.accept(next); err != nil {
			return usage, usageKnown, false, err
		}
		event = next
	}
}

func classifyContextError(ctx context.Context, err error) *Error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
		return withCause(ErrCanceled, err)
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return withCause(ErrTimeout, err)
	}
	return asGatewayError(err)
}

func (g *Gateway) modelMetricAlias(alias string) string {
	if _, exists := g.router.aliases[alias]; !exists {
		return "unknown"
	}
	return metricLabel(alias)
}

func (g *Gateway) recordFailure(request Request, provider string, gatewayErr *Error, started time.Time, ttft time.Duration) {
	g.telemetry.record(g.modelMetricAlias(request.ModelAlias), provider, gatewayErr.Code, request.Stream, Usage{}, 0, g.now().Sub(started), ttft)
}

func (g *Gateway) logRequest(state string, workload Workload, request Request, route *routeRuntime, usage Usage, cost int64, started time.Time, ttft time.Duration, gatewayErr *Error) {
	attributes := []any{
		"request_id", request.ID, "workload_id", workload.ID, "tenant_id", workload.TenantID,
		"model_alias", request.ModelAlias, "route_id", route.ID, "provider_id", route.ProviderID,
		"stream", request.Stream, "duration_ms", g.now().Sub(started).Milliseconds(), "ttft_ms", ttft.Milliseconds(),
		"input_tokens", usage.InputTokens, "cached_input_tokens", usage.CachedInputTokens, "output_tokens", usage.OutputTokens,
		"cost_microusd", cost,
	}
	if gatewayErr != nil {
		attributes = append(attributes, "error_code", gatewayErr.Code)
	}
	if state == "completed" {
		g.logger.Info("ai gateway request completed", attributes...)
	} else {
		g.logger.Warn("ai gateway request failed", attributes...)
	}
}

func responseID(protocol Protocol, requestID string) string {
	if protocol == ProtocolResponses {
		return "resp_" + requestID
	}
	return "chatcmpl_" + requestID
}
