package sourceaccess

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
)

const (
	hardMaxRESTPathBytes       = 2048
	hardMaxRESTPointerBytes    = 1024
	hardMaxRESTQueryParams     = 32
	hardMaxRESTQueryValueBytes = 4096
)

type RESTJSONPaginationMode string

const (
	RESTJSONPaginationNone   RESTJSONPaginationMode = "NONE"
	RESTJSONPaginationCursor RESTJSONPaginationMode = "CURSOR"
	RESTJSONPaginationETag   RESTJSONPaginationMode = "ETAG"
)

type RESTJSONPagination struct {
	Mode                  RESTJSONPaginationMode `json:"mode"`
	CursorQueryParam      string                 `json:"cursor_query_param,omitempty"`
	NextCursorPointer     string                 `json:"next_cursor_pointer,omitempty"`
	PageSizeQueryParam    string                 `json:"page_size_query_param,omitempty"`
}

type RESTJSONLookup struct {
	Path       string `json:"path,omitempty"`
	QueryParam string `json:"query_param"`
}

type RESTJSONViewDefinition struct {
	Path           string            `json:"path"`
	RecordsPointer string            `json:"records_pointer,omitempty"`
	FixedQuery     map[string]string `json:"fixed_query,omitempty"`
	Pagination     RESTJSONPagination `json:"pagination"`
	Lookup         *RESTJSONLookup   `json:"lookup,omitempty"`
}

func normalizeRESTJSONConnectionDefinition(raw json.RawMessage, secretRef string) (json.RawMessage, error) {
	var definition RESTJSONConnectionDefinition
	if err := strictJSON(raw, &definition); err != nil {
		return nil, fmt.Errorf("%w: REST/JSON connection definition is invalid", ErrDefinitionInvalid)
	}
	baseURL, err := normalizeRESTBaseURL(definition.BaseURL)
	if err != nil {
		return nil, err
	}
	definition.BaseURL = baseURL
	if definition.Authentication.Kind == "" {
		definition.Authentication.Kind = RESTJSONAuthNone
	}
	definition.Authentication.HeaderName = strings.TrimSpace(definition.Authentication.HeaderName)
	switch definition.Authentication.Kind {
	case RESTJSONAuthNone:
		if definition.Authentication.HeaderName != "" || strings.TrimSpace(secretRef) != "" {
			return nil, fmt.Errorf("%w: unauthenticated REST connections cannot carry authentication configuration", ErrDefinitionInvalid)
		}
	case RESTJSONAuthBearer:
		if definition.Authentication.HeaderName != "" || strings.TrimSpace(secretRef) == "" {
			return nil, fmt.Errorf("%w: bearer REST connections require only an opaque secret reference", ErrDefinitionInvalid)
		}
	case RESTJSONAuthHeader:
		if !validRESTHeaderName(definition.Authentication.HeaderName) || strings.TrimSpace(secretRef) == "" {
			return nil, fmt.Errorf("%w: header-auth REST connections require a safe header name and opaque secret reference", ErrDefinitionInvalid)
		}
		definition.Authentication.HeaderName = textproto.CanonicalMIMEHeaderKey(definition.Authentication.HeaderName)
	default:
		return nil, fmt.Errorf("%w: REST authentication kind is unsupported", ErrDefinitionInvalid)
	}
	encoded, err := json.Marshal(definition)
	if err != nil || len(encoded) > HardMaxDefinitionBytes {
		return nil, ErrLimitExceeded
	}
	return encoded, nil
}

func decodeRESTJSONConnection(connection Connection) (RESTJSONConnectionDefinition, error) {
	normalized, err := normalizeRESTJSONConnectionDefinition(connection.Definition, connection.SecretRef)
	if err != nil {
		return RESTJSONConnectionDefinition{}, err
	}
	var definition RESTJSONConnectionDefinition
	if err := json.Unmarshal(normalized, &definition); err != nil {
		return RESTJSONConnectionDefinition{}, ErrDefinitionInvalid
	}
	return definition, nil
}

func normalizeRESTJSONViewDefinition(raw json.RawMessage) (json.RawMessage, error) {
	var definition RESTJSONViewDefinition
	if err := strictJSON(raw, &definition); err != nil {
		return nil, fmt.Errorf("%w: REST/JSON view definition is invalid", ErrDefinitionInvalid)
	}
	path, err := normalizeRESTPath(definition.Path)
	if err != nil {
		return nil, err
	}
	definition.Path = path
	if err := validateJSONPointer(definition.RecordsPointer); err != nil {
		return nil, err
	}
	if len(definition.FixedQuery) > hardMaxRESTQueryParams {
		return nil, ErrLimitExceeded
	}
	fixed := make(map[string]string, len(definition.FixedQuery))
	for key, value := range definition.FixedQuery {
		if !validRESTQueryParam(key) || len(value) > hardMaxRESTQueryValueBytes || containsControl(value) {
			return nil, fmt.Errorf("%w: REST fixed query is invalid", ErrDefinitionInvalid)
		}
		fixed[key] = value
	}
	definition.FixedQuery = fixed

	if definition.Pagination.Mode == "" {
		definition.Pagination.Mode = RESTJSONPaginationNone
	}
	definition.Pagination.CursorQueryParam = strings.TrimSpace(definition.Pagination.CursorQueryParam)
	definition.Pagination.PageSizeQueryParam = strings.TrimSpace(definition.Pagination.PageSizeQueryParam)
	if definition.Pagination.PageSizeQueryParam != "" && !validRESTQueryParam(definition.Pagination.PageSizeQueryParam) {
		return nil, fmt.Errorf("%w: REST page-size query parameter is invalid", ErrDefinitionInvalid)
	}
	switch definition.Pagination.Mode {
	case RESTJSONPaginationNone:
		if definition.Pagination.CursorQueryParam != "" || definition.Pagination.NextCursorPointer != "" {
			return nil, fmt.Errorf("%w: non-paginated REST views cannot define a cursor", ErrDefinitionInvalid)
		}
	case RESTJSONPaginationCursor:
		if !validRESTQueryParam(definition.Pagination.CursorQueryParam) || definition.Pagination.NextCursorPointer == "" {
			return nil, fmt.Errorf("%w: cursor pagination requires a query parameter and next-cursor pointer", ErrDefinitionInvalid)
		}
		if err := validateJSONPointer(definition.Pagination.NextCursorPointer); err != nil {
			return nil, err
		}
	case RESTJSONPaginationETag:
		if definition.Pagination.CursorQueryParam != "" || definition.Pagination.NextCursorPointer != "" {
			return nil, fmt.Errorf("%w: ETag pagination uses HTTP headers rather than cursor fields", ErrDefinitionInvalid)
		}
	default:
		return nil, fmt.Errorf("%w: REST pagination mode is unsupported", ErrDefinitionInvalid)
	}

	dynamic := map[string]struct{}{}
	for _, key := range []string{definition.Pagination.CursorQueryParam, definition.Pagination.PageSizeQueryParam} {
		if key == "" {
			continue
		}
		if _, exists := fixed[key]; exists {
			return nil, fmt.Errorf("%w: dynamic REST query parameter collides with fixed query", ErrDefinitionInvalid)
		}
		if _, exists := dynamic[key]; exists {
			return nil, fmt.Errorf("%w: REST query parameters must be distinct", ErrDefinitionInvalid)
		}
		dynamic[key] = struct{}{}
	}

	if definition.Lookup != nil {
		definition.Lookup.QueryParam = strings.TrimSpace(definition.Lookup.QueryParam)
		if !validRESTQueryParam(definition.Lookup.QueryParam) {
			return nil, fmt.Errorf("%w: REST lookup query parameter is invalid", ErrDefinitionInvalid)
		}
		if definition.Lookup.Path == "" {
			definition.Lookup.Path = definition.Path
		} else {
			lookupPath, err := normalizeRESTPath(definition.Lookup.Path)
			if err != nil {
				return nil, err
			}
			definition.Lookup.Path = lookupPath
		}
		if _, exists := fixed[definition.Lookup.QueryParam]; exists {
			return nil, fmt.Errorf("%w: lookup parameter collides with fixed REST query", ErrDefinitionInvalid)
		}
		if _, exists := dynamic[definition.Lookup.QueryParam]; exists {
			return nil, fmt.Errorf("%w: lookup parameter collides with pagination", ErrDefinitionInvalid)
		}
	}

	encoded, err := json.Marshal(definition)
	if err != nil || len(encoded) > HardMaxDefinitionBytes {
		return nil, ErrLimitExceeded
	}
	return encoded, nil
}

func decodeRESTJSONView(view View) (RESTJSONViewDefinition, error) {
	normalized, err := normalizeRESTJSONViewDefinition(view.Definition)
	if err != nil {
		return RESTJSONViewDefinition{}, err
	}
	var definition RESTJSONViewDefinition
	if err := json.Unmarshal(normalized, &definition); err != nil {
		return RESTJSONViewDefinition{}, ErrDefinitionInvalid
	}
	return definition, nil
}

func strictJSON(raw json.RawMessage, target any) error {
	if len(raw) == 0 || len(raw) > HardMaxDefinitionBytes || !json.Valid(raw) {
		return ErrDefinitionInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return ErrDefinitionInvalid
	}
	return nil
}

func normalizeRESTBaseURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > hardMaxRESTPathBytes || containsControl(value) {
		return "", fmt.Errorf("%w: bounded REST base URL is required", ErrDefinitionInvalid)
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.RawPath != "" {
		return "", fmt.Errorf("%w: REST base URL must be a fixed HTTPS origin without credentials, query or fragment", ErrDefinitionInvalid)
	}
	if strings.Contains(parsed.Path, "\\") || hasDotPathSegment(parsed.Path) {
		return "", fmt.Errorf("%w: REST base URL path is invalid", ErrDefinitionInvalid)
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return parsed.String(), nil
}

func normalizeRESTPath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > hardMaxRESTPathBytes || !strings.HasPrefix(value, "/") || containsControl(value) || strings.Contains(value, "\\") {
		return "", fmt.Errorf("%w: bounded absolute REST path is required", ErrDefinitionInvalid)
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.RawPath != "" || hasDotPathSegment(parsed.Path) {
		return "", fmt.Errorf("%w: REST path cannot contain origin, query, fragment or traversal", ErrDefinitionInvalid)
	}
	return parsed.Path, nil
}

func hasDotPathSegment(value string) bool {
	for _, segment := range strings.Split(value, "/") {
		decoded, err := url.PathUnescape(segment)
		if err != nil || decoded == "." || decoded == ".." {
			return true
		}
	}
	return false
}

func validRESTHeaderName(value string) bool {
	value = strings.TrimSpace(value)
	canonical := textproto.CanonicalMIMEHeaderKey(value)
	if value == "" || canonical == "" || len(value) > 128 || containsControl(value) {
		return false
	}
	switch canonical {
	case "Authorization", "Proxy-Authorization", "Host", "Cookie", "Set-Cookie", "Connection", "Content-Length", "Transfer-Encoding", "Accept", "If-None-Match":
		return false
	default:
		return true
	}
}

func validRESTQueryParam(value string) bool {
	if value == "" || len(value) > 128 || containsControl(value) {
		return false
	}
	for _, character := range value {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') || strings.ContainsRune("._~-", character) {
			continue
		}
		return false
	}
	return true
}

func validateJSONPointer(pointer string) error {
	if len(pointer) > hardMaxRESTPointerBytes || containsControl(pointer) {
		return fmt.Errorf("%w: JSON pointer is invalid", ErrDefinitionInvalid)
	}
	if pointer == "" {
		return nil
	}
	if !strings.HasPrefix(pointer, "/") {
		return fmt.Errorf("%w: JSON pointer must be empty or start with slash", ErrDefinitionInvalid)
	}
	for _, token := range strings.Split(pointer[1:], "/") {
		for index := 0; index < len(token); index++ {
			if token[index] != '~' {
				continue
			}
			if index+1 >= len(token) || (token[index+1] != '0' && token[index+1] != '1') {
				return fmt.Errorf("%w: JSON pointer escape is invalid", ErrDefinitionInvalid)
			}
			index++
		}
	}
	return nil
}

func restSameOrigin(left, right *url.URL) bool {
	if left == nil || right == nil {
		return false
	}
	return strings.EqualFold(left.Scheme, right.Scheme) && strings.EqualFold(left.Host, right.Host)
}

func restContentTypeJSON(response *http.Response) bool {
	if response == nil {
		return false
	}
	value := strings.ToLower(strings.TrimSpace(response.Header.Get("Content-Type")))
	if semicolon := strings.Index(value, ";"); semicolon >= 0 {
		value = strings.TrimSpace(value[:semicolon])
	}
	return value == "application/json" || strings.HasSuffix(value, "+json")
}
