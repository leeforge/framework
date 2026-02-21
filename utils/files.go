package utils

import (
	"os"
	"path/filepath"
	"runtime"
)

func Dirname() string {
	_, filename, _, _ := runtime.Caller(1)
	return filepath.Dir(filename)
}

func Resolve(paths ...string) string {
	dir := Dirname()
	allPaths := append([]string{dir}, paths...)
	return filepath.Join(allPaths...)
}

func Exists(path string) (isDir bool, exists bool, err error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	return info.IsDir(), true, nil
}

func CreateDir(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}

func CreateFile(path string) (*os.File, error) {
	return os.Create(path)
}

func OnlyCreateFile(path string) error {
	_, err := os.Create(path)
	return err
}
