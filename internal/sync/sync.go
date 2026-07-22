// Package sync orchestrates one-way folder and message transfer host1 → host2.
package sync

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/emersion/go-imap/v2"

	"go-imapsync/internal/config"
	"go-imapsync/internal/identity"
	"go-imapsync/internal/imapclient"
	"go-imapsync/internal/report"
)

// Runner performs a sync.
type Runner struct {
	Cfg *config.Config
	Log *slog.Logger
}

// Run connects both sides and syncs.
func (r *Runner) Run(ctx context.Context) (*report.Stats, error) {
	stats := report.New()
	log := r.Log
	if log == nil {
		log = slog.Default()
	}

	src, err := imapclient.Dial(ctx, imapclient.Options{
		Label:    "host1",
		Side:     r.Cfg.Host1,
		Timeout:  r.Cfg.Timeout,
		Insecure: r.Cfg.InsecureTLS,
	})
	if err != nil {
		return stats, err
	}
	defer src.Close()

	dst, err := imapclient.Dial(ctx, imapclient.Options{
		Label:    "host2",
		Side:     r.Cfg.Host2,
		Timeout:  r.Cfg.Timeout,
		Insecure: r.Cfg.InsecureTLS,
	})
	if err != nil {
		return stats, err
	}
	defer dst.Close()

	srcFolders, err := src.ListFolders(ctx)
	if err != nil {
		return stats, err
	}
	dstFolders, err := dst.ListFolders(ctx)
	if err != nil {
		return stats, err
	}
	dstSet := make(map[string]struct{}, len(dstFolders))
	for _, f := range dstFolders {
		dstSet[f.Name] = struct{}{}
	}

	srcDelim := src.Delimiter()
	dstDelim := dst.Delimiter()

	for _, folder := range srcFolders {
		if err := ctx.Err(); err != nil {
			return stats, err
		}
		if folder.NoSelect || folder.Name == "" {
			continue
		}
		// Skip non-selectable specials only; INBOX and user folders proceed.
		dstName := mapFolderName(folder.Name, srcDelim, dstDelim)
		stats.FoldersProcessed++

		exists := false
		if _, ok := dstSet[dstName]; ok {
			exists = true
		}

		// Skip empty folders when requested (need SELECT to know).
		if r.Cfg.SkipEmptyFolders || !r.Cfg.JustFolders {
			if err := src.Select(ctx, folder.Name, true); err != nil {
				log.Warn("select host1 folder failed", "folder", folder.Name, "error", err)
				stats.Failed++
				continue
			}
		}

		if r.Cfg.SkipEmptyFolders {
			// Re-select already done; check via fetch path after select.
			// numMessages is internal — use identity fetch count after select.
		}

		if !exists {
			if r.Cfg.Dry {
				log.Info("would create folder on host2", "folder", dstName)
			} else {
				if err := dst.CreateFolder(ctx, dstName); err != nil {
					// Some servers return exists race; log and continue if list later finds it.
					log.Warn("create folder on host2 failed", "folder", dstName, "error", err)
					// Still try to use it if SELECT works.
				} else {
					stats.FoldersCreated++
					dstSet[dstName] = struct{}{}
					log.Info("created folder on host2", "folder", dstName)
				}
			}
		}

		if r.Cfg.JustFolders {
			continue
		}

		if err := r.syncFolder(ctx, src, dst, folder.Name, dstName, stats); err != nil {
			log.Error("folder sync failed", "folder", folder.Name, "error", err)
			stats.Failed++
		}
	}

	stats.Finish()
	return stats, nil
}

func (r *Runner) syncFolder(
	ctx context.Context,
	src, dst *imapclient.Client,
	srcName, dstName string,
	stats *report.Stats,
) error {
	log := r.Log
	if err := src.Select(ctx, srcName, true); err != nil {
		return err
	}
	// Destination select for identity set (create may have just happened).
	if !r.Cfg.Dry {
		if err := dst.Select(ctx, dstName, true); err != nil {
			// Try create again then select.
			_ = dst.CreateFolder(ctx, dstName)
			if err2 := dst.Select(ctx, dstName, true); err2 != nil {
				return fmt.Errorf("host2 select %q: %w", dstName, err2)
			}
		}
	}

	fields := r.Cfg.UseHeader
	srcMsgs, err := src.FetchAllForIdentity(ctx, fields)
	if err != nil {
		return err
	}
	if r.Cfg.SkipEmptyFolders && len(srcMsgs) == 0 {
		return nil
	}

	dstKeys := make(map[string]struct{})
	if !r.Cfg.Dry {
		dstMsgs, err := dst.FetchAllForIdentity(ctx, fields)
		if err != nil {
			return err
		}
		for _, m := range dstMsgs {
			k := identity.KeyFromHeaders(m.Headers, fields)
			if k != "" {
				dstKeys[k] = struct{}{}
			}
		}
	}

	for _, sm := range srcMsgs {
		if err := ctx.Err(); err != nil {
			return err
		}
		key := identity.KeyFromHeaders(sm.Headers, fields)
		if key == "" {
			// Unidentified: still transfer using UID-only uniqueness within run,
			// but always attempt copy when no key (imapsync can leave these).
			log.Debug("message without identity headers", "folder", srcName, "uid", sm.UID)
		} else if _, ok := dstKeys[key]; ok {
			stats.Skipped++
			continue
		}

		if r.Cfg.Dry {
			log.Info("would transfer message", "folder", srcName, "uid", sm.UID, "key", truncate(key, 80))
			stats.Transferred++
			if key != "" {
				dstKeys[key] = struct{}{}
			}
			continue
		}

		full, err := src.FetchFull(ctx, sm.UID)
		if err != nil {
			log.Error("fetch message failed", "folder", srcName, "uid", sm.UID, "error", err)
			stats.Failed++
			continue
		}
		flags := filterFlags(full.Flags)
		date := full.InternalDate
		if date.IsZero() {
			date = sm.InternalDate
		}
		if err := dst.Append(ctx, dstName, full.Body, flags, date); err != nil {
			log.Error("append message failed", "folder", dstName, "uid", sm.UID, "error", err)
			stats.Failed++
			continue
		}
		stats.Transferred++
		stats.Bytes += int64(len(full.Body))
		if key != "" {
			dstKeys[key] = struct{}{}
		}
		log.Info("transferred message", "folder", srcName, "to", dstName, "uid", sm.UID, "bytes", len(full.Body))
	}
	return nil
}

func mapFolderName(name string, srcDelim, dstDelim rune) string {
	if srcDelim == 0 || dstDelim == 0 || srcDelim == dstDelim {
		return name
	}
	return strings.ReplaceAll(name, string(srcDelim), string(dstDelim))
}

func filterFlags(in []imap.Flag) []imap.Flag {
	var out []imap.Flag
	for _, f := range in {
		// \Recent is session-local; never set on APPEND.
		if string(f) == "\\Recent" {
			continue
		}
		out = append(out, f)
	}
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
