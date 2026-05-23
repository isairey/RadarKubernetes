package updater

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// findInExtracted searches dir (and one level of subdirectories) for an entry
// matching the predicate. Returns the full path to the first match.
func findInExtracted(dir string, match func(os.DirEntry) bool) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if match(e) {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	// Check one level deeper
	for _, e := range entries {
		if e.IsDir() {
			subEntries, err := os.ReadDir(filepath.Join(dir, e.Name()))
			if err != nil {
				log.Printf("[updater] Warning: could not read subdirectory %s: %v", e.Name(), err)
				continue
			}
			for _, se := range subEntries {
				if match(se) {
					return filepath.Join(dir, e.Name(), se.Name()), nil
				}
			}
		}
	}
	return "", fmt.Errorf("no matching file found in extracted contents")
}
