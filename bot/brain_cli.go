package bot

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

type brainPullOptions struct {
	force          bool
	dryRun         bool
	upgradeCloudV3 bool
	budget         int
}

type brainRestoreOptions struct {
	force  bool
	dryRun bool
	v2     bool
	budget int
}

const v2RemoteBrainMigrationHint = "run gopherbot pull-brain -upgrade-cloud-v3 or run gopherbot pull-brain followed by gopherbot restore-brain before starting v3 runtime"

func (opts brainRestoreOptions) effectiveRemoteFormat() string {
	if opts.v2 {
		return "v2"
	}
	return "v3"
}

func cliPullBrain(opts brainPullOptions) error {
	initCLIConfigOnly()
	provider := currentCfg.brainProvider
	if provider == "" {
		provider = "file"
	}
	if provider == "file" {
		return cliPullFileBrain(opts)
	}
	remote, legacy, providerName, err := initRemoteBrainForCLI()
	if err != nil {
		return err
	}
	defer remote.Shutdown()
	cache, err := openBrainCacheForImport(currentCfg.brainCache, "", opts.force)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	cursor := ""
	var metas []robot.RemoteBrainRecord
	for {
		page, err := remote.ListMetadata(ctx, cursor, 1000)
		if err != nil {
			return err
		}
		metas = append(metas, page.Records...)
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	metas = pullableRemoteBrainMetadata(metas)
	sort.Slice(metas, func(i, j int) bool { return metas[i].Key < metas[j].Key })
	var v2Count, v3Count int
	for _, meta := range metas {
		if meta.Format == brainCacheFormat {
			v3Count++
		} else {
			v2Count++
		}
	}
	if opts.dryRun {
		reportCloudListSyncStatus(metas)
		fmt.Printf("Remote provider: %s\nTotal keys: %d\nV2 keys: %d\nV3 keys: %d\nWould modify remote: %t\n", providerName, len(metas), v2Count, v3Count, opts.upgradeCloudV3)
		return nil
	}
	if v2Count > 0 && legacy == nil {
		return fmt.Errorf("brain provider %q cannot import v2 remote records", providerName)
	}
	cloudWriteBudget := effectiveCloudWriteBudget(remote, opts.budget)
	var downloaded, cloudWrites int
	budgetExhausted := false
	for _, meta := range metas {
		if meta.Format != brainCacheFormat {
			continue
		}
		record, exists, err := remote.Get(ctx, meta.Key)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		downloaded++
		if err := cache.importV3Record(record); err != nil {
			return err
		}
	}
	for _, meta := range metas {
		if meta.Format == brainCacheFormat {
			continue
		}
		record, exists, err := legacy.GetV2(ctx, meta.Key)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		downloaded++
		if err := cache.importRaw(record.Key, record.Payload, false); err != nil {
			return err
		}
		if opts.upgradeCloudV3 {
			if cloudWriteBudget > 0 && cloudWrites >= cloudWriteBudget {
				budgetExhausted = true
				continue
			}
			localMeta, exists, err := cache.readMeta(record.Key)
			if err != nil {
				return err
			}
			if !exists {
				return fmt.Errorf("imported memory %s is missing local metadata", record.Key)
			}
			if err := remote.Put(ctx, robot.RemoteBrainRecord{
				Key:       record.Key,
				Payload:   record.Payload,
				Format:    brainCacheFormat,
				Version:   localMeta.Version,
				Checksum:  localMeta.Checksum,
				UpdatedAt: localMeta.UpdatedAt,
			}); err != nil {
				return err
			}
			cloudWrites++
		}
	}
	importedFrom := "v3"
	if v2Count > 0 {
		importedFrom = "v2"
	}
	if err := cache.finalizeImport(importedFrom); err != nil {
		return err
	}
	if opts.upgradeCloudV3 {
		if records, err := listAllRemoteBrainMetadata(ctx, remote); err == nil {
			reportCloudListSyncStatus(pullableRemoteBrainMetadata(records))
		} else {
			fmt.Fprintf(os.Stderr, "Brain cache sync: unable to refresh cloud status: %v\n", err)
		}
	} else {
		reportCloudListSyncStatus(metas)
	}
	fmt.Printf("Pulled %d memories into local brain cache (%d v2, %d v3, %d downloaded)\n", len(metas), v2Count, v3Count, downloaded)
	if budgetExhausted {
		return fmt.Errorf("cloud write budget exhausted after %d writes; local cache is complete, but the remote brain is not fully v3 yet; rerun restore-brain to continue", cloudWrites)
	}
	if v2Count > 0 && !opts.upgradeCloudV3 {
		fmt.Printf("Remote brain is still v2/unversioned; %s.\n", v2RemoteBrainMigrationHint)
	}
	return nil
}

func pullableRemoteBrainMetadata(records []robot.RemoteBrainRecord) []robot.RemoteBrainRecord {
	pullable := make([]robot.RemoteBrainRecord, 0, len(records))
	for _, record := range records {
		if record.Key == brainLockKey {
			continue
		}
		pullable = append(pullable, record)
	}
	return pullable
}

func cliPullFileBrain(opts brainPullOptions) error {
	var cfg struct {
		BrainDirectory string
		Encode         bool
	}
	_ = handle.GetBrainConfig(&cfg)
	if strings.TrimSpace(cfg.BrainDirectory) == "" {
		return fmt.Errorf("BrainConfig.BrainDirectory is required to import legacy file brain data")
	}
	cache, err := openBrainCacheForImport(currentCfg.brainCache, "file", opts.force)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(cfg.BrainDirectory)
	if err != nil {
		return err
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		key := entry.Name()
		payload, err := os.ReadFile(filepath.Join(cfg.BrainDirectory, key))
		if err != nil {
			return err
		}
		if !opts.dryRun {
			if cfg.Encode {
				decoded, err := base64Decode(payload)
				if err != nil {
					return fmt.Errorf("decoding legacy file brain key %s: %w", key, err)
				}
				payload = decoded
			}
			if err := cache.importRaw(key, payload, false); err != nil {
				return err
			}
		}
		count++
	}
	if opts.dryRun {
		fmt.Printf("Would import %d legacy file brain memories from %s\n", count, cfg.BrainDirectory)
		return nil
	}
	if err := cache.finalizeImport("file"); err != nil {
		return err
	}
	fmt.Printf("Imported %d legacy file brain memories into local cache\n", count)
	return nil
}

func cliRestoreBrain(opts brainRestoreOptions) error {
	initCLIConfigOnly()
	remoteFormat := opts.effectiveRemoteFormat()
	remote, legacy, providerName, err := initRemoteBrainForCLI()
	if err != nil {
		return err
	}
	defer remote.Shutdown()
	if remoteFormat == "v2" && legacy == nil {
		return fmt.Errorf("brain provider %q cannot write v2-compatible records", providerName)
	}
	cache, err := openExistingBrainCacheAny(currentCfg.brainCache)
	if err != nil {
		return err
	}
	keys, err := cache.List()
	if err != nil {
		return err
	}
	sort.Strings(keys)
	localKeys := make(map[string]bool, len(keys))
	for _, key := range keys {
		localKeys[key] = true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	writes := 0
	writeBudget := effectiveCloudWriteBudget(remote, opts.budget)
	if opts.force {
		deleted, err := restoreBrainDeleteRemoteExtras(ctx, cache, remote, legacy, localKeys, remoteFormat, opts.dryRun, writeBudget, &writes)
		if err != nil {
			return err
		}
		if deleted > 0 && opts.dryRun {
			fmt.Printf("Would remove %d remote memories not present in local cache\n", deleted)
		}
	}
	for _, key := range keys {
		payload, exists, err := cache.Retrieve(key)
		if err != nil {
			return err
		}
		if !exists || payload == nil {
			continue
		}
		if opts.dryRun {
			writes++
			continue
		}
		if writeBudget > 0 && writes >= writeBudget {
			return fmt.Errorf("write budget exhausted after %d writes; rerun restore-brain to continue", writes)
		}
		switch remoteFormat {
		case "v2":
			if err := legacy.PutV2(ctx, robot.LegacyBrainRecord{Key: key, Payload: *payload}); err != nil {
				return err
			}
		case "v3":
			meta, exists, err := cache.readMeta(key)
			if err != nil {
				return err
			}
			if !exists {
				continue
			}
			if err := remote.Put(ctx, robot.RemoteBrainRecord{
				Key:       key,
				Payload:   *payload,
				Format:    brainCacheFormat,
				Version:   meta.Version,
				Checksum:  meta.Checksum,
				UpdatedAt: meta.UpdatedAt,
			}); err != nil {
				return err
			}
			if err := markLocalV3Synced(cache, key, meta); err != nil {
				return err
			}
		}
		writes++
	}
	if opts.dryRun {
		if records, err := listAllRemoteBrainMetadata(ctx, remote); err == nil {
			reportCloudListSyncStatus(records)
		} else {
			fmt.Fprintf(os.Stderr, "Brain cache sync: unable to inspect cloud status: %v\n", err)
		}
		fmt.Printf("Would write %d memories to %s in %s format\n", writes, providerName, remoteFormat)
		return nil
	}
	fmt.Printf("Wrote %d memories to %s in %s format\n", writes, providerName, remoteFormat)
	if records, err := listAllRemoteBrainMetadata(ctx, remote); err == nil {
		reportCloudListSyncStatus(records)
	} else {
		fmt.Fprintf(os.Stderr, "Brain cache sync: unable to inspect cloud status: %v\n", err)
	}
	if remoteFormat == "v2" {
		fmt.Println("Remote brain is now v2-compatible and is not valid for v3 runtime startup.")
	}
	return nil
}

func restoreBrainDeleteRemoteExtras(ctx context.Context, cache *cachedBrain, remote robot.RemoteBrainBackend, legacy robot.LegacyBrainBackend, localKeys map[string]bool, remoteFormat string, dryRun bool, budget int, writes *int) (int, error) {
	deleted := 0
	switch remoteFormat {
	case "v2":
		cursor := ""
		for {
			page, err := legacy.ListV2(ctx, cursor, 1000)
			if err != nil {
				return deleted, err
			}
			for _, record := range page.Records {
				if localKeys[record.Key] {
					continue
				}
				if dryRun {
					deleted++
					*writes++
					continue
				}
				if budget > 0 && *writes >= budget {
					return deleted, fmt.Errorf("write budget exhausted after %d writes; rerun restore-brain to continue", *writes)
				}
				if err := legacy.DeleteV2(ctx, record.Key); err != nil {
					return deleted, err
				}
				deleted++
				*writes++
			}
			if page.NextCursor == "" {
				break
			}
			cursor = page.NextCursor
		}
	case "v3":
		cursor := ""
		for {
			page, err := remote.ListMetadata(ctx, cursor, 1000)
			if err != nil {
				return deleted, err
			}
			for _, record := range page.Records {
				if localKeys[record.Key] {
					continue
				}
				if dryRun {
					deleted++
					*writes++
					continue
				}
				if budget > 0 && *writes >= budget {
					return deleted, fmt.Errorf("write budget exhausted after %d writes; rerun restore-brain to continue", *writes)
				}
				version := record.Version + 1
				if version == 0 {
					version = 1
				}
				if err := remote.Delete(ctx, robot.RemoteBrainRecord{
					Key:       record.Key,
					Format:    brainCacheFormat,
					Version:   version,
					Deleted:   true,
					UpdatedAt: time.Now().UTC(),
				}); err != nil {
					return deleted, err
				}
				deleted++
				*writes++
			}
			if page.NextCursor == "" {
				break
			}
			cursor = page.NextCursor
		}
	}
	return deleted, nil
}

func markLocalV3Synced(cache *cachedBrain, key string, meta brainCacheMeta) error {
	now := time.Now().UTC()
	meta.SyncedAt = now
	if err := cache.writeMeta(meta); err != nil {
		return err
	}
	if err := cache.removeOutbox(key); err != nil {
		return err
	}
	if meta.Version >= cache.control.CheckpointVers {
		cache.control.CheckpointKey = key
		cache.control.CheckpointVers = meta.Version
		cache.control.CheckpointSum = meta.Checksum
		cache.control.UpdatedAt = now
		return cache.writeControl()
	}
	return nil
}

func effectiveCloudWriteBudget(remote robot.RemoteBrainBackend, override int) int {
	if override > 0 {
		return override
	}
	return remote.SyncPolicy().WriteBudgetPerDay
}

func listAllRemoteBrainMetadata(ctx context.Context, remote robot.RemoteBrainBackend) ([]robot.RemoteBrainRecord, error) {
	cursor := ""
	var records []robot.RemoteBrainRecord
	for {
		page, err := remote.ListMetadata(ctx, cursor, 1000)
		if err != nil {
			return nil, err
		}
		records = append(records, page.Records...)
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	return records, nil
}

func initRemoteBrainForCLI() (robot.RemoteBrainBackend, robot.LegacyBrainBackend, string, error) {
	provider := currentCfg.brainProvider
	if provider == "" || provider == "file" || provider == "mem" {
		return nil, nil, provider, fmt.Errorf("configured Brain %q is not a cloud remote brain", provider)
	}
	registration, ok := brainProviderRegistration(provider)
	if !ok || registration.RemoteProvider == nil {
		return nil, nil, provider, fmt.Errorf("no remote provider registered for brain %q", provider)
	}
	remote := registration.RemoteProvider(handle)
	legacy, _ := remote.(robot.LegacyBrainBackend)
	return remote, legacy, provider, nil
}

func base64Decode(data []byte) ([]byte, error) {
	data = []byte(strings.TrimSpace(string(data)))
	if len(data) == 0 {
		return nil, errors.New("empty base64 data")
	}
	out := make([]byte, base64.StdEncoding.DecodedLen(len(data)))
	n, err := base64.StdEncoding.Decode(out, data)
	if err != nil {
		return nil, err
	}
	return out[:n], nil
}
