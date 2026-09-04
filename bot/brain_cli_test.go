package bot

import (
	"strings"
	"testing"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

func TestPullBrainExcludesCloudInstanceLock(t *testing.T) {
	payload := []byte("memory")
	remote := newTestRemote(map[string]robot.RemoteBrainRecord{
		"alpha": {
			Key:       "alpha",
			Payload:   payload,
			Format:    brainCacheFormat,
			Version:   7,
			Checksum:  checksumBytes(payload),
			UpdatedAt: time.Now().UTC(),
		},
		brainLockKey: {
			Key:     brainLockKey,
			Format:  "",
			Version: 8,
		},
	})

	oldCLIConfigInitialized := cliConfigInitialized
	oldConfiguration := currentCfg.configuration
	oldRegistration, hadRegistration := brainProviderRegistrationOverrides["pull-test"]
	cliConfigInitialized = true
	currentCfg.configuration = &configuration{
		brainProvider: "pull-test",
		brainCache:    BrainCacheConfig{Directory: t.TempDir()},
	}
	brainProviderRegistrationOverrides["pull-test"] = robot.BrainProviderRegistration{
		RemoteProvider: func(robot.Handler) robot.RemoteBrainBackend { return remote },
	}
	t.Cleanup(func() {
		cliConfigInitialized = oldCLIConfigInitialized
		currentCfg.configuration = oldConfiguration
		if hadRegistration {
			brainProviderRegistrationOverrides["pull-test"] = oldRegistration
		} else {
			delete(brainProviderRegistrationOverrides, "pull-test")
		}
	})

	output := captureStdout(t, func() {
		if err := cliPullBrain(brainPullOptions{}); err != nil {
			t.Fatalf("cliPullBrain: %v", err)
		}
	})
	if !strings.Contains(output, "Pulled 1 memories") {
		t.Fatalf("pull output did not exclude cloud lock from count:\n%s", output)
	}
	if remote.getCalls != 1 {
		t.Fatalf("remote Get calls = %d, want only the ordinary memory download", remote.getCalls)
	}

	cache, err := openExistingBrainCacheComplete(currentCfg.brainCache)
	if err != nil {
		t.Fatalf("open pulled cache: %v", err)
	}
	keys, err := cache.List()
	if err != nil {
		t.Fatalf("list pulled cache: %v", err)
	}
	if len(keys) != 1 || keys[0] != "alpha" {
		t.Fatalf("pulled cache keys = %v, want [alpha]", keys)
	}
	if _, exists, err := cache.Retrieve(brainLockKey); err != nil {
		t.Fatalf("retrieve cloud lock from pulled cache: %v", err)
	} else if exists {
		t.Fatal("cloud instance lock was copied into the local cache")
	}
}
