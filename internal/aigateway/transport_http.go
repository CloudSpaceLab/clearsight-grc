package aigateway

import (
	"net/http"
	"strings"
)

func (h *HTTPHandler) transportStatus(writer http.ResponseWriter, request *http.Request) {
	if h.config.MetricsDigest == nil {
		http.NotFound(writer, request)
		return
	}
	if !bearerDigestMatches(request.Header.Get("Authorization"), h.config.MetricsDigest) {
		writer.Header().Set("WWW-Authenticate", `Bearer realm="clearsight-ai-gateway-operations"`)
		writeGatewayError(writer, ErrUnauthorized)
		return
	}
	tenantID := strings.TrimSpace(request.URL.Query().Get("tenant_id"))
	if tenantID == "" || !validIdentifier(tenantID) {
		writeGatewayError(writer, invalid("tenant_id", "A valid tenant identifier is required."))
		return
	}
	environment := strings.ToUpper(strings.TrimSpace(request.URL.Query().Get("environment")))
	if environment == "" {
		environment = strings.ToUpper(strings.TrimSpace(h.config.Environment))
	}
	if environment != "DEVELOPMENT" && environment != "TEST" && environment != "PRODUCTION" {
		writeGatewayError(writer, invalid("environment", "The gateway environment is invalid."))
		return
	}
	status := h.gateway.RefreshTransportStatus(request.Context(), tenantID, environment)
	writer.Header().Set("Cache-Control", "no-store")
	writeJSON(writer, http.StatusOK, status)
}
