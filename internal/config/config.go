package config

import (
	"os"
	"path/filepath"
)

// TODO: Don't panic
func CacheDir() string {
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

// TODO: Don't panic
func DefaultSock() string {
	return filepath.Join(CacheDir(), "gorun.sock")
}
