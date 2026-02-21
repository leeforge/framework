package config

import (
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type Validator interface {
	Validate() error
}

type ConfigInterface interface {
	Bind(instance any) error
	Validate() error
	Export(path string) error
	Snapshot() (map[string]any, error)
	Restore() error
}

type Config struct {
	instance   *viper.Viper
	opts       ConfigOptions
	watchOnce  sync.Once
	watchMutex sync.RWMutex
	snapshot   map[string]any
}

type ConfigOptions struct {
	BasePath  string
	FileName  string
	FileType  string
	EnvPrefix string
	WatchAble bool
	OnChange  func(e fsnotify.Event)
	LoadAll   bool
}
