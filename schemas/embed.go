package schemas

import (
	"embed"
	"fmt"
)

//go:embed *.json
var files embed.FS

func Load(name string) ([]byte, error) {
	content, err := files.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("load schema %q: %w", name, err)
	}
	return content, nil
}
