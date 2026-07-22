// Package sync orchestrates one-way folder and message transfer host1 → host2.
//
// The algorithm is stateless: each run rebuilds identity key sets from IMAP
// header FETCHes, creates missing folders on host2, then APPENDs messages
// whose keys are not already present. Safe modes: Config.Dry (no mutations)
// and Config.JustFolders (folders only).
//
// Fatal host2 conditions (quota, closed connection) abort the run early so
// operators are not flooded with repeated identical errors. Re-running after
// fixing quota or network is safe because duplicates are skipped.
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
	"go-imapsync/internal/imaperr"
	"go-imapsync/internal/report"
)

// maxConsecutiveConnFails is how many back-to-back closed-connection failures
// are tolerated before aborting the current folder/run.
const maxConsecutiveConnFails = 3

// Runner performs a one-way sync using Cfg and optional structured Log.
type Runner struct {
	Cfg *config.Config
	Log *slog.Logger
}

// Run connects to both IMAP sides and syncs folders and messages according to Cfg.
//
// It always returns a non-nil *report.Stats (possibly partial). A non-nil error
// means the run stopped early or a hard connection failure occurred; soft
// per-message failures are counted in Stats.Failed without necessarily failing
// the return value until a fatal kind is hit.
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
		return stats, r.failConnect(stats, "host1", "connect/login", err)
	}
	defer src.Close()

	dst, err := imapclient.Dial(ctx, imapclient.Options{
		Label:    "host2",
		Side:     r.Cfg.Host2,
		Timeout:  r.Cfg.Timeout,
		Insecure: r.Cfg.InsecureTLS,
	})
	if err != nil {
		return stats, r.failConnect(stats, "host2", "connect/login", err)
	}
	defer dst.Close()

	srcFolders, err := src.ListFolders(ctx)
	if err != nil {
		return stats, r.failConnect(stats, "host1", "LIST folders", err)
	}
	dstFolders, err := dst.ListFolders(ctx)
	if err != nil {
		return stats, r.failConnect(stats, "host2", "LIST folders", err)
	}
	dstSet := make(map[string]struct{}, len(dstFolders))
	for _, f := range dstFolders {
		dstSet[f.Name] = struct{}{}
	}

	srcDelim := src.Delimiter()
	dstDelim := dst.Delimiter()

	for _, folder := range srcFolders {
		if err := ctx.Err(); err != nil {
			info := imaperr.Classify(err)
			stats.Aborted = info.Message
			return stats, err
		}
		if folder.NoSelect || folder.Name == "" {
			continue
		}
		dstName := mapFolderName(folder.Name, srcDelim, dstDelim)
		stats.FoldersProcessed++

		exists := false
		if _, ok := dstSet[dstName]; ok {
			exists = true
		}

		if r.Cfg.SkipEmptyFolders || !r.Cfg.JustFolders {
			if err := src.Select(ctx, folder.Name, true); err != nil {
				r.logOpError(log, stats, "host1", "SELECT", folder.Name, 0, err)
				continue
			}
		}

		if !exists {
			if r.Cfg.Dry {
				log.Info("would create folder on host2", "folder", dstName)
			} else {
				if err := dst.CreateFolder(ctx, dstName); err != nil {
					info := imaperr.Classify(err)
					// EXISTS races are common and non-fatal.
					if info.Kind != imaperr.KindUnknown && !strings.Contains(strings.ToLower(err.Error()), "exist") {
						log.Warn("could not create folder on host2",
							"folder", dstName,
							"reason", info.Message,
							"detail", imaperr.Detail(err),
							"hint", info.Hint,
						)
					} else {
						log.Debug("create folder on host2 returned error (may already exist)",
							"folder", dstName, "detail", imaperr.Detail(err))
					}
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
			info := imaperr.Classify(err)
			log.Error("stopped syncing folder",
				"folder", folder.Name,
				"reason", info.Message,
				"detail", imaperr.Detail(err),
				"hint", info.Hint,
			)
			if info.Fatal && (info.Kind == imaperr.KindQuota || info.Kind == imaperr.KindClosed) {
				stats.Aborted = info.Message
				// Abort remaining folders: continuing after dead connection or full quota is useless.
				stats.Finish()
				return stats, err
			}
		}
	}

	stats.Finish()
	return stats, nil
}

func (r *Runner) failConnect(stats *report.Stats, side, op string, err error) error {
	info := imaperr.Classify(err)
	stats.RecordError(info)
	stats.Aborted = fmt.Sprintf("%s %s failed: %s", side, op, info.Message)
	stats.Finish()
	if r.Log != nil {
		r.Log.Error(side+" "+op+" failed",
			"reason", info.Message,
			"detail", imaperr.Detail(err),
			"hint", info.Hint,
		)
	}
	return fmt.Errorf("%s: %w", stats.Aborted, err)
}

func (r *Runner) logOpError(log *slog.Logger, stats *report.Stats, side, op, folder string, uid imap.UID, err error) {
	info := imaperr.Classify(err)
	stats.RecordError(info)
	attrs := []any{
		"side", side,
		"op", op,
		"folder", folder,
		"reason", info.Message,
		"detail", imaperr.Detail(err),
	}
	if uid != 0 {
		attrs = append(attrs, "uid", uid)
	}
	if info.Hint != "" {
		attrs = append(attrs, "hint", info.Hint)
	}
	log.Error("operation failed", attrs...)
}

func (r *Runner) syncFolder(
	ctx context.Context,
	src, dst *imapclient.Client,
	srcName, dstName string,
	stats *report.Stats,
) error {
	log := r.Log
	if err := src.Select(ctx, srcName, true); err != nil {
		r.logOpError(log, stats, "host1", "SELECT", srcName, 0, err)
		return err
	}
	if !r.Cfg.Dry {
		if err := dst.Select(ctx, dstName, true); err != nil {
			_ = dst.CreateFolder(ctx, dstName)
			if err2 := dst.Select(ctx, dstName, true); err2 != nil {
				r.logOpError(log, stats, "host2", "SELECT", dstName, 0, err2)
				return err2
			}
		}
	}

	fields := r.Cfg.UseHeader
	srcMsgs, err := src.FetchAllForIdentity(ctx, fields)
	if err != nil {
		r.logOpError(log, stats, "host1", "FETCH headers", srcName, 0, err)
		return err
	}
	if r.Cfg.SkipEmptyFolders && len(srcMsgs) == 0 {
		return nil
	}

	dstKeys := make(map[string]struct{})
	if !r.Cfg.Dry {
		dstMsgs, err := dst.FetchAllForIdentity(ctx, fields)
		if err != nil {
			r.logOpError(log, stats, "host2", "FETCH headers", dstName, 0, err)
			return err
		}
		for _, m := range dstMsgs {
			k := identity.KeyFromHeaders(m.Headers, fields)
			if k != "" {
				dstKeys[k] = struct{}{}
			}
		}
	}

	var consecConnFails int
	var suppressed int

	for _, sm := range srcMsgs {
		if err := ctx.Err(); err != nil {
			return err
		}
		key := identity.KeyFromHeaders(sm.Headers, fields)
		if key == "" {
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
			info := imaperr.Classify(err)
			stats.RecordError(info)
			log.Error("could not fetch message from host1",
				"folder", srcName,
				"uid", sm.UID,
				"reason", info.Message,
				"detail", imaperr.Detail(err),
				"hint", info.Hint,
			)
			if info.Kind == imaperr.KindClosed {
				consecConnFails++
				if consecConnFails >= maxConsecutiveConnFails {
					return fmt.Errorf("host1 connection lost after %d consecutive failures: %w", consecConnFails, err)
				}
			} else {
				consecConnFails = 0
			}
			continue
		}
		flags := filterFlags(full.Flags)
		date := full.InternalDate
		if date.IsZero() {
			date = sm.InternalDate
		}
		if err := dst.Append(ctx, dstName, full.Body, flags, date); err != nil {
			info := imaperr.Classify(err)
			stats.RecordError(info)

			switch info.Kind {
			case imaperr.KindQuota:
				log.Error("destination mailbox is full (quota exceeded)",
					"folder", dstName,
					"uid", sm.UID,
					"bytes", len(full.Body),
					"reason", info.Message,
					"detail", imaperr.Detail(err),
					"hint", info.Hint,
				)
				stats.Aborted = "host2 mailbox quota exceeded"
				return fmt.Errorf("%s: %w", info.Message, err)

			case imaperr.KindClosed:
				consecConnFails++
				if consecConnFails == 1 {
					log.Error("connection to host2 closed during APPEND",
						"folder", dstName,
						"uid", sm.UID,
						"reason", info.Message,
						"detail", imaperr.Detail(err),
						"hint", info.Hint,
					)
				} else {
					suppressed++
					log.Debug("append failed (connection still closed)",
						"folder", dstName, "uid", sm.UID, "consecutive", consecConnFails)
				}
				if consecConnFails >= maxConsecutiveConnFails {
					if suppressed > 0 {
						log.Error("stopped folder after repeated connection failures",
							"folder", dstName,
							"consecutive_failures", consecConnFails,
							"similar_errors_suppressed", suppressed,
							"hint", "fix host2 connectivity/quota, then re-run — already copied mail is skipped",
						)
					}
					stats.Aborted = "host2 IMAP connection closed"
					return fmt.Errorf("host2 connection closed after %d consecutive APPEND failures: %w", consecConnFails, err)
				}

			default:
				consecConnFails = 0
				log.Error("could not append message to host2",
					"folder", dstName,
					"uid", sm.UID,
					"bytes", len(full.Body),
					"reason", info.Message,
					"detail", imaperr.Detail(err),
					"hint", info.Hint,
				)
			}
			continue
		}
		consecConnFails = 0
		suppressed = 0
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
