package artifactquality

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GrinRus/ProvenArch/internal/contracts"
	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func ValidateCollectManifest(task acpruntime.Task) error {
	writeRoot := strings.TrimSpace(task.WriteRoot)
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
