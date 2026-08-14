package onboarding

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestOnboardingStateVersioning(t *testing.T) {
	service := NewService(NewMemoryRepository())
	initial, err := service.State(context.Background(), "bank-demo", "user-demo", "reviewer-first-run")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.Update(context.Background(), initial.TenantID, initial.PrincipalID, initial.GuideCode, UpdateInput{CurrentStep: 1, ExpectedVersion: initial.Version})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != 1 || updated.CurrentStep != 1 {
		t.Fatalf("unexpected state: %#v", updated)
	}
	_, err = service.Update(context.Background(), initial.TenantID, initial.PrincipalID, initial.GuideCode, UpdateInput{CurrentStep: 2, ExpectedVersion: 0})
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected version conflict, got %v", err)
	}
}

func TestGuideCannotBeCompletedAndDismissed(t *testing.T) {
	service := NewService(NewMemoryRepository())
	guide, err := service.Guide("", "reviewer-first-run")
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Update(context.Background(), "bank-demo", "user-demo", guide.Code, UpdateInput{CurrentStep: len(guide.Steps), Completed: true, Dismissed: true, ExpectedVersion: 0})
	if err == nil {
		t.Fatal("expected invalid terminal state")
	}
}

func TestCompletedGuideMustReachFinalStep(t *testing.T) {
	service := NewService(NewMemoryRepository())
	_, err := service.Update(context.Background(), "bank-demo", "user-demo", "reviewer-first-run", UpdateInput{CurrentStep: 2, Completed: true, ExpectedVersion: 0})
	if err == nil {
		t.Fatal("expected incomplete guide completion to fail")
	}
}

func TestGuideResolutionUsesRolePriorityAndFallback(t *testing.T) {
	service := NewService(NewMemoryRepository())
	guide, err := service.ResolveRoles([]string{"program owner", "cro"})
	if err != nil {
		t.Fatal(err)
	}
	if guide.Code != "executive-first-run" {
		t.Fatalf("expected executive priority, got %s", guide.Code)
	}
	guide, err = service.ResolveRoles(nil)
	if err != nil || guide.Code != "general-first-run" {
		t.Fatalf("expected general fallback, guide=%#v err=%v", guide, err)
	}
}

func TestDemoGuidesAvoidScriptedTourCopy(t *testing.T) {
	blocked := []string{
		"open one material record",
		"open your ongoing responsibility",
		"resolve the smallest evidence gap",
		"use the reason, not only the colour",
		"see the bank from a real role",
	}
	for _, guide := range DemoGuides() {
		visible := guide.Title + " " + guide.Description
		for _, step := range guide.Steps {
			visible += " " + step.Title + " " + step.Description + " " + step.Action
		}
		visible = strings.ToLower(visible)
		for _, phrase := range blocked {
			if strings.Contains(visible, phrase) {
				t.Fatalf("guide %q contains scripted copy %q", guide.Code, phrase)
			}
		}
	}
}
