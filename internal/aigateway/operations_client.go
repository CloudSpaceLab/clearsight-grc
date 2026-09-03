package aigateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxOperationsStatusBytes = 32 << 10

type OperationsClient struct {
	baseURL string
	token   string
	client  *http.Client
}

func NewOperationsClient(baseURL, token string, timeout time.Duration) (*OperationsClient, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	token = strings.TrimSpace(token)
	if baseURL == "" || token == "" || timeout <= 0 || timeout > 10*time.Second {
		return nil, fmt.Errorf("AI gateway operations client configuration is incomplete")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || !parsed.IsAbs() || parsed.User != nil || parsed.Hostname() == "" || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return nil, fmt.Errorf("AI gateway operations URL must be a fixed origin")
	}
	if len(token) < 32 || len(token) > 4096 || !safeHeaderValue(token) {
		return nil, fmt.Errorf("AI gateway operations credential is invalid")
	}
	return &OperationsClient{
		baseURL: baseURL,
		token:   token,
		client: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

func (client *OperationsClient) TransportStatus(ctx context.Context, tenantID, environment string) (TransportApplyStatus, error) {
	if client == nil || client.client == nil || !validIdentifier(tenantID) {
		return TransportApplyStatus{}, fmt.Errorf("AI gateway operations client is unavailable")
	}
	environment = strings.ToUpper(strings.TrimSpace(environment))
	if environment != "DEVELOPMENT" && environment != "TEST" && environment != "PRODUCTION" {
		return TransportApplyStatus{}, fmt.Errorf("AI gateway operations environment is invalid")
	}
	endpoint, err := url.Parse(client.baseURL + "/health/config")
	if err != nil {
		return TransportApplyStatus{}, fmt.Errorf("construct AI gateway operations request")
	}
	query := endpoint.Query()
	query.Set("tenant_id", strings.TrimSpace(tenantID))
	query.Set("environment", environment)
	endpoint.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return TransportApplyStatus{}, fmt.Errorf("construct AI gateway operations request")
	}
	request.Header.Set("Authorization", "Bearer "+client.token)
	request.Header.Set("Accept", "application/json")
	response, err := client.client.Do(request)
	if err != nil {
		return TransportApplyStatus{}, fmt.Errorf("AI gateway operations request failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maxOperationsStatusBytes))
		return TransportApplyStatus{}, fmt.Errorf("AI gateway operations returned status %d", response.StatusCode)
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, maxOperationsStatusBytes+1))
	decoder.DisallowUnknownFields()
	var status TransportApplyStatus
	if err := decoder.Decode(&status); err != nil {
		return TransportApplyStatus{}, fmt.Errorf("decode AI gateway operations status")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return TransportApplyStatus{}, fmt.Errorf("AI gateway operations status contains trailing data")
	}
	if status.TenantID != strings.TrimSpace(tenantID) || !strings.EqualFold(status.Environment, environment) || status.DesiredRevision < 0 || status.AppliedRevision < 0 {
		return TransportApplyStatus{}, fmt.Errorf("AI gateway operations status scope is invalid")
	}
	if !validStatusChecksum(status.DesiredChecksum, status.DesiredRevision) || !validStatusChecksum(status.AppliedChecksum, status.AppliedRevision) {
		return TransportApplyStatus{}, fmt.Errorf("AI gateway operations status checksum is invalid")
	}
	status.Environment = environment
	return status, nil
}

func validStatusChecksum(checksum string, revision int64) bool {
	checksum = strings.TrimSpace(checksum)
	if revision == 0 {
		return checksum == ""
	}
	if revision < 1 || len(checksum) != 64 {
		return false
	}
	_, err := parseDigest(checksum)
	return err == nil
}
