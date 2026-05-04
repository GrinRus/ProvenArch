package artifactquality

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/contracts"
)

func ValidateCollectManifestInRoot(writeRoot string) error {
	writeRoot = strings.TrimSpace(writeRoot)
	if writeRoot == "" {
		return fmt.Errorf("collect write_root is empty")
	}

	manifestPath := filepath.Join(writeRoot, shardPackManifestFile)
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	return ValidateCollectManifestBytes(raw)
}

func ValidateCollectManifestBytes(raw []byte) error {
	_, err := contracts.ParseShardPackManifest(raw)
	return err
}
