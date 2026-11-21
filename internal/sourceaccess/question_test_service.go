package sourceaccess

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"strings"
	"time"
)

// QuestionTester is a stateless building block. It neither creates catalog
// records nor records observations or assessment outcomes. It is not a
// substitute for the governed question-binding lifecycle.
type QuestionTester struct {
	adapter RESTJSONAdapter
	now     func() time.Time
}
type QuestionTestOutcome string

const (
	QuestionTestValid       QuestionTestOutcome = "VALID"
	QuestionTestInvalid     QuestionTestOutcome = "INVALID"
	QuestionTestUnavailable QuestionTestOutcome = "UNAVAILABLE"
	questionTestMaxBytes                        = 65536
)

type QuestionTestInput struct {
	Mode       string
	ResultType string
	Endpoint   string
	Sample     json.RawMessage
}

type QuestionTestResult struct {
	Mode       string              `json:"mode"`
	Outcome    QuestionTestOutcome `json:"outcome"`
	ResultType string              `json:"result_type"`
	CheckedAt  time.Time           `json:"checked_at"`
	Reason     string              `json:"reason,omitempty"`
}

func NewQuestionTester() *QuestionTester {
	return newQuestionTesterWithOptions(RestrictedQuestionTestOptions())
}

// Injectable clients are intentionally confined to tests in this package.
func newQuestionTesterWithOptions(options RESTJSONOptions) *QuestionTester {
	return &QuestionTester{adapter: NewRESTJSONAdapter(options), now: time.Now}
}

func (t *QuestionTester) Test(ctx context.Context, input QuestionTestInput) (QuestionTestResult, error) {
	if t == nil || t.now == nil || (input.ResultType != "BOOLEAN" && input.ResultType != "NUMBER") {
		return QuestionTestResult{}, ErrDefinitionInvalid
	}
	result := QuestionTestResult{Mode: input.Mode, ResultType: input.ResultType, CheckedAt: t.now().UTC()}
	if input.Mode == "SAMPLE" {
		if input.Endpoint != "" || len(input.Sample) == 0 || len(input.Sample) > questionTestMaxBytes {
			return QuestionTestResult{}, ErrDefinitionInvalid
		}
		decoder := json.NewDecoder(bytes.NewReader(input.Sample))
		decoder.UseNumber()
		var value, trailing any
		if decoder.Decode(&value) != nil || decoder.Decode(&trailing) != io.EOF || validateRESTQuestionPayload(input.Sample, value, input.ResultType) != nil {
			result.Outcome = QuestionTestInvalid
			result.Reason = "Response does not match the selected result format."
			return result, nil
		}
		result.Outcome = QuestionTestValid
		return result, nil
	}
	if input.Mode != "LIVE" || len(input.Sample) != 0 {
		return QuestionTestResult{}, ErrDefinitionInvalid
	}
	endpoint, err := url.Parse(input.Endpoint)
	if err != nil || endpoint.Scheme != "https" || endpoint.Hostname() == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.ForceQuery || endpoint.Fragment != "" || endpoint.RawPath != "" || strings.Contains(input.Endpoint, "#") {
		return QuestionTestResult{}, ErrDefinitionInvalid
	}
	path := endpoint.Path
	if path == "" {
		path = "/"
	}
	if _, err := normalizeRESTPath(path); err != nil {
		return QuestionTestResult{}, ErrDefinitionInvalid
	}
	origin := (&url.URL{Scheme: endpoint.Scheme, Host: endpoint.Host}).String()
	definition, err := json.Marshal(RESTJSONConnectionDefinition{BaseURL: origin, Authentication: RESTJSONAuthentication{Kind: RESTJSONAuthNone}})
	if err != nil {
		return QuestionTestResult{}, ErrDefinitionInvalid
	}
	connection := Connection{ID: "test", SourceID: "test", Version: "1", AdapterKind: AdapterRESTJSON, AdapterVersion: RESTJSONAdapterVersion, Definition: definition}
	opened, err := t.adapter.Open(ctx, connection, nil)
	if err != nil {
		return QuestionTestResult{}, err
	}
	defer opened.Close()
	session, ok := opened.(*RESTJSONSession)
	if !ok {
		return QuestionTestResult{}, ErrConnection
	}
	bounded, cancel := context.WithTimeout(ctx, DefaultInspectTimeout)
	defer cancel()
	response, err := session.getJSON(bounded, path, url.Values{}, "", questionTestMaxBytes)
	result.CheckedAt = t.now().UTC()
	if err != nil {
		if errors.Is(err, errRESTInvalidResponse) {
			result.Outcome = QuestionTestInvalid
			result.Reason = "Response does not match the selected result format."
			return result, nil
		}
		result.Outcome = QuestionTestUnavailable
		result.Reason = "Endpoint test unavailable."
		return result, nil
	}
	if validateRESTQuestionPayload(response.payload, response.root, input.ResultType) != nil {
		result.Outcome = QuestionTestInvalid
		result.Reason = "Response does not match the selected result format."
		return result, nil
	}
	result.Outcome = QuestionTestValid
	return result, nil
}
