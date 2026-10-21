package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// The permissive-mode spool. When the main audit log cannot be written, the
// record goes to a per-run file in the spool folder instead (same format, plus
// logged=spool), and mergeSpool moves those records into the main log later
// (plus late=true). Nothing here is wired into the operations yet.

// newSpoolIO wraps the primary log operations. While the primary Append works
// the line is passed through unchanged. When it fails, the line plus
// " logged=spool" is appended to <spoolDir>/<id>.log and Append succeeds;
// notice is called once, on the first such failure. If the spool cannot be
// written either, that error is returned. Size and Rotate are the primary's.
func newSpoolIO(primary logIO, spoolDir, id string, notice func()) logIO {
	spoolPath := filepath.Join(spoolDir, id+".log")
	noticed := false
	out := primary
	out.Append = func(path, line string) error {
		if primary.Append != nil {
			if err := primary.Append(path, line); err == nil {
				return nil
			}
		}
		if err := logAppendFile(spoolPath, line+" logged=spool"); err != nil {
			return err
		}
		if !noticed {
			noticed = true
			if notice != nil {
				notice()
			}
		}
		return nil
	}
	return out
}

// spoolLogField returns the value of the " key=" token of a log line, up to
// the next space, and whether the token is present.
func spoolLogField(line, key string) (string, bool) {
	tok := " " + key + "="
	i := strings.Index(line, tok)
	if i < 0 {
		return "", false
	}
	rest := line[i+len(tok):]
	if j := strings.IndexByte(rest, ' '); j >= 0 {
		rest = rest[:j]
	}
	return rest, true
}

// spoolSeenKey is the "id|phase" identity of a record; ok is false when the
// line has no id or no phase token.
func spoolSeenKey(line string) (string, bool) {
	id, ok1 := spoolLogField(line, "id")
	phase, ok2 := spoolLogField(line, "phase")
	if !ok1 || !ok2 {
		return "", false
	}
	return id + "|" + phase, true
}

// mergeSpool appends the spooled records to the main log, each with
// " late=true", and returns how many it appended. A missing spool folder is
// not an error. Only *.log and *.log.merging files are touched, in name
// order. A *.log file is first renamed to *.log.merging, so another process
// cannot merge it twice. A record whose id and phase are already in the main
// log (or were merged a moment ago) is skipped, so a retry appends only what
// is missing. When every line is handled the .merging file is deleted; on an
// Append error the count so far and the error are returned and the .merging
// file is kept.
func mergeSpool(spoolDir, mainPath string, lio logIO) (int, error) {
	entries, err := os.ReadDir(spoolDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}
	if lio.Append == nil {
		return 0, errors.New("no log writer configured")
	}

	seen := map[string]bool{}
	if b, rerr := os.ReadFile(mainPath); rerr == nil {
		for _, l := range strings.Split(string(b), "\n") {
			if k, ok := spoolSeenKey(strings.TrimRight(l, "\r")); ok {
				seen[k] = true
			}
		}
	} else if !errors.Is(rerr, os.ErrNotExist) {
		return 0, rerr
	}

	count := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		path := filepath.Join(spoolDir, name)
		switch {
		case strings.HasSuffix(name, ".log.merging"):
		case strings.HasSuffix(name, ".log"):
			target := path + ".merging"
			if _, serr := os.Lstat(target); serr == nil {
				// A leftover .merging file with this name is handled on
				// its own; renaming over it could lose records.
				continue
			}
			if rerr := os.Rename(path, target); rerr != nil {
				if errors.Is(rerr, os.ErrNotExist) {
					continue
				}
				return count, rerr
			}
			path = target
		default:
			continue
		}

		data, rerr := os.ReadFile(path)
		if rerr != nil {
			if errors.Is(rerr, os.ErrNotExist) {
				continue
			}
			return count, rerr
		}
		for _, l := range strings.Split(string(data), "\n") {
			l = strings.TrimRight(l, "\r")
			if strings.TrimSpace(l) == "" {
				continue
			}
			k, hasKey := spoolSeenKey(l)
			if hasKey && seen[k] {
				continue
			}
			if aerr := lio.Append(mainPath, l+" late=true"); aerr != nil {
				return count, aerr
			}
			if hasKey {
				seen[k] = true
			}
			count++
		}
		if derr := os.Remove(path); derr != nil && !errors.Is(derr, os.ErrNotExist) {
			return count, derr
		}
	}
	return count, nil
}
