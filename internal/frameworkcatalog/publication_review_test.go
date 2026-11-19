package frameworkcatalog

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestPublicationReviewIndependentAndBoundToSource(t *testing.T) {
	manifest, source := fixtureReaders(t)
	now := time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)
	draft, err := PreparePublicationReview(manifest, source, "maker", now)
	if err != nil {
		t.Fatal(err)
	}
	if draft.Status != PublicationReviewDraft || draft.Version != 1 || draft.ReleaseSHA256 == "" || draft.SourceBytes == 0 {
		t.Fatalf("invalid draft: %+v", draft)
	}
	if _, err = SubmitPublicationReview(draft, "other", 1); !errors.Is(err, ErrPublicationReview) {
		t.Fatalf("other submitter accepted: %v", err)
	}
	if _, err = SubmitPublicationReview(draft, "maker", 2); !errors.Is(err, ErrPublicationReview) {
		t.Fatalf("stale submit accepted: %v", err)
	}
	pending, err := SubmitPublicationReview(draft, "maker", 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = DecidePublicationReview(pending, "maker", "self review", 2, true, now); !errors.Is(err, ErrPublicationReview) {
		t.Fatalf("self-review accepted: %v", err)
	}
	if _, err = DecidePublicationReview(pending, "checker", "", 2, true, now); !errors.Is(err, ErrPublicationReview) {
		t.Fatalf("empty decision rationale accepted: %v", err)
	}
	if _, err = DecidePublicationReview(pending, "checker", "reviewed", 1, true, now); !errors.Is(err, ErrPublicationReview) {
		t.Fatalf("stale review accepted: %v", err)
	}
	changed := pending
	changed.Publication = pending.Publication
	changed.Publication.Title = "Changed after submission"
	if _, err = DecidePublicationReview(changed, "checker", "reviewed", 2, true, now); !errors.Is(err, ErrPublicationReview) {
		t.Fatalf("changed release accepted: %v", err)
	}
	approved, err := DecidePublicationReview(pending, "checker", "Source and citations reviewed", 2, true, now.Add(time.Second))
	if err != nil || approved.Status != PublicationReviewApproved || approved.ReviewerID != "checker" || approved.Version != 3 {
		t.Fatalf("bad review: %+v %v", approved, err)
	}
	if _, err = DecidePublicationReview(approved, "checker", "again", 3, true, now); !errors.Is(err, ErrPublicationReview) {
		t.Fatalf("repeat decision accepted: %v", err)
	}
}

func TestPublicationReviewRejectionAndSourceValidation(t *testing.T) {
	manifest, source := fixtureReaders(t)
	now := time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)
	draft, err := PreparePublicationReview(manifest, source, "maker", now)
	if err != nil {
		t.Fatal(err)
	}
	changed := draft
	changed.Publication.SourceSHA256 = "0000000000000000000000000000000000000000000000000000000000000000"
	if _, err := SubmitPublicationReview(changed, "maker", 1); !errors.Is(err, ErrPublicationReview) {
		t.Fatalf("changed source hash accepted: %v", err)
	}
	pending, err := SubmitPublicationReview(draft, "maker", 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecidePublicationReview(pending, "checker", "reviewed", 2, false, now.Add(-time.Second)); !errors.Is(err, ErrPublicationReview) {
		t.Fatalf("earlier decision accepted: %v", err)
	}
	rejected, err := DecidePublicationReview(pending, "checker", "Source needs correction", 2, false, now.Add(time.Second))
	if err != nil || rejected.Status != PublicationReviewRejected {
		t.Fatalf("rejection: %+v, %v", rejected, err)
	}
	if _, err := SubmitPublicationReview(rejected, "maker", 3); !errors.Is(err, ErrPublicationReview) {
		t.Fatalf("decided review resubmitted: %v", err)
	}
}

func TestPublicationReviewRejectsWrongSource(t *testing.T) {
	manifest, _ := fixtureReaders(t)
	if _, err := PreparePublicationReview(manifest, strings.NewReader("different source"), "maker", time.Now()); !errors.Is(err, ErrInvalidPublication) {
		t.Fatalf("unverified source accepted: %v", err)
	}
}
