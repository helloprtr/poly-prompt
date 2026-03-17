package capsule

import (
	"time"

	"github.com/helloprtr/poly-prompt/internal/config"
)

// ApplyRetentionPolicy returns a list of capsule IDs that should be deleted
// according to the configured retention days. Pinned capsules are never included.
func ApplyRetentionPolicy(caps []Capsule, cfg config.MemoryConfig) []string {
	now := time.Now().UTC()
	var toDelete []string

	for _, c := range caps {
		if c.Pinned {
			continue
		}
		var maxAge time.Duration
		switch c.Kind {
		case KindManual:
			maxAge = time.Duration(cfg.CapsuleRetentionDays) * 24 * time.Hour
		case KindAuto:
			maxAge = time.Duration(cfg.AutosaveRetentionDays) * 24 * time.Hour
		default:
			maxAge = time.Duration(cfg.CapsuleRetentionDays) * 24 * time.Hour
		}
		if maxAge > 0 && now.Sub(c.CreatedAt) > maxAge {
			toDelete = append(toDelete, c.ID)
		}
	}

	return toDelete
}

// ApplyOlderThan returns IDs of capsules older than the given duration.
// Pinned capsules are never included.
func ApplyOlderThan(caps []Capsule, d time.Duration) []string {
	now := time.Now().UTC()
	var toDelete []string
	for _, c := range caps {
		if c.Pinned {
			continue
		}
		if now.Sub(c.CreatedAt) > d {
			toDelete = append(toDelete, c.ID)
		}
	}
	return toDelete
}
