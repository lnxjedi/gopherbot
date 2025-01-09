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

// -----------------------------------------------------------------------------
// 1) Configuration & ephemeral store
// -----------------------------------------------------------------------------

type cfKVBrainConfig struct {
    AccountID   string
    NamespaceID string
    APIToken    string
    MaxAgeHours int // e.g., 24 => evict if not used in 24 hours
}

// ephemeralRecord holds the data and the last time we used it
type ephemeralRecord struct {
    data     []byte
    lastUsed time.Time
}

type ephemeralCFKVBrain struct {
    cfKVBrainConfig
    handler robot.Handler

    // localMem caches data for Store/Retrieve
    localMem map[string]*ephemeralRecord
    mu       sync.RWMutex

    // queue for eventual "store" ops
    queue chan kvOperation
    done  chan struct{}
}

// kvOperation is a queued store request
type kvOperation struct {
    key  string
    data []byte
}

// -----------------------------------------------------------------------------
// 2) provider function
// -----------------------------------------------------------------------------

func provider(r robot.Handler) robot.SimpleBrain {
    var cfg cfKVBrainConfig
    r.GetBrainConfig(&cfg)

    b := &ephemeralCFKVBrain{
        cfKVBrainConfig: cfg,
        handler:         r,
        localMem:        make(map[string]*ephemeralRecord),
        queue:           make(chan kvOperation, 100),
        done:            make(chan struct{}),
    }

    // background goroutines
    go b.flusher()
    go b.janitor()

    return b
}

// -----------------------------------------------------------------------------
// 3) Background goroutines
// -----------------------------------------------------------------------------

// flusher processes store ops enqueued by Store()
func (b *ephemeralCFKVBrain) flusher() {
    for {
        select {
        case <-b.done:
            return
        case op := <-b.queue:
            // Attempt to store in CF KV
            if err := b.storeToCloudflare(op.key, op.data); err != nil {
                b.handler.Log(robot.Warn, "CF KV flusher: store failed for key=%s: %v", op.key, err)
            }
        }
    }
}

// janitor evicts items that haven’t been used for more than MaxAgeHours
func (b *ephemeralCFKVBrain) janitor() {
    ticker := time.NewTicker(5 * time.Minute) // run every 5 minutes
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

// -----------------------------------------------------------------------------
// 4) robot.SimpleBrain methods
// -----------------------------------------------------------------------------

// Store: immediate local cache + enqueue for CF
func (b *ephemeralCFKVBrain) Store(key string, blob *[]byte) error {
    b.mu.Lock()
    b.localMem[key] = &ephemeralRecord{
        data:     *blob,
        lastUsed: time.Now(),
    }
    b.mu.Unlock()

    // enqueue for eventual consistency
    select {
    case b.queue <- kvOperation{key: key, data: *blob}:
    default:
        // queue is full => do synchronous store
        b.handler.Log(robot.Warn, "CF KV queue full; doing immediate store for key=%s", key)
        if err := b.storeToCloudflare(key, *blob); err != nil {
            b.handler.Log(robot.Error, "Immediate store for key=%s failed: %v", key, err)
            return err
        }
    }

    return nil
}

// Retrieve: if found locally, just use it and refresh timestamp.
// If not found, get from CF, cache it, and return.
func (b *ephemeralCFKVBrain) Retrieve(key string) (blob *[]byte, exists bool, err error) {
    // Check local first
    b.mu.RLock()
    rec, ok := b.localMem[key]
    b.mu.RUnlock()

    if ok {
        // We have it locally, so "lucky us"—even if it's older than MaxAge,
        // we won't forcibly remove it here. We'll just refresh lastUsed.
        b.mu.Lock()
        rec.lastUsed = time.Now()
        dataCopy := rec.data
        b.mu.Unlock()
        return &dataCopy, true, nil
    }

    // Not in cache => fetch from CF
    data, err := b.fetchFromCloudflare(key)
    if err != nil {
        return nil, false, err
    }
    if data == nil {
        // 404
        return nil, false, nil
    }

    // Cache it
    b.mu.Lock()
    b.localMem[key] = &ephemeralRecord{
        data:     data,
        lastUsed: time.Now(),
    }
    b.mu.Unlock()

    return &data, true, nil
}

// Delete: ALWAYS calls CF KV directly, then removes local copy if any
func (b *ephemeralCFKVBrain) Delete(key string) error {
    if err := b.deleteFromCloudflare(key); err != nil {
        return err
    }
    b.mu.Lock()
    delete(b.localMem, key)
    b.mu.Unlock()
    return nil
}

// List: ALWAYS calls CF KV directly, ignoring the local cache
func (b *ephemeralCFKVBrain) List() (keys []string, err error) {
    return b.listFromCloudflare()
}

// -----------------------------------------------------------------------------
// 5) Cloudflare KV helpers
// -----------------------------------------------------------------------------

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

    var lr struct {
        Success  bool   `json:"success"`
        Errors   []any  `json:"errors"`
        Messages []any  `json:"messages"`
        Result   []struct {
            Name string `json:"name"`
        } `json:"result"`
    }
    dec := json.NewDecoder(resp.Body)
    if err := dec.Decode(&lr); err != nil {
        b.handler.Log(robot.Error, "CF KV list: decode error: %v", err)
        return nil, err
    }

    keys := make([]string, 0, len(lr.Result))
    for _, r := range lr.Result {
        keys = append(keys, r.Name)
    }
    return keys, nil
}

// Optional: if you need a graceful shutdown
func (b *ephemeralCFKVBrain) shutdown() {
    close(b.done)
    // optionally drain b.queue, etc.
}
