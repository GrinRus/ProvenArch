package api

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	producttasks "github.com/GrinRus/ProvenArch/internal/tasks"
	"github.com/GrinRus/ProvenArch/internal/workspace"
)

const publicationIntentPrefix = "publication_intent_v1:"
const publicationJournalGitPath = "acp-publication-journal.json"

// publicationIntent is a durable prepare record for a Git mutation. Git and
// task-history persistence cannot share one filesystem transaction, so the
// intent gives restart reconciliation an exact, fail-closed identity boundary.
type publicationIntent struct {
	TaskID       string               `json:"task_id"`
	AttemptID    string               `json:"attempt_id"`
	RunID        string               `json:"run_id"`
	Action       string               `json:"action"`
	TargetBranch string               `json:"target_branch,omitempty"`
	MessageHash  string               `json:"message_hash,omitempty"`
	Before       gitWorkspaceIdentity `json:"before"`
	Fingerprint  string               `json:"fingerprint"`
}

// taskPublicationContext is intentionally all-or-nothing. A partial context
// is never completed from the latest run or any other heuristic source.
type taskPublicationContext struct {
	TaskID    string
	AttemptID string
	RunID     string
}

func (value taskPublicationContext) provided() bool {
	return strings.TrimSpace(value.TaskID) != "" || strings.TrimSpace(value.AttemptID) != "" || strings.TrimSpace(value.RunID) != ""
}

func (value taskPublicationContext) validateShape() error {
	if !value.provided() {
		return nil
	}
	if !producttasks.IsOpaqueID(value.TaskID) || !producttasks.IsOpaqueID(value.AttemptID) || strings.TrimSpace(value.RunID) == "" {
		return errors.New("task_id, attempt_id and run_id are required and must identify an exact Task/Attempt/run join")
	}
	return nil
}

func validateTaskPublicationContext(history producttasks.History, value taskPublicationContext) (producttasks.Task, producttasks.Attempt, error) {
	if err := value.validateShape(); err != nil {
		return producttasks.Task{}, producttasks.Attempt{}, err
	}
	if !value.provided() {
		return producttasks.Task{}, producttasks.Attempt{}, nil
	}
	task, found := findTask(history, value.TaskID)
	if !found {
		return producttasks.Task{}, producttasks.Attempt{}, fmt.Errorf("task %q was not found", value.TaskID)
	}
	attempt, found := findAttempt(history, value.TaskID, value.AttemptID)
	if !found {
		return producttasks.Task{}, producttasks.Attempt{}, fmt.Errorf("attempt %q was not found for task %q", value.AttemptID, value.TaskID)
	}
	if attempt.RunID != strings.TrimSpace(value.RunID) {
		return producttasks.Task{}, producttasks.Attempt{}, fmt.Errorf("attempt %q is linked to run %q, not %q", value.AttemptID, attempt.RunID, value.RunID)
	}
	return task, attempt, nil
}

func publicationUnavailable(reason string) producttasks.Publication {
	return producttasks.Publication{
		State:             producttasks.PublicationUnavailable,
		UnavailableReason: strings.TrimSpace(reason),
	}
}

func encodePublicationIntent(value publicationIntent) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal publication intent: %w", err)
	}
	return publicationIntentPrefix + base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodePublicationIntent(value string) (publicationIntent, bool) {
	if !strings.HasPrefix(value, publicationIntentPrefix) {
		return publicationIntent{}, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, publicationIntentPrefix))
	if err != nil {
		return publicationIntent{}, false
	}
	var intent publicationIntent
	if err := json.Unmarshal(raw, &intent); err != nil || !producttasks.IsOpaqueID(intent.TaskID) || !producttasks.IsOpaqueID(intent.AttemptID) || strings.TrimSpace(intent.RunID) == "" {
		return publicationIntent{}, false
	}
	if intent.Action != "commit" && intent.Action != "branch" {
		return publicationIntent{}, false
	}
	if intent.Action == "commit" && !isSHA256(intent.MessageHash) {
		return publicationIntent{}, false
	}
	if !isSHA256(intent.Fingerprint) || strings.TrimSpace(intent.Before.HeadOID) == "" || strings.TrimSpace(intent.Before.Branch) == "" {
		return publicationIntent{}, false
	}
	return intent, true
}

func prepareTaskPublication(registry *producttasks.Registry, ws workspace.Root, publicationContext taskPublicationContext, action, targetBranch, message string, before gitWorkspaceState) error {
	if registry == nil {
		return errors.New("task history is not available")
	}
	if err := publicationContext.validateShape(); err != nil {
		return err
	}
	if !publicationContext.provided() {
		return nil
	}
	intent := publicationIntent{
		TaskID: publicationContext.TaskID, AttemptID: publicationContext.AttemptID, RunID: publicationContext.RunID,
		Action: action, TargetBranch: strings.TrimSpace(targetBranch), Before: before.Identity,
		MessageHash: publicationMessageHash(message), Fingerprint: before.Fingerprint,
	}
	if _, _, err := validateTaskPublicationContext(registry.Snapshot(), publicationContext); err != nil {
		return err
	}
	journalPath, err := publicationJournalPath(context.Background(), ws)
	if err != nil {
		return err
	}
	intents, err := readPublicationJournal(journalPath)
	if err != nil {
		return err
	}
	for _, existing := range intents {
		if samePublicationIntent(existing, intent) {
			return nil
		}
	}
	intents = append(intents, intent)
	return writePublicationJournal(journalPath, intents)
}

func clearTaskPublicationIntent(ws workspace.Root, publicationContext taskPublicationContext, action string) error {
	if !publicationContext.provided() {
		return nil
	}
	journalPath, err := publicationJournalPath(context.Background(), ws)
	if err != nil {
		return err
	}
	intents, err := readPublicationJournal(journalPath)
	if err != nil {
		return err
	}
	filtered := make([]publicationIntent, 0, len(intents))
	for _, intent := range intents {
		if intent.TaskID == publicationContext.TaskID && intent.AttemptID == publicationContext.AttemptID && intent.RunID == publicationContext.RunID && (action == "" || intent.Action == action) {
			continue
		}
		filtered = append(filtered, intent)
	}
	if len(filtered) == len(intents) {
		return nil
	}
	return writePublicationJournal(journalPath, filtered)
}

func samePublicationIntent(left, right publicationIntent) bool {
	return left.TaskID == right.TaskID && left.AttemptID == right.AttemptID && left.RunID == right.RunID && left.Action == right.Action && left.TargetBranch == right.TargetBranch && left.MessageHash == right.MessageHash && left.Fingerprint == right.Fingerprint && left.Before == right.Before
}

func publicationJournalPath(ctx context.Context, ws workspace.Root) (string, error) {
	pathValue, err := runGit(ctx, ws.Path, "rev-parse", "--git-path", publicationJournalGitPath)
	if err != nil {
		return "", fmt.Errorf("resolve publication journal: %w", err)
	}
	pathValue = strings.TrimSpace(pathValue)
	if pathValue == "" {
		return "", errors.New("git publication journal path is empty")
	}
	if !filepath.IsAbs(pathValue) {
		pathValue = filepath.Join(ws.Path, pathValue)
	}
	return filepath.Clean(pathValue), nil
}

func readPublicationJournal(pathValue string) ([]publicationIntent, error) {
	raw, err := os.ReadFile(pathValue)
	if errors.Is(err, os.ErrNotExist) {
		return []publicationIntent{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read publication journal: %w", err)
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return []publicationIntent{}, nil
	}
	var intents []publicationIntent
	if err := json.Unmarshal(raw, &intents); err != nil {
		return nil, fmt.Errorf("decode publication journal: %w", err)
	}
	for _, intent := range intents {
		marker, err := encodePublicationIntent(intent)
		if err != nil {
			return nil, errors.New("publication journal contains an invalid intent")
		}
		if _, ok := decodePublicationIntent(marker); !ok {
			return nil, errors.New("publication journal contains an invalid intent")
		}
	}
	return intents, nil
}

func writePublicationJournal(pathValue string, intents []publicationIntent) error {
	raw, err := json.Marshal(intents)
	if err != nil {
		return fmt.Errorf("marshal publication journal: %w", err)
	}
	directory := filepath.Dir(pathValue)
	temporary, err := os.CreateTemp(directory, ".acp-publication-journal-*")
	if err != nil {
		return fmt.Errorf("create publication journal temp: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("protect publication journal temp: %w", err)
	}
	if _, err := temporary.Write(raw); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write publication journal temp: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync publication journal temp: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close publication journal temp: %w", err)
	}
	if err := os.Rename(temporaryPath, pathValue); err != nil {
		return fmt.Errorf("publish publication journal: %w", err)
	}
	directoryHandle, err := os.Open(directory)
	if err != nil {
		return fmt.Errorf("open publication journal directory: %w", err)
	}
	defer directoryHandle.Close()
	if err := directoryHandle.Sync(); err != nil {
		return fmt.Errorf("sync publication journal directory: %w", err)
	}
	return nil
}

func buildTaskPublication(action string, context taskPublicationContext, before, after gitWorkspaceState) producttasks.Publication {
	return producttasks.Publication{
		State:                producttasks.PublicationLinked,
		AttemptID:            strings.TrimSpace(context.AttemptID),
		RunID:                strings.TrimSpace(context.RunID),
		Action:               action,
		Branch:               after.Identity.Branch,
		BaseRef:              before.Identity.BaseRef,
		BaseOID:              before.Identity.BaseOID,
		HeadOID:              after.Identity.HeadOID,
		Commit:               after.Identity.HeadOID,
		InventoryFingerprint: before.Fingerprint,
	}
}

func recordTaskPublication(registry *producttasks.Registry, ws workspace.Root, publicationContext taskPublicationContext, publication producttasks.Publication) error {
	if registry == nil {
		return errors.New("task history is not available")
	}
	if err := publicationContext.validateShape(); err != nil {
		return err
	}
	if !publicationContext.provided() {
		return nil
	}
	if err := registry.Update(func(history *producttasks.History) error {
		task, attempt, err := validateTaskPublicationContext(*history, publicationContext)
		if err != nil {
			return err
		}
		if publication.AttemptID != attempt.AttemptID || publication.RunID != attempt.RunID {
			return errors.New("publication does not match the exact Attempt/run join")
		}
		if index := indexTask(history.Tasks, task.TaskID); index >= 0 {
			history.Tasks[index].Publication = publication
			history.Tasks[index].LastActivityAt = time.Now().UTC().Format(time.RFC3339Nano)
		}
		for index := range history.Attempts {
			if history.Attempts[index].TaskID == attempt.TaskID && history.Attempts[index].AttemptID == attempt.AttemptID {
				history.Attempts[index].Publication = publication
				return nil
			}
		}
		return errors.New("attempt disappeared while recording publication")
	}); err != nil {
		return err
	}
	if err := clearTaskPublicationIntent(ws, publicationContext, publication.Action); err != nil {
		return fmt.Errorf("publication linked but intent cleanup failed: %w", err)
	}
	return nil
}

func (s *Server) reconcilePendingTaskPublications(registry *producttasks.Registry) {
	if registry == nil || strings.TrimSpace(s.workspace.Path) == "" {
		return
	}
	journalPath, err := publicationJournalPath(context.Background(), s.workspace)
	if err != nil {
		return
	}
	intents, err := readPublicationJournal(journalPath)
	if err != nil {
		return
	}
	for _, intent := range intents {
		before := gitWorkspaceState{
			Identity:    intent.Before,
			Fingerprint: intent.Fingerprint,
		}
		after, err := collectWorkspaceGitState(context.Background(), s.workspace)
		if err != nil || !pendingPublicationWasApplied(context.Background(), s.workspace, intent, after) {
			continue
		}
		publication := buildTaskPublication(intent.Action, taskPublicationContext{
			TaskID: intent.TaskID, AttemptID: intent.AttemptID, RunID: intent.RunID,
		}, before, after)
		if err := recordTaskPublication(registry, s.workspace, taskPublicationContext{
			TaskID: intent.TaskID, AttemptID: intent.AttemptID, RunID: intent.RunID,
		}, publication); err != nil {
			continue
		}
		// A single current Git state can prove at most one pending mutation. Any
		// additional intent needs a fresh state transition before it is eligible.
		break
	}
}

func pendingPublicationWasApplied(ctx context.Context, ws workspace.Root, intent publicationIntent, after gitWorkspaceState) bool {
	switch intent.Action {
	case "branch":
		return strings.TrimSpace(intent.TargetBranch) != "" && after.Identity.Branch == intent.TargetBranch && after.Identity.HeadOID == intent.Before.HeadOID
	case "commit":
		if after.Identity.Branch != intent.Before.Branch || after.Identity.HeadOID == "" || after.Identity.HeadOID == intent.Before.HeadOID {
			return false
		}
		parents, err := runGit(ctx, ws.Path, "rev-list", "--parents", "-n", "1", after.Identity.HeadOID)
		if err != nil {
			return false
		}
		fields := strings.Fields(parents)
		if len(fields) != 2 || fields[0] != after.Identity.HeadOID || fields[1] != intent.Before.HeadOID {
			return false
		}
		message, err := runGit(ctx, ws.Path, "show", "-s", "--format=%B", after.Identity.HeadOID)
		return err == nil && publicationMessageHash(message) == intent.MessageHash
	default:
		return false
	}
}

func publicationMessageHash(message string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(message)))
	return fmt.Sprintf("%x", sum[:])
}

func isSHA256(value string) bool {
	if len(strings.TrimSpace(value)) != 64 {
		return false
	}
	for _, char := range strings.TrimSpace(value) {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') && (char < 'A' || char > 'F') {
			return false
		}
	}
	return true
}
