package bot

import "testing"

func TestGetHistoryProviderFallsBackToMem(t *testing.T) {
	prevHistory := interfaces.history
	prevMem := memHistories
	prevHistoryConfig := historyConfig
	t.Cleanup(func() {
		interfaces.history = prevHistory
		memHistories = prevMem
		historyConfig = prevHistoryConfig
	})

	interfaces.history = nil
	memHistories = nil
	historyConfig = nil

	hprovider := getHistoryProvider()
	if hprovider == nil {
		t.Fatal("expected fallback history provider, got nil")
	}
	if memHistories == nil {
		t.Fatal("expected mem history provider to initialize")
	}
	if hprovider != memHistories {
		t.Fatalf("expected mem history provider fallback, got %T", hprovider)
	}

	logger, err := hprovider.NewLog("test-fallback", 1, 0)
	if err != nil {
		t.Fatalf("creating fallback history log: %v", err)
	}
	logger.Log("hello")
	logger.Close()
	logger.Finalize()
}
