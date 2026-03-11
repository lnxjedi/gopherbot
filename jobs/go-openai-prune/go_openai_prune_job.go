package main

import (
	"sort"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

const (
	conversationIndexDatumKey = "openaifallback:conversation:index:v1"
	defaultRetentionDays      = 30
	defaultMaxDeletesPerRun   = 200
)

type pruneConfig struct {
	RetentionDays    int  `json:"RetentionDays"`
	MaxDeletesPerRun int  `json:"MaxDeletesPerRun"`
	DryRun           bool `json:"DryRun"`
}

type conversationIndexEntry struct {
	Key       string `json:"key"`
	UpdatedAt string `json:"updated_at"`
}

type conversationIndex struct {
	Version       int                               `json:"version"`
	Conversations map[string]conversationIndexEntry `json:"conversations"`
}

type staleConversation struct {
	ID        string
	Key       string
	UpdatedAt time.Time
}

func JobHandler(r robot.Robot, args ...string) robot.TaskRetVal {
	cfg := loadPruneConfig(r)
	cutoff := time.Now().UTC().AddDate(0, 0, -cfg.RetentionDays)

	idx := conversationIndex{}
	_, exists, ret := r.CheckoutDatum(conversationIndexDatumKey, &idx, false)
	if ret != robot.Ok {
		r.Log(robot.Error, "go-openai-prune: failed to read conversation index: %s", ret)
		return robot.Fail
	}
	if !exists || len(idx.Conversations) == 0 {
		r.Log(robot.Debug, "go-openai-prune: nothing to prune")
		return robot.Normal
	}

	stale := selectStaleConversations(idx, cutoff)
	stale = limitStaleConversations(stale, cfg.MaxDeletesPerRun)
	if len(stale) == 0 {
		r.Log(robot.Debug, "go-openai-prune: no stale conversations older than %s", cutoff.Format(time.RFC3339))
		return robot.Normal
	}

	deleted := make(map[string]string)
	failures := 0
	for _, entry := range stale {
		if cfg.DryRun {
			deleted[entry.ID] = entry.Key
			continue
		}
		if del := r.DeleteDatum(entry.Key); del != robot.Ok {
			failures++
			r.Log(robot.Warn, "go-openai-prune: failed deleting conversation datum id=%s key=%s: %s", entry.ID, entry.Key, del)
			continue
		}
		deleted[entry.ID] = entry.Key
	}

	if cfg.DryRun {
		r.Log(robot.Info, "go-openai-prune (dry-run): matched=%d retention_days=%d", len(deleted), cfg.RetentionDays)
		return robot.Normal
	}

	removed := updateIndexAfterDeletes(r, deleted)
	r.Log(robot.Info, "go-openai-prune: deleted=%d index_removed=%d failed=%d retention_days=%d max_per_run=%d", len(deleted), removed, failures, cfg.RetentionDays, cfg.MaxDeletesPerRun)
	if failures > 0 {
		return robot.Fail
	}
	return robot.Normal
}

func loadPruneConfig(r robot.Robot) pruneConfig {
	cfg := pruneConfig{
		RetentionDays:    defaultRetentionDays,
		MaxDeletesPerRun: defaultMaxDeletesPerRun,
	}
	loaded := pruneConfig{}
	if ret := r.GetTaskConfig(&loaded); ret != robot.Ok && ret != robot.NoConfigFound {
		r.Log(robot.Warn, "go-openai-prune: failed loading job config, using defaults: %s", ret)
	}
	if loaded.RetentionDays > 0 {
		cfg.RetentionDays = loaded.RetentionDays
	}
	if loaded.MaxDeletesPerRun > 0 {
		cfg.MaxDeletesPerRun = loaded.MaxDeletesPerRun
	}
	cfg.DryRun = loaded.DryRun
	return cfg
}

func selectStaleConversations(idx conversationIndex, cutoff time.Time) []staleConversation {
	if len(idx.Conversations) == 0 {
		return nil
	}
	out := make([]staleConversation, 0, len(idx.Conversations))
	for id, entry := range idx.Conversations {
		if entry.Key == "" || entry.UpdatedAt == "" {
			continue
		}
		updatedAt, err := time.Parse(time.RFC3339, entry.UpdatedAt)
		if err != nil {
			continue
		}
		if updatedAt.Before(cutoff) {
			out = append(out, staleConversation{
				ID:        id,
				Key:       entry.Key,
				UpdatedAt: updatedAt,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].UpdatedAt.Before(out[j].UpdatedAt)
	})
	return out
}

func limitStaleConversations(stale []staleConversation, maxDeletesPerRun int) []staleConversation {
	if maxDeletesPerRun <= 0 || len(stale) <= maxDeletesPerRun {
		return stale
	}
	return stale[:maxDeletesPerRun]
}

func updateIndexAfterDeletes(r robot.Robot, deleted map[string]string) int {
	if len(deleted) == 0 {
		return 0
	}
	idx := conversationIndex{}
	locktoken, exists, ret := r.CheckoutDatum(conversationIndexDatumKey, &idx, true)
	if ret != robot.Ok {
		r.Log(robot.Warn, "go-openai-prune: failed checkout for index update: %s", ret)
		return 0
	}
	if !exists {
		r.CheckinDatum(conversationIndexDatumKey, locktoken)
		return 0
	}

	removed := removeDeletedFromIndex(&idx, deleted)
	if removed == 0 {
		r.CheckinDatum(conversationIndexDatumKey, locktoken)
		return 0
	}
	if ret = r.UpdateDatum(conversationIndexDatumKey, locktoken, idx); ret != robot.Ok {
		r.CheckinDatum(conversationIndexDatumKey, locktoken)
		r.Log(robot.Warn, "go-openai-prune: failed writing updated index: %s", ret)
		return 0
	}
	return removed
}

func removeDeletedFromIndex(idx *conversationIndex, deleted map[string]string) int {
	if idx == nil || len(idx.Conversations) == 0 || len(deleted) == 0 {
		return 0
	}
	removed := 0
	for id, expectedKey := range deleted {
		entry, ok := idx.Conversations[id]
		if !ok {
			continue
		}
		if expectedKey != "" && entry.Key != expectedKey {
			continue
		}
		delete(idx.Conversations, id)
		removed++
	}
	return removed
}
