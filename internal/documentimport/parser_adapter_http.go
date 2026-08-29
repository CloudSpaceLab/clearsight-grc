package documentimport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type HTTPParserAdapter struct {
	name     string
	endpoint string
	client   *http.Client
}

func NewHTTPParserAdapter(name, endpoint string, client *http.Client) (*HTTPParserAdapter, error) {
	name = strings.TrimSpace(name)
	endpoint = strings.TrimSpace(endpoint)
	parsed, err := url.Parse(endpoint)
	if name == "" || err != nil || !parsed.IsAbs() || parsed.Hostname() == "" || parsed.User != nil || parsed.Fragment != "" {
		return nil, errors.New("parser adapter name and absolute endpoint are required")
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &HTTPParserAdapter{name: name, endpoint: endpoint, client: client}, nil
}

func (a *HTTPParserAdapter) Name() string { return a.name }

func (a *HTTPParserAdapter) Extract(ctx context.Context, request ParserRequest) (ParserResponse, error) {
	if a == nil || a.client == nil || strings.TrimSpace(a.endpoint) == "" || request.Data == nil {
		return ParserResponse{}, errors.New("parser adapter is not configured")
	}
	inputLimit := request.MaxBytes
	if inputLimit <= 0 {
		inputLimit = 20 << 20
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint, io.LimitReader(request.Data, inputLimit+1))
	if err != nil {
		return ParserResponse{}, err
	}
	httpRequest.Header.Set("Content-Type", "application/octet-stream")
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("X-ClearSight-Artifact-ID", request.ArtifactID)
	httpRequest.Header.Set("X-ClearSight-File-Name", request.FileName)
	httpRequest.Header.Set("X-ClearSight-Media-Type", request.MediaType)
	httpRequest.Header.Set("X-ClearSight-Max-Bytes", strconv.FormatInt(request.MaxBytes, 10))
	httpRequest.Header.Set("X-ClearSight-Max-Pages", strconv.Itoa(request.MaxPages))
	httpRequest.Header.Set("X-ClearSight-Deadline", request.Deadline.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"))

	response, err := a.client.Do(httpRequest)
	if err != nil {
		return ParserResponse{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return ParserResponse{}, fmt.Errorf("parser adapter returned HTTP %d", response.StatusCode)
	}
	limit := request.MaxBytes
	if limit <= 0 || limit > 16<<20 {
		limit = 16 << 20
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return ParserResponse{}, err
	}
	if int64(len(body)) > limit {
		return ParserResponse{}, errors.Join(ErrParserAdapterInvalid, errors.New("parser adapter response body exceeded the configured limit"))
	}
	var decoded ParserResponse
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return ParserResponse{}, errors.Join(ErrParserAdapterInvalid, err)
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return ParserResponse{}, errors.Join(ErrParserAdapterInvalid, errors.New("parser adapter response contains trailing JSON"))
	}
	return decoded, nil
}

var _ ParserAdapter = (*HTTPParserAdapter)(nil)
