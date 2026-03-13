package bot

import (
	"testing"
	"time"
)

func TestWaitForPluginInitQuiescenceReturnsWithoutBatch(t *testing.T) {
	done := make(chan struct{})
	go func() {
		waitForPluginInitQuiescence()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("waitForPluginInitQuiescence blocked without an active batch")
	}
}

func TestWaitForPluginInitQuiescenceWaitsForCurrentBatch(t *testing.T) {
	batch := newPluginInitBatch()
	batch.add()
	batch.add()
	setCurrentPluginInitBatch(batch)
	batch.seal()

	done := make(chan struct{})
	go func() {
		waitForPluginInitQuiescence()
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("waitForPluginInitQuiescence returned before the batch completed")
	case <-time.After(25 * time.Millisecond):
	}

	batch.complete()

	select {
	case <-done:
		t.Fatal("waitForPluginInitQuiescence returned before the batch fully completed")
	case <-time.After(25 * time.Millisecond):
	}

	batch.complete()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("waitForPluginInitQuiescence did not return after the batch completed")
	}
}
