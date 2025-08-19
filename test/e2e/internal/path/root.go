package pathutil

import (
	"os"
	"path/filepath"
	"sync"
)

var (
	repoRoot string
	once     sync.Once
)

// RepoRoot walks up from CWD to find the directory containing go.mod.
// Caches the result for subsequent calls.
func RepoRoot() (string, error) {
	var err error
	once.Do(func() {
		wd, e := os.Getwd()
		if e != nil {
			err = e
			return
		}
		cur := wd
		for {
			if _, e := os.Stat(filepath.Join(cur, "go.mod")); e == nil {
				repoRoot = cur
				return
			}
			parent := filepath.Dir(cur)
			if parent == cur {
				err = os.ErrNotExist
				return
			}
			cur = parent
		}
	})
	if repoRoot == "" && err == nil {
		err = os.ErrNotExist
	}
	return repoRoot, err
}
