package bot

import (
	"strings"
	"testing"
	"time"
)

func testConfigDurationValue(d time.Duration) ConfigDuration {
	return ConfigDuration{duration: d, set: true}
}

func TestResolveTimeOutThresholdsExplicitZeroDisablesDefaults(t *testing.T) {
	defaults := runtimeTimeOutThresholds{
		Warn: 7 * time.Minute,
		Kill: 14 * time.Minute,
	}
	overrides := TimeOutThresholds{
		Warn: testConfigDurationValue(0),
		Kill: testConfigDurationValue(0),
	}

	resolved := resolveTimeOutThresholds(defaults, overrides)
	if resolved.Warn != 0 || resolved.Kill != 0 {
		t.Fatalf("resolveTimeOutThresholds() = %+v, want both thresholds disabled", resolved)
	}
}

func TestResolveTimeOutThresholdsUsesExplicitOverrides(t *testing.T) {
	defaults := runtimeTimeOutThresholds{
		Warn: 7 * time.Minute,
		Kill: 14 * time.Minute,
	}
	overrides := TimeOutThresholds{
		Warn: testConfigDurationValue(90 * time.Second),
	}

	resolved := resolveTimeOutThresholds(defaults, overrides)
	if resolved.Warn != 90*time.Second {
		t.Fatalf("resolved Warn = %v, want %v", resolved.Warn, 90*time.Second)
	}
	if resolved.Kill != defaults.Kill {
		t.Fatalf("resolved Kill = %v, want default %v", resolved.Kill, defaults.Kill)
	}
}

func TestValidateRuntimeTimeOutThresholdsRejectsEffectiveKillNotGreaterThanWarn(t *testing.T) {
	err := validateRuntimeTimeOutThresholds("task 'slow' effective TimeOuts", runtimeTimeOutThresholds{
		Warn: 3 * time.Minute,
		Kill: 2 * time.Minute,
	})
	if err == nil {
		t.Fatalf("expected invalid effective timeout validation failure")
	}
	if !strings.Contains(err.Error(), "Kill must be greater than Warn") {
		t.Fatalf("unexpected validation error: %v", err)
	}
}
