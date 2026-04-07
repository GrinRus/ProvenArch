package orchestrator

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type RunLogLevel string

const (
	RunLogLevelInfo    RunLogLevel = "info"
	RunLogLevelWarning RunLogLevel = "warning"
	RunLogLevelError   RunLogLevel = "error"
)

type RunLogEntry struct {
	Cursor      int            `json:"cursor"`
	Timestamp   time.Time      `json:"timestamp"`
	Level       RunLogLevel    `json:"level"`
	StepID      string         `json:"step_id,omitempty"`
	DomainID    string         `json:"domain_id,omitempty"`
	Message     string         `json:"message"`
	TaskrunPath string         `json:"taskrun_path,omitempty"`
	Fields      map[string]any `json:"fields,omitempty"`
}

type RunLogPage struct {
	RunID      string        `json:"run_id"`
	Items      []RunLogEntry `json:"items"`
	NextCursor int           `json:"next_cursor"`
	EOF        bool          `json:"eof"`
}

func (s *Service) appendRunLog(runID string, entry RunLogEntry) {
	if !s.runLogsEnabled {
		return
	}
	if strings.TrimSpace(runID) == "" {
		return
	}
	entry.Message = strings.TrimSpace(entry.Message)
	if entry.Message == "" {
		return
	}
	if entry.Level == "" {
		entry.Level = RunLogLevelInfo
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = s.clock().UTC()
	}
	entry.StepID = strings.TrimSpace(entry.StepID)
	entry.DomainID = strings.TrimSpace(entry.DomainID)
	entry.TaskrunPath = strings.TrimSpace(entry.TaskrunPath)

	path, err := s.resolveRunLogPath(runID)
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer file.Close()

	line, err := json.Marshal(entry)
	if err != nil {
		return
	}
	_, _ = file.Write(append(line, '\n'))
}

func (s *Service) queryRunLogs(runID string, cursor int, limit int) (RunLogPage, error) {
	if cursor < 0 {
		return RunLogPage{}, errors.New("cursor must be >= 0")
	}
	if limit <= 0 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}
	path, err := s.resolveRunLogPath(runID)
	if err != nil {
		return RunLogPage{}, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return RunLogPage{
				RunID:      runID,
				Items:      []RunLogEntry{},
				NextCursor: cursor,
				EOF:        true,
			}, nil
		}
		return RunLogPage{}, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	items := make([]RunLogEntry, 0, limit)
	lineIndex := 0
	hasMore := false
	for scanner.Scan() {
		if lineIndex < cursor {
			lineIndex++
			continue
		}
		if len(items) >= limit {
			hasMore = true
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			lineIndex++
			continue
		}
		var entry RunLogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return RunLogPage{}, fmt.Errorf("decode run log line %d: %w", lineIndex, err)
		}
		entry.Cursor = lineIndex
		items = append(items, entry)
		lineIndex++
	}
	if err := scanner.Err(); err != nil {
		return RunLogPage{}, fmt.Errorf("scan run logs: %w", err)
	}

	nextCursor := cursor + len(items)
	if len(items) > 0 {
		nextCursor = items[len(items)-1].Cursor + 1
	}

	return RunLogPage{
		RunID:      runID,
		Items:      items,
		NextCursor: nextCursor,
		EOF:        !hasMore,
	}, nil
}

func (s *Service) cleanupRunLogs() error {
	if !s.runLogsEnabled {
		return nil
	}

	dir, err := s.resolveRunLogsDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	type logFile struct {
		path    string
		modTime time.Time
	}
	files := make([]logFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".ndjson") {
			continue
		}
		info, statErr := entry.Info()
		if statErr != nil {
			continue
		}
		files = append(files, logFile{
			path:    filepath.Join(dir, entry.Name()),
			modTime: info.ModTime(),
		})
	}

	ttl := s.runLogsTTL
	if ttl <= 0 {
		ttl = 7 * 24 * time.Hour
	}
	now := s.clock().UTC()
	remaining := make([]logFile, 0, len(files))
	for _, file := range files {
		age := now.Sub(file.modTime.UTC())
		if age > ttl {
			_ = os.Remove(file.path)
			continue
		}
		remaining = append(remaining, file)
	}

	maxRuns := s.runLogsMaxRuns
	if maxRuns <= 0 {
		maxRuns = 200
	}
	sort.Slice(remaining, func(i, j int) bool {
		return remaining[i].modTime.After(remaining[j].modTime)
	})
	if len(remaining) > maxRuns {
		for _, file := range remaining[maxRuns:] {
			_ = os.Remove(file.path)
		}
	}
	return nil
}

func (s *Service) resolveRunLogsDir() (string, error) {
	return s.runLogsWorkspace.Resolve(runLogsPath)
}

func (s *Service) resolveRunLogPath(runID string) (string, error) {
	dir, err := s.resolveRunLogsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, sanitizeRunLogSlug(runID)+".ndjson"), nil
}

func sanitizeRunLogSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "run"
	}

	var out []rune
	prevDash := false
	for _, r := range value {
		isAlpha := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'
		if isAlpha || isDigit {
			out = append(out, r)
			prevDash = false
			continue
		}
		if prevDash {
			continue
		}
		out = append(out, '-')
		prevDash = true
	}
	slug := strings.Trim(string(out), "-")
	if slug == "" {
		return "run"
	}
	return slug
}
