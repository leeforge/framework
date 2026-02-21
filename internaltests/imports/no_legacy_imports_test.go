package imports_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoLegacyFrameCoreImportsInLeeforgeFrame(t *testing.T) {
	root := filepath.Clean("../..")
	legacy := []string{
		"github.com/JsonLee12138/leeforge/frame-core",
		"\"leeforge/frame-core",
	}
	var hits []string

	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.Contains(path, "/internaltests/") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		b, _ := os.ReadFile(path)
		content := string(b)
		for _, k := range legacy {
			if strings.Contains(content, k) {
				hits = append(hits, path)
				break
			}
		}
		return nil
	})

	if len(hits) > 0 {
		t.Fatalf("legacy imports found: %v", hits[:min(10, len(hits))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
