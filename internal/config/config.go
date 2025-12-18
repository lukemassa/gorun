package config

import (
	"os"
	"path/filepath"
)

// TODO: Don't panic
func WorkingDir() string {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		panic(err)
	}
	cacheDir := filepath.Join(userCacheDir, "gorun-cache")
	err = os.MkdirAll(cacheDir, os.ModePerm)
	if err != nil {
		panic(err)
	}
	return cacheDir
}

func Sock(workingDir string) string {
	return filepath.Join(workingDir, "gorun.sock")
}
