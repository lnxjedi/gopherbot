package cloudflarekv

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

// cfKVBrainConfig is populated from Gopherbot's config (YAML/JSON/etc.)
type cfKVBrainConfig struct {
	AccountID   string
	NamespaceID string
	APIToken    string
	MaxAgeHours int // e.g., 24 => evict if not used in 24 hours
}

// ephemeralRecord holds a single cached memory plus the last time we accessed it
type ephemeralRecord struct {
	data     []byte
	lastUsed time.Time
}

// ephemeralCFKVBrain implements robot.SimpleBrain (+ a Shutdown method).
type ephemeralCFKVBrain struct {
	cfKVBrainConfig

	handler robot.Handler

	// localMem is our in-memory cache of key -> ephemeralRecord
	localMem map[string]*ephemeralRecord
	mu       sync.RWMutex // protects localMem and the 'stopped' flag

	// queue is a channel for eventual "store" ops
	queue chan kvOperation

	// done signals background goroutines to stop
	done chan struct{}

	// stopped indicates we've started a shutdown sequence
	stopped bool

	// flusherWG allows us to wait for the flusher goroutine to finish
	flusherWG sync.WaitGroup
}

// kvOperation is what we enqueue for our background flusher
type kvOperation struct {
	key  string
	data []byte
}

//------------------------------------------------------------------------------
//  The Provider function is called by the Gopherbot engine
//------------------------------------------------------------------------------

func provider(r robot.Handler) robot.SimpleBrain {
	var cfg cfKVBrainConfig
	r.GetBrainConfig(&cfg)

	b := &ephemeralCFKVBrain{
		cfKVBrainConfig: cfg,
		handler:         r,
		localMem:        make(map[string]*ephemeralRecord),
		queue:           make(chan kvOperation, 100), // up to 100 queued store ops
		done:            make(chan struct{}),
	}

	// Start the flusher
	b.flusherWG.Add(1)
	go b.flusher()

	// Start the janitor
	go b.janitor()

	return b
}

//------------------------------------------------------------------------------
//  The flusher goroutine: processes enqueued CF KV "store" ops
//------------------------------------------------------------------------------

func (b *ephemeralCFKVBrain) flusher() {
	defer b.flusherWG.Done()

	for {
		select {
		case <-b.done:
			// We were signaled to stop
			return
		case op, ok := <-b.queue:
			if !ok {
				// The queue channel was closed => no more new store ops
				return
			}
			// Perform the actual PUT to CF KV
			if err := b.storeToCloudflare(op.key, op.data); err != nil {
				b.handler.Log(robot.Warn, "CF KV flusher: store failed for key=%s: %v", op.key, err)
			}
		}
	}
}

//------------------------------------------------------------------------------
//  The janitor goroutine: periodically evicts stale local cache entries
//------------------------------------------------------------------------------

func (b *ephemeralCFKVBrain) janitor() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-b.done:
			return
		case <-ticker.C:
			b.evictStale()
		}
	}
}

// evictStale removes any record that hasn't been accessed in > MaxAgeHours
func (b *ephemeralCFKVBrain) evictStale() {
	maxAge := time.Duration(b.MaxAgeHours) * time.Hour
	if maxAge <= 0 {
		return
	}
	now := time.Now()

	b.mu.Lock()
	for k, rec := range b.localMem {
		if now.Sub(rec.lastUsed) > maxAge {
			delete(b.localMem, k)
			b.handler.Log(robot.Debug, "CF KV janitor: evicted key=%s (idle > %dh)", k, b.MaxAgeHours)
		}
	}
	b.mu.Unlock()
}

//------------------------------------------------------------------------------
//  Implementation of robot.SimpleBrain
//------------------------------------------------------------------------------

func (b *ephemeralCFKVBrain) Store(key string, blob *[]byte) error {
	// Prevent new writes if we've already stopped
	b.mu.Lock()
	if b.stopped {
		b.mu.Unlock()
		return fmt.Errorf("brain is shutting down; no new writes accepted")
	}

	// Update local memory immediately
	b.localMem[key] = &ephemeralRecord{
		data:     *blob,
		lastUsed: time.Now(),
	}
	b.mu.Unlock()

	// Enqueue a store operation for eventual consistency
	select {
	case b.queue <- kvOperation{key: key, data: *blob}:
		// enqueued successfully
	default:
		// queue is full => fallback to synchronous store
		b.handler.Log(robot.Warn, "CF KV queue is full; doing immediate store for key=%s", key)
		if err := b.storeToCloudflare(key, *blob); err != nil {
			b.handler.Log(robot.Error, "Immediate store for key=%s failed: %v", key, err)
			return err
		}
	}
	return nil
}

func (b *ephemeralCFKVBrain) Retrieve(key string) (blob *[]byte, exists bool, err error) {
	// First check our local cache
	b.mu.RLock()
	rec, found := b.localMem[key]
	b.mu.RUnlock()

	if found {
		// We have it locally; "lucky us," we can use it (even if it might be older than MaxAge)
		// We'll update 'lastUsed' so the janitor won't evict it
		b.mu.Lock()
		rec.lastUsed = time.Now()
		dataCopy := rec.data
		b.mu.Unlock()

		return &dataCopy, true, nil
	}

	// Otherwise, fetch from Cloudflare
	data, err := b.fetchFromCloudflare(key)
	if err != nil {
		return nil, false, err
	}
	if data == nil {
		// Means 404 (not found)
		return nil, false, nil
	}

	// Cache the newly fetched data
	b.mu.Lock()
	b.localMem[key] = &ephemeralRecord{
		data:     data,
		lastUsed: time.Now(),
	}
	b.mu.Unlock()

	return &data, true, nil
}

// Delete: always call CF KV directly. Remove from local cache if it exists.
func (b *ephemeralCFKVBrain) Delete(key string) error {
	if err := b.deleteFromCloudflare(key); err != nil {
		return err
	}

	// Clean up locally so we don't return stale data
	b.mu.Lock()
	delete(b.localMem, key)
	b.mu.Unlock()

	return nil
}

// List: always call CF KV directly, ignoring local cache
func (b *ephemeralCFKVBrain) List() (keys []string, err error) {
	return b.listFromCloudflare()
}

//------------------------------------------------------------------------------
//  Shutdown() method for graceful shutdown
//------------------------------------------------------------------------------

func (b *ephemeralCFKVBrain) Shutdown() {
	b.mu.Lock()
	if b.stopped {
		b.mu.Unlock()
		// Already stopping/stopped, no-op
		return
	}
	b.stopped = true
	b.mu.Unlock()

	// Close the queue so no new store ops will be accepted
	close(b.queue)

	// Wait for the flusher goroutine to finish draining the queue
	b.flusherWG.Wait()

	// Signal the janitor to exit
	close(b.done)

	// At this point, no more goroutines are running, and we've flushed all writes.
	return
}

//------------------------------------------------------------------------------
//  Low-level Cloudflare KV calls
//------------------------------------------------------------------------------

func (b *ephemeralCFKVBrain) storeToCloudflare(key string, data []byte) error {
	endpoint := fmt.Sprintf(
		"https://api.cloudflare.com/client/v4/accounts/%s/storage/kv/namespaces/%s/values/%s",
		b.AccountID, b.NamespaceID, url.PathEscape(key),
	)

	req, err := http.NewRequest("PUT", endpoint, bytes.NewReader(data))
	if err != nil {
		b.handler.Log(robot.Error, "CF KV store: error creating PUT request: %v", err)
		return err
	}
	req.Header.Set("Authorization", "Bearer "+b.APIToken)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		b.handler.Log(robot.Warn, "CF KV store: HTTP error: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		switch resp.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			b.handler.Log(robot.Fatal, "CF KV store: invalid token? body=%s", string(body))
		default:
			b.handler.Log(robot.Error, "CF KV store: status=%d, body=%s", resp.StatusCode, string(body))
		}
		return fmt.Errorf("CF KV store error: status %d", resp.StatusCode)
	}
	return nil
}

func (b *ephemeralCFKVBrain) fetchFromCloudflare(key string) ([]byte, error) {
	endpoint := fmt.Sprintf(
		"https://api.cloudflare.com/client/v4/accounts/%s/storage/kv/namespaces/%s/values/%s",
		b.AccountID, b.NamespaceID, url.PathEscape(key),
	)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		b.handler.Log(robot.Error, "CF KV fetch: error creating GET request: %v", err)
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+b.APIToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		b.handler.Log(robot.Warn, "CF KV fetch: HTTP error: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return body, nil
	case http.StatusNotFound:
		// Key doesn't exist
		return nil, nil
	case http.StatusUnauthorized, http.StatusForbidden:
		body, _ := io.ReadAll(resp.Body)
		b.handler.Log(robot.Fatal, "CF KV fetch: invalid token? body=%s", string(body))
		return nil, fmt.Errorf("CF KV fetch fatal: unauthorized/forbidden")
	default:
		body, _ := io.ReadAll(resp.Body)
		b.handler.Log(robot.Error, "CF KV fetch: status=%d, body=%s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("CF KV fetch error: status %d", resp.StatusCode)
	}
}

func (b *ephemeralCFKVBrain) deleteFromCloudflare(key string) error {
	endpoint := fmt.Sprintf(
		"https://api.cloudflare.com/client/v4/accounts/%s/storage/kv/namespaces/%s/values/%s",
		b.AccountID, b.NamespaceID, url.PathEscape(key),
	)

	req, err := http.NewRequest("DELETE", endpoint, nil)
	if err != nil {
		b.handler.Log(robot.Error, "CF KV delete: error creating DELETE request: %v", err)
		return err
	}
	req.Header.Set("Authorization", "Bearer "+b.APIToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		b.handler.Log(robot.Warn, "CF KV delete: HTTP error: %v", err)
		return err
	}
	defer resp.Body.Close()

	// 200 = deleted, 404 = not found. Either is "gone."
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		switch resp.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			b.handler.Log(robot.Fatal, "CF KV delete: invalid token? body=%s", string(body))
		default:
			b.handler.Log(robot.Error, "CF KV delete: status=%d, body=%s", resp.StatusCode, string(body))
		}
		return fmt.Errorf("CF KV delete error: status %d", resp.StatusCode)
	}
	return nil
}

func (b *ephemeralCFKVBrain) listFromCloudflare() ([]string, error) {
	endpoint := fmt.Sprintf(
		"https://api.cloudflare.com/client/v4/accounts/%s/storage/kv/namespaces/%s/keys?limit=1000",
		b.AccountID, b.NamespaceID,
	)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		b.handler.Log(robot.Error, "CF KV list: error creating request: %v", err)
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+b.APIToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		b.handler.Log(robot.Warn, "CF KV list: HTTP error: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		switch resp.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			b.handler.Log(robot.Fatal, "CF KV list: invalid token? body=%s", string(body))
		default:
			b.handler.Log(robot.Error, "CF KV list: status=%d, body=%s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("CF KV list error: status %d", resp.StatusCode)
	}

	var listResp struct {
		Success  bool  `json:"success"`
		Errors   []any `json:"errors"`
		Messages []any `json:"messages"`
		Result   []struct {
			Name string `json:"name"`
		} `json:"result"`
	}
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&listResp); err != nil {
		b.handler.Log(robot.Error, "CF KV list: decode error: %v", err)
		return nil, err
	}

	keys := make([]string, 0, len(listResp.Result))
	for _, r := range listResp.Result {
		keys = append(keys, r.Name)
	}
	return keys, nil
}
