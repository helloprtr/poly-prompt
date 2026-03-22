package capsule_test

import (
	"testing"
	"time"

	"github.com/helloprtr/poly-prompt/internal/capsule"
	"github.com/helloprtr/poly-prompt/internal/config"
)

func makeCapsuleAge(id, kind string, age time.Duration, pinned bool) capsule.Capsule {
	now := time.Now().UTC()
	return capsule.Capsule{
		ID:        id,
		Kind:      kind,
		Pinned:    pinned,
		CreatedAt: now.Add(-age),
		UpdatedAt: now.Add(-age),
		Repo:      capsule.RepoState{Branch: "main", HeadSHA: "abc"},
	}
}

func TestPruneByRetentionDays(t *testing.T) {
	cfg := config.MemoryConfig{
		CapsuleRetentionDays:  30,
		AutosaveRetentionDays: 14,
	}

	caps := []capsule.Capsule{
		makeCapsuleAge("cap_keep_manual", capsule.KindManual, 10*24*time.Hour, false), // 10d — keep
		makeCapsuleAge("cap_drop_manual", capsule.KindManual, 31*24*time.Hour, false), // 31d — drop
		makeCapsuleAge("cap_keep_auto", capsule.KindAuto, 5*24*time.Hour, false),      // 5d  — keep
		makeCapsuleAge("cap_drop_auto", capsule.KindAuto, 15*24*time.Hour, false),     // 15d — drop
		makeCapsuleAge("cap_pinned_old", capsule.KindManual, 365*24*time.Hour, true),  // 1yr pinned — keep
	}

	toDelete := capsule.ApplyRetentionPolicy(caps, cfg)

	deleteSet := map[string]bool{}
	for _, id := range toDelete {
		deleteSet[id] = true
	}

	if deleteSet["cap_keep_manual"] {
		t.Error("cap_keep_manual should not be deleted (10d < 30d)")
	}
	if !deleteSet["cap_drop_manual"] {
		t.Error("cap_drop_manual should be deleted (31d > 30d)")
	}
	if deleteSet["cap_keep_auto"] {
		t.Error("cap_keep_auto should not be deleted (5d < 14d)")
	}
	if !deleteSet["cap_drop_auto"] {
		t.Error("cap_drop_auto should be deleted (15d > 14d)")
	}
	if deleteSet["cap_pinned_old"] {
		t.Error("cap_pinned_old should never be deleted (pinned)")
	}
}
