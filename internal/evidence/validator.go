// Package evidence contains the single bounded validator for line-based source evidence.
// It is deliberately provider-free: callers pass bytes from an authoritative file and the
// validator compares the exact selected bytes without trimming or Unicode normalization.
package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/GrinRus/ProvenArch/internal/contracts"
)

const (
	MaxSourceBytes  = 16 * 1024 * 1024
	MaxExcerptBytes = 256 * 1024
	MaxLineCount    = 1_000_000
)

var (
	ErrLinesRequired       = errors.New("line range is required when excerpt evidence is present")
	ErrLineRangeInvalid    = errors.New("line range is invalid")
	ErrSourceTooLarge      = errors.New("source evidence exceeds the bounded read limit")
	ErrLineCountExceeded   = errors.New("source evidence exceeds the bounded line limit")
	ErrInvalidUTF8         = errors.New("source evidence is not valid UTF-8")
	ErrExcerptMismatch     = errors.New("excerpt is not the exact selected source lines")
	ErrExcerptHashMismatch = errors.New("excerpt_hash does not match the exact selected source bytes")
)

// Code returns the stable cross-consumer issue taxonomy for a validation error.
// Callers may add their surface prefix (for example, "audit.") without routing
// behavior on error-message text.
func Code(err error) string {
	switch {
	case errors.Is(err, ErrLinesRequired), errors.Is(err, ErrLineRangeInvalid),
		errors.Is(err, ErrLineCountExceeded), errors.Is(err, ErrInvalidUTF8):
		return "evidence.lines_invalid"
	case errors.Is(err, ErrExcerptMismatch):
		return "evidence.excerpt_mismatch"
	case errors.Is(err, ErrExcerptHashMismatch):
		return "evidence.hash_mismatch"
	case errors.Is(err, ErrSourceTooLarge):
		return "evidence.source_too_large"
	default:
		return "evidence.invalid"
	}
}

// HasBoundedEvidence reports whether a locator carries any line/excerpt/hash
// assertion that must be checked against source bytes.
func HasBoundedEvidence(lines *contracts.LineRange, excerpt, excerptHash string) bool {
	return lines != nil || excerpt != "" || strings.TrimSpace(excerptHash) != ""
}

// ValidateFile performs the bounded read and then applies Validate. It is the
// shared entry point for filesystem-backed citation consumers.
func ValidateFile(filePath string, lines *contracts.LineRange, excerpt, excerptHash string) error {
	info, err := os.Stat(filepath.Clean(filePath))
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("evidence source is not a regular file")
	}
	if info.Size() > MaxSourceBytes {
		return fmt.Errorf("%w: %d bytes", ErrSourceTooLarge, info.Size())
	}
	source, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return err
	}
	return Validate(source, lines, excerpt, excerptHash)
}

// Validate checks one citation's optional bounded evidence against source bytes.
// CRLF and CR are normalized to LF before line selection; all other bytes, including
// whitespace and Unicode scalar values, are preserved. The selected lines are joined
// with LF and never receive an implicit trailing newline.
func Validate(source []byte, lines *contracts.LineRange, excerpt, excerptHash string) error {
	if len(source) > MaxSourceBytes {
		return fmt.Errorf("%w: %d bytes", ErrSourceTooLarge, len(source))
	}
	if lines == nil {
		if excerpt != "" || strings.TrimSpace(excerptHash) != "" {
			return ErrLinesRequired
		}
		return nil
	}
	if lines.Start < 1 || lines.End < lines.Start {
		return ErrLineRangeInvalid
	}
	normalized := strings.ReplaceAll(string(source), "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	if !utf8.ValidString(normalized) {
		return ErrInvalidUTF8
	}
	lineValues := []string{}
	if normalized != "" {
		lineValues = strings.Split(normalized, "\n")
		if len(lineValues) > 1 && lineValues[len(lineValues)-1] == "" {
			lineValues = lineValues[:len(lineValues)-1]
		}
	}
	if len(lineValues) > MaxLineCount {
		return fmt.Errorf("%w: %d lines", ErrLineCountExceeded, len(lineValues))
	}
	if len(lineValues) == 0 || lines.End > len(lineValues) {
		return ErrLineRangeInvalid
	}
	selected := strings.Join(lineValues[lines.Start-1:lines.End], "\n")
	if len(selected) > MaxExcerptBytes {
		return fmt.Errorf("selected excerpt exceeds %d bytes", MaxExcerptBytes)
	}
	if excerpt != selected {
		return ErrExcerptMismatch
	}
	if len(excerpt) > MaxExcerptBytes {
		return fmt.Errorf("excerpt exceeds %d bytes", MaxExcerptBytes)
	}
	digest := sha256.Sum256([]byte(selected))
	if !strings.EqualFold(strings.TrimSpace(excerptHash), hex.EncodeToString(digest[:])) {
		return ErrExcerptHashMismatch
	}
	return nil
}
