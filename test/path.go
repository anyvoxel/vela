package test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// CurrentProjectPath get the project root path
func CurrentProjectPath() string {
	path := currentFilePath()

	ppath, err := filepath.Abs(filepath.Join(filepath.Dir(path), "../"))
	if err != nil {
		panic(fmt.Errorf("build current project path with %s failed, %w", path, err))
	}

	f, err := os.Stat(ppath)
	if err != nil {
		panic(fmt.Errorf("stat project path %s failed, %w", ppath, err))
	}

	if f.Mode()&os.ModeSymlink != 0 {
		fpath, err := os.Readlink(ppath)
		if err != nil {
			panic(fmt.Errorf("readlink from path %s failed, %w", fpath, err))
		}
		ppath = fpath
	}

	return ppath
}

func currentFilePath() string {
	_, file, _, _ := runtime.Caller(1)
	return file
}
