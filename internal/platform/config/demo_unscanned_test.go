package config

import (
	"strings"
	"testing"
)

func TestDemoUnscannedSettingRejectsOutsideDemo(t *testing.T) {
	for _, environment := range []string{"development", "production"} {
		t.Run(environment, func(t *testing.T) {
			t.Setenv("CLEARSIGHT_ENV", environment)
			t.Setenv("CLEARSIGHT_DEMO_MODE", "false")
			t.Setenv("CLEARSIGHT_DEMO_ALLOW_UNSCANNED_ARTIFACTS", "true")
			t.Setenv("CLEARSIGHT_IDENTITY_MODE", "signed")
			t.Setenv("CLEARSIGHT_IDENTITY_HMAC_SECRET", strings.Repeat("s", 32))
			t.Setenv("CLEARSIGHT_COMMAND_AUTHORIZATION", "enforce")
			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), "CLEARSIGHT_DEMO_ALLOW_UNSCANNED_ARTIFACTS") {
				t.Fatalf("unscanned acceptance outside demo: %v", err)
			}
		})
	}
}

func TestDemoUnscannedSettingDefaultsAndOverride(t *testing.T) {
	for _, test := range []struct {
		demo, flag string
		want       bool
	}{
		{"true", "", true}, {"true", "false", false}, {"true", "true", true}, {"false", "", false},
	} {
		t.Run(test.demo+"/"+test.flag, func(t *testing.T) {
			t.Setenv("CLEARSIGHT_ENV", "development")
			t.Setenv("CLEARSIGHT_DEMO_MODE", test.demo)
			t.Setenv("CLEARSIGHT_DEMO_ALLOW_UNSCANNED_ARTIFACTS", test.flag)
			cfg, err := Load()
			if err != nil || cfg.DemoAllowUnscannedArtifacts != test.want {
				t.Fatalf("allowed=%v want=%v err=%v", cfg.DemoAllowUnscannedArtifacts, test.want, err)
			}
		})
	}
	t.Setenv("CLEARSIGHT_DEMO_ALLOW_UNSCANNED_ARTIFACTS", "sometimes")
	if _, err := Load(); err == nil {
		t.Fatal("invalid boolean accepted")
	}
}
