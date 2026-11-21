package sourceaccess

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestMalformedRESTResponseRemainsAnExecutionFailure(t *testing.T) {
	if !errors.Is(errRESTInvalidResponse, ErrExecution) {
		t.Fatal("invalid response must preserve the source execution error contract")
	}
}

func TestQuestionSampleNeverCallsEndpoint(t *testing.T) {
	var calls atomic.Int64
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { calls.Add(1); _, _ = w.Write([]byte(`{"result":true}`)) }))
	defer server.Close()
	tester := newQuestionTesterWithOptions(RESTJSONOptions{Client: server.Client()})
	for _, sample := range []struct {
		body, kind string
		outcome    QuestionTestOutcome
	}{
		{`{"result":false}`, "BOOLEAN", QuestionTestValid}, {`{"result":0}`, "NUMBER", QuestionTestValid},
		{`{"result":true,"extra":1}`, "BOOLEAN", QuestionTestInvalid}, {`{"result":1e999}`, "NUMBER", QuestionTestInvalid},
	} {
		result, err := tester.Test(context.Background(), QuestionTestInput{Mode: "SAMPLE", ResultType: sample.kind, Sample: []byte(sample.body)})
		if err != nil || result.Outcome != sample.outcome {
			t.Fatalf("sample %s: %+v %v", sample.body, result, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("sample contacted endpoint %d times", calls.Load())
	}
}

func TestQuestionTestsRejectDuplicateFields(t *testing.T) {
	for _, sample := range []string{`{"result":false,"result":true}`, `{"result":false,"res\u0075lt":true}`, `{"result":true,"message":"one","message":"two"}`, "{\"result\":true,\"message\":\"\xff\"}"} {
		result, err := NewQuestionTester().Test(context.Background(), QuestionTestInput{Mode: "SAMPLE", ResultType: "BOOLEAN", Sample: []byte(sample)})
		if err != nil || result.Outcome != QuestionTestInvalid {
			t.Fatalf("ambiguous sample accepted: %+v %v", result, err)
		}
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":false,"result":true}`))
	}))
	defer server.Close()
	result, err := newQuestionTesterWithOptions(RESTJSONOptions{Client: server.Client()}).Test(context.Background(), QuestionTestInput{Mode: "LIVE", ResultType: "BOOLEAN", Endpoint: server.URL})
	if err != nil || result.Outcome != QuestionTestInvalid {
		t.Fatalf("ambiguous live response accepted: %+v %v", result, err)
	}
}

func TestQuestionLiveChecksOnceWithoutReturningResponse(t *testing.T) {
	var calls atomic.Int64
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":true,"message":"sensitive internal details"}`))
	}))
	defer server.Close()
	tester := newQuestionTesterWithOptions(RESTJSONOptions{Client: server.Client()})
	result, err := tester.Test(context.Background(), QuestionTestInput{Mode: "LIVE", ResultType: "BOOLEAN", Endpoint: server.URL + "/check"})
	if err != nil || result.Outcome != QuestionTestValid || result.CheckedAt.IsZero() || calls.Load() != 1 {
		t.Fatalf("live check: %+v %v calls=%d", result, err, calls.Load())
	}
	if strings.Contains(result.Reason, "sensitive") {
		t.Fatal("raw endpoint response leaked")
	}
}

func TestQuestionLiveCheckedAtReflectsCompletedResponse(t *testing.T) {
	started := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	completed := started.Add(5 * time.Second)
	var responded atomic.Bool
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		responded.Store(true)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":true}`))
	}))
	defer server.Close()
	tester := newQuestionTesterWithOptions(RESTJSONOptions{Client: server.Client()})
	tester.now = func() time.Time {
		if responded.Load() {
			return completed
		}
		return started
	}
	result, err := tester.Test(context.Background(), QuestionTestInput{Mode: "LIVE", ResultType: "BOOLEAN", Endpoint: server.URL})
	if err != nil || result.Outcome != QuestionTestValid || !result.CheckedAt.Equal(completed) {
		t.Fatalf("checked_at should reflect completed response: %+v %v", result, err)
	}
}

func TestQuestionLiveRejectsCredentialsQueriesAndFragmentsBeforeCalling(t *testing.T) {
	var calls atomic.Int64
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { calls.Add(1); w.WriteHeader(http.StatusOK) }))
	defer server.Close()
	tester := newQuestionTesterWithOptions(RESTJSONOptions{Client: server.Client()})
	for _, endpoint := range []string{"http://bank.example/status", "https://user:secret@bank.example/status", server.URL + "/status?token=secret", server.URL + "/status#secret"} {
		if _, err := tester.Test(context.Background(), QuestionTestInput{Mode: "LIVE", ResultType: "BOOLEAN", Endpoint: endpoint}); err == nil {
			t.Fatalf("unsafe endpoint accepted")
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("rejected endpoint was contacted")
	}
}

func TestQuestionLiveReportsInvalidContractAndUnavailableService(t *testing.T) {
	var response = `{"result":"true"}`
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}))
	tester := newQuestionTesterWithOptions(RESTJSONOptions{Client: server.Client()})
	result, err := tester.Test(context.Background(), QuestionTestInput{Mode: "LIVE", ResultType: "BOOLEAN", Endpoint: server.URL})
	if err != nil || result.Outcome != QuestionTestInvalid {
		t.Fatalf("invalid response: %+v %v", result, err)
	}
	server.Close()
	result, err = tester.Test(context.Background(), QuestionTestInput{Mode: "LIVE", ResultType: "BOOLEAN", Endpoint: server.URL})
	if err != nil || result.Outcome != QuestionTestUnavailable {
		t.Fatalf("unavailable response: %+v %v", result, err)
	}
}

func TestQuestionLiveDoesNotFollowRedirectsOrReturnLargeBodies(t *testing.T) {
	var targetCalls atomic.Int64
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		targetCalls.Add(1)
		_, _ = w.Write([]byte(`{"result":true}`))
	}))
	defer target.Close()
	redirect := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, http.StatusFound) }))
	defer redirect.Close()
	result, err := newQuestionTesterWithOptions(RESTJSONOptions{Client: redirect.Client()}).Test(context.Background(), QuestionTestInput{Mode: "LIVE", ResultType: "BOOLEAN", Endpoint: redirect.URL})
	if err != nil || result.Outcome != QuestionTestUnavailable || targetCalls.Load() != 0 {
		t.Fatalf("redirect: %+v %v", result, err)
	}
	large := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":true,"message":"` + strings.Repeat("x", questionTestMaxBytes) + `"}`))
	}))
	defer large.Close()
	result, err = newQuestionTesterWithOptions(RESTJSONOptions{Client: large.Client()}).Test(context.Background(), QuestionTestInput{Mode: "LIVE", ResultType: "BOOLEAN", Endpoint: large.URL})
	if err != nil || result.Outcome != QuestionTestUnavailable {
		t.Fatalf("oversized response: %+v %v", result, err)
	}
}

func TestQuestionLiveRespectsCancellation(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write([]byte(`{"result":true}`))
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	result, err := newQuestionTesterWithOptions(RESTJSONOptions{Client: server.Client()}).Test(ctx, QuestionTestInput{Mode: "LIVE", ResultType: "BOOLEAN", Endpoint: server.URL})
	if err != nil || result.Outcome != QuestionTestUnavailable {
		t.Fatalf("cancelled: %+v %v", result, err)
	}
}

func TestQuestionLiveDefaultTransportBlocksPrivateEndpoint(t *testing.T) {
	var calls atomic.Int64
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":true}`))
	}))
	defer server.Close()
	result, err := NewQuestionTester().Test(context.Background(), QuestionTestInput{Mode: "LIVE", ResultType: "BOOLEAN", Endpoint: server.URL})
	if err != nil || result.Outcome != QuestionTestUnavailable || calls.Load() != 0 {
		t.Fatalf("private endpoint contacted: %+v %v calls=%d", result, err, calls.Load())
	}
}

func TestQuestionLiveDistinguishesMalformedResponseFromUnavailableEndpoint(t *testing.T) {
	for _, tc := range []struct {
		name, contentType, body string
		status                  int
		want                    QuestionTestOutcome
	}{
		{"malformed JSON", "application/json", `{"result":`, 200, QuestionTestInvalid},
		{"trailing JSON", "application/json", `{"result":true} {}`, 200, QuestionTestInvalid},
		{"wrong content type", "text/html", `<p>secret</p>`, 200, QuestionTestInvalid},
		{"request rejected", "application/json", `{"error":"secret"}`, 400, QuestionTestUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", tc.contentType)
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			result, err := newQuestionTesterWithOptions(RESTJSONOptions{Client: server.Client()}).Test(context.Background(), QuestionTestInput{Mode: "LIVE", ResultType: "BOOLEAN", Endpoint: server.URL})
			if err != nil || result.Outcome != tc.want || strings.Contains(result.Reason, "secret") {
				t.Fatalf("outcome=%+v err=%v", result, err)
			}
		})
	}
}
