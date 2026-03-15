package search

import (
	"io/fs"
	"path/filepath"
	"sync"

	"github.com/charlievieth/fastwalk"
)

func walkJSONLFiles(root string) ([]string, error) {
	files := make([]string, 0, 64)
	var mu sync.Mutex

	err := fastwalk.Walk(nil, root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if entry.IsDir() && entry.Name() == "subagents" {
			return filepath.SkipDir
		}

		if entry.IsDir() || filepath.Ext(entry.Name()) != ".jsonl" {
			return nil
		}

		mu.Lock()
		files = append(files, path)
		mu.Unlock()
		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}
