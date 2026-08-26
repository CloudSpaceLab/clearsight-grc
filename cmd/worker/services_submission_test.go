package main

import (
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

func TestAssessmentSubmissionConsumerUsesSharedWorkerRepositories(t *testing.T) {
	inbox := workflowruntime.NewMemoryRepository()
	requestRepository := evidence.NewMemoryRepository(nil, nil)
	requests := evidence.NewService(requestRepository, evidence.NewMemoryObjectStore())
	assessments := thirdparty.NewMemoryAssessmentRepository()

	consumer := newAssessmentSubmissionConsumer(inbox, requests, assessments)
	if consumer.Inbox != inbox {
		t.Fatal("consumer does not use the worker inbox repository")
	}
	if consumer.Requests != requests {
		t.Fatal("consumer does not use the worker evidence repository")
	}
	if consumer.Resolver != assessments {
		t.Fatal("consumer does not use the worker assessment repository")
	}
	if _, ok := consumer.Reactions.(*thirdparty.AssessmentService); !ok {
		t.Fatalf("consumer reaction service = %T", consumer.Reactions)
	}
}
