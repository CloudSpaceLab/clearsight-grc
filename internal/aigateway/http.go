package aigateway

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"sort"
	"strings"
	"time"
)

type HTTPHandler struct {
	gateway *Gateway
	config  RuntimeConfig
	logger  *slog.Logger
	mux     *http.ServeMux
}

type responseStateWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (w *responseStateWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *responseStateWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseStateWriter) Write(payload []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(payload)
}

func NewHTTPHandler(gateway *Gateway, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = gateway.logger
	}
	handler := &HTTPHandler{gateway: gateway, config: gateway.config, logger: logger, mux: http.NewServeMux()}
	handler.registerRoutes()
	return handler.recover(handler.mux)
}

func (h *HTTPHandler) recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		tracked := &responseStateWriter{ResponseWriter: writer}
		defer func() {
			if recovered := recover(); recovered != nil {
				h.logger.Error("ai gateway handler panic", "error_code", "panic")
				if !tracked.wroteHeader {
					writeGatewayError(tracked, ErrInternal)
				}
			}
		}()
		next.ServeHTTP(tracked, request)
	})
}

func (h *HTTPHandler) live(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{"status": "live"})
}

func (h *HTTPHandler) ready(writer http.ResponseWriter, _ *http.Request) {
	if !h.gateway.Ready() {
		writeJSON(writer, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
		return
	}
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *HTTPHandler) metrics(writer http.ResponseWriter, request *http.Request) {
	if h.config.MetricsDigest == nil {
		http.NotFound(writer, request)
		return
	}
	if !bearerDigestMatches(request.Header.Get("Authorization"), h.config.MetricsDigest) {
		writer.Header().Set("WWW-Authenticate", `Bearer realm="clearsight-ai-gateway-metrics"`)
		writeGatewayError(writer, ErrUnauthorized)
		return
	}
	writer.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-store")
	if err := h.gateway.Metrics(writer); err != nil {
		h.logger.Error("ai gateway metrics write failed", "error_code", "metrics_write")
	}
}

func (h *HTTPHandler) models(writer http.ResponseWriter, request *http.Request) {
	workload, err := h.gateway.Authenticate(request.Context(), request.Header.Get("Authorization"))
	if err != nil {
		writer.Header().Set("WWW-Authenticate", `Bearer realm="clearsight-ai-gateway"`)
		writeGatewayError(writer, err)
		return
	}
	aliases := make([]string, 0, len(workload.AllowedModels))
	for alias := range workload.AllowedModels {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	data := make([]map[string]any, 0, len(aliases))
	for _, alias := range aliases {
		data = append(data, map[string]any{"id": alias, "object": "model", "created": 0, "owned_by": "clearsight"})
	}
	writeJSON(writer, http.StatusOK, map[string]any{"object": "list", "data": data})
}

func (h *HTTPHandler) chatCompletions(writer http.ResponseWriter, httpRequest *http.Request) {
	workload, requestID, ok := h.prepare(writer, httpRequest)
	if !ok {
		return
	}
	if !hasJSONContentType(httpRequest.Header.Get("Content-Type")) {
		writeGatewayError(writer, invalid("Content-Type", "Content-Type must be application/json."))
		return
	}
	input, err := decodeChatRequest(httpRequest.Body, h.config.MaxRequestBytes, requestID)
	if err != nil {
		writeGatewayError(writer, err)
		return
	}
	governed, err := h.gateway.Govern(httpRequest.Context(), *workload, input)
	setDecisionHeaders(writer, governed.Decision)
	if err != nil {
		writeGatewayError(writer, err)
		return
	}
	if governed.Request.Stream {
		setStreamingWriteDeadline(writer, h.config.RequestTimeout)
		sink := newChatStreamSink(writer, governed.Request, requestID)
		err := h.gateway.StreamGoverned(httpRequest.Context(), *workload, governed, sink)
		if err != nil && !sink.started {
			writeGatewayError(writer, err)
		}
		return
	}
	response, routeID, err := h.gateway.CompleteGoverned(httpRequest.Context(), *workload, governed)
	if err != nil {
		writeGatewayError(writer, err)
		return
	}
	writer.Header().Set("X-ClearSight-Route", routeID)
	writeJSON(writer, http.StatusOK, chatCompletionResponse(input, response))
}

func (h *HTTPHandler) responses(writer http.ResponseWriter, httpRequest *http.Request) {
	workload, requestID, ok := h.prepare(writer, httpRequest)
	if !ok {
		return
	}
	if !hasJSONContentType(httpRequest.Header.Get("Content-Type")) {
		writeGatewayError(writer, invalid("Content-Type", "Content-Type must be application/json."))
		return
	}
	input, err := decodeResponsesRequest(httpRequest.Body, h.config.MaxRequestBytes, requestID)
	if err != nil {
		writeGatewayError(writer, err)
		return
	}
	governed, err := h.gateway.Govern(httpRequest.Context(), *workload, input)
	setDecisionHeaders(writer, governed.Decision)
	if err != nil {
		writeGatewayError(writer, err)
		return
	}
	if governed.Request.Stream {
		setStreamingWriteDeadline(writer, h.config.RequestTimeout)
		sink := newResponsesStreamSink(writer, governed.Request, requestID)
		err := h.gateway.StreamGoverned(httpRequest.Context(), *workload, governed, sink)
		if err != nil && !sink.started {
			writeGatewayError(writer, err)
		}
		return
	}
	response, routeID, err := h.gateway.CompleteGoverned(httpRequest.Context(), *workload, governed)
	if err != nil {
		writeGatewayError(writer, err)
		return
	}
	writer.Header().Set("X-ClearSight-Route", routeID)
	writeJSON(writer, http.StatusOK, responsesObject(input, response, "completed"))
}

func (h *HTTPHandler) prepare(writer http.ResponseWriter, request *http.Request) (*Workload, string, bool) {
	requestID := strings.TrimSpace(request.Header.Get("X-Request-ID"))
	if requestID == "" {
		requestID = newRequestID()
	}
	if !validRequestID(requestID) {
		writeGatewayError(writer, invalid("X-Request-ID", "The request identifier is invalid."))
		return nil, "", false
	}
	writer.Header().Set("X-Request-ID", requestID)
	writer.Header().Set("Cache-Control", "no-store")
	workload, err := h.gateway.Authenticate(request.Context(), request.Header.Get("Authorization"))
	if err != nil {
		writer.Header().Set("WWW-Authenticate", `Bearer realm="clearsight-ai-gateway"`)
		writeGatewayError(writer, err)
		return nil, "", false
	}
	return workload, requestID, true
}

func setDecisionHeaders(writer http.ResponseWriter, decision Decision) {
	if decision.PolicyID == "" {
		return
	}
	writer.Header().Set("X-ClearSight-Decision", string(decision.Action))
	writer.Header().Set("X-ClearSight-Policy", formatPolicyVersion(PolicySnapshot{Code: decision.PolicyCode, Version: decision.PolicyVersion}))
	if decision.ProposedAction != "" {
		writer.Header().Set("X-ClearSight-Proposed-Action", string(decision.ProposedAction))
	}
	if len(decision.ReasonCodes) > 0 {
		writer.Header().Set("X-ClearSight-Reasons", strings.Join(decision.ReasonCodes, ","))
	}
	if len(decision.Obligations) > 0 {
		codes := make([]string, 0, len(decision.Obligations))
		for _, obligation := range decision.Obligations {
			if obligation.Code != "" {
				codes = append(codes, obligation.Code)
			}
		}
		if len(codes) > 0 {
			writer.Header().Set("X-ClearSight-Obligations", strings.Join(uniqueSortedStrings(codes), ","))
		}
	}
}

func validRequestID(value string) bool {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "req_") {
		hexValue := value[len("req_"):]
		if len(hexValue) < 16 || len(hexValue) > 64 || len(hexValue)%2 != 0 {
			return false
		}
		_, err := hex.DecodeString(hexValue)
		return err == nil
	}
	if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
		return false
	}
	_, err := hex.DecodeString(strings.ReplaceAll(value, "-", ""))
	return err == nil
}

func newRequestID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return fmt.Sprintf("req_%032x", uint64(time.Now().UTC().UnixNano()))
	}
	return "req_" + hex.EncodeToString(bytes[:])
}

func hasJSONContentType(value string) bool {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(value))
	return err == nil && strings.EqualFold(mediaType, "application/json")
}

func setStreamingWriteDeadline(writer http.ResponseWriter, timeout time.Duration) {
	_ = http.NewResponseController(writer).SetWriteDeadline(time.Now().Add(timeout))
}

func decodeStrictJSON(reader io.Reader, maxBytes int64, target any) error {
	payload, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return invalid("", "The request body could not be read.")
	}
	if int64(len(payload)) > maxBytes {
		return invalid("", "The request body exceeds the configured byte limit.")
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return invalid("", "The request body is not valid for this API operation.")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return invalid("", "The request body must contain one JSON object.")
	}
	return nil
}

func writeGatewayError(writer http.ResponseWriter, err error) {
	gatewayErr := asGatewayError(err)
	status := gatewayErr.Status
	if status == 0 {
		status = http.StatusInternalServerError
	}
	writeJSON(writer, status, map[string]any{"error": map[string]any{
		"message": gatewayErr.Message, "type": errorType(gatewayErr), "param": nullableString(gatewayErr.Param), "code": gatewayErr.Code,
	}})
}

func errorType(err *Error) string {
	switch err.Status {
	case http.StatusBadRequest, http.StatusNotFound:
		return "invalid_request_error"
	case http.StatusUnauthorized, http.StatusForbidden:
		return "authentication_error"
	case http.StatusTooManyRequests:
		return "rate_limit_error"
	default:
		return "api_error"
	}
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-store")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
