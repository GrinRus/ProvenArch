package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/GrinRus/ProvenArch/internal/contracts"
)

func TestValidateUsesExactNormalizedSelectedLinesAndBytes(t *testing.T) {
	source := []byte("first  \r\n第二\r\nthird\r\n")
	selected := "first  \n第二"
	digest := sha256.Sum256([]byte(selected))
	if err := Validate(source, &contracts.LineRange{Start: 1, End: 2}, selected, hex.EncodeToString(digest[:])); err != nil {
		t.Fatalf("expected exact evidence to pass: %v", err)
	}
}

func TestValidateRejectsTrimmedExcerptAndWrongHash(t *testing.T) {
	source := []byte(" first\nsecond\n")
	digest := sha256.Sum256([]byte(" first\nsecond"))
	if err := Validate(source, &contracts.LineRange{Start: 1, End: 2}, "first\nsecond", hex.EncodeToString(digest[:])); !errors.Is(err, ErrExcerptMismatch) {
		t.Fatalf("expected whitespace-preserving excerpt mismatch, got %v", err)
	}
	if err := Validate(source, &contracts.LineRange{Start: 1, End: 2}, " first\nsecond", "00"); !errors.Is(err, ErrExcerptHashMismatch) {
		t.Fatalf("expected hash mismatch, got %v", err)
	}
}

func TestValidateRejectsInvalidRangesAndUnboundedEvidence(t *testing.T) {
	if err := Validate([]byte("one\n"), &contracts.LineRange{Start: 0, End: 1}, "one", ""); !errors.Is(err, ErrLineRangeInvalid) {
		t.Fatalf("expected invalid range, got %v", err)
	}
	if err := Validate([]byte("one\n"), nil, "one", ""); !errors.Is(err, ErrLinesRequired) {
		t.Fatalf("expected required lines error, got %v", err)
	}
	if err := Validate(nil, &contracts.LineRange{Start: 1, End: 1}, "", ""); !errors.Is(err, ErrLineRangeInvalid) {
		t.Fatalf("expected empty source range error, got %v", err)
	}
}
