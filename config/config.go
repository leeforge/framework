package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/creasty/defaults"
	"github.com/fsnotify/fsnotify"
	"github.com/leeforge/framework/env_mode"
	"github.com/leeforge/framework/utils"
	"github.com/spf13/viper"
)

func DefaultConfigOptions() ConfigOptions {
	basePath := os.Getenv("CONFIG_PATH")
	if basePath == "" {
		basePath = "config"
	}

	return ConfigOptions{
		BasePath:  basePath,
		FileName:  "config",
		FileType:  "yaml",
		EnvPrefix: "",
		WatchAble: false,
		OnChange:  nil,
	}
}

func DevConfigOptions() ConfigOptions {
	opts := DefaultConfigOptions()
	opts.WatchAble = true
	return opts
}

func NewConfig(optsArr ...ConfigOptions) (*Config, error) {
	var opts ConfigOptions
	if len(optsArr) == 0 {
		opts = DefaultConfigOptions()
	} else {
		opts = optsArr[0]
	}

	instance, err := CreateConfig(opts)
	if err != nil {
		return nil, err
	}

	return &Config{
		instance: instance,
		opts:     opts,
	}, nil
}

func (c *Config) Bind(instance any) error {
	if c == nil || c.instance == nil {
		return fmt.Errorf("❌ Config instance is nil")
	}

	if instance == nil {
		return fmt.Errorf("❌ Target instance is nil")
	}

	c.watchMutex.Lock()
	defer c.watchMutex.Unlock()

	if err := c.instance.Unmarshal(&instance); err != nil {
		return fmt.Errorf("❌ Failed to unmarshal config (path: %s, file: %s.%s): %w",
			c.opts.BasePath, c.opts.FileName, c.opts.FileType, err)
	}

	if c.opts.WatchAble {
		c.watchOnce.Do(func() {
			c.instance.WatchConfig()
			c.instance.OnConfigChange(func(e fsnotify.Event) {
				c.watchMutex.Lock()
				defer c.watchMutex.Unlock()

				if err := c.instance.Unmarshal(&instance); err != nil {
					fmt.Printf("❌ Config watch error: %v\n", err)
					return
				}

				if c.opts.OnChange != nil {
					c.opts.OnChange(e)
				}
			})
		})
	}

	return nil
}

func (c *Config) BindWithDefaults(instance any) error {
	if err := defaults.Set(instance); err != nil {
		return fmt.Errorf("❌ Failed to set defaults: %w", err)
	}

	if err := c.Bind(instance); err != nil {
		return err
	}

	if err := defaults.Set(instance); err != nil {
		return fmt.Errorf("❌ Failed to set defaults after unmarshal: %w", err)
	}

	return nil
}

func (c *Config) Validate() error {
	var instance any
	if err := c.instance.Unmarshal(&instance); err != nil {
		return fmt.Errorf("❌ Failed to unmarshal for validation: %w", err)
	}

	if v, ok := instance.(Validator); ok {
		if err := v.Validate(); err != nil {
			return fmt.Errorf("❌ Config validation failed: %w", err)
		}
	}

	return nil
}

func (c *Config) ValidateType(instance any) error {
	c.watchMutex.RLock()
	defer c.watchMutex.RUnlock()

	val := reflect.ValueOf(instance)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("❌ Config must be a struct")
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Type().Field(i)
		fieldValue := val.Field(i)

		if tag := field.Tag.Get("required"); tag == "true" {
			if fieldValue.IsZero() {
				return fmt.Errorf("❌ Required field %s is missing", field.Name)
			}
		}

		mapstructureTag := field.Tag.Get("mapstructure")
		if mapstructureTag != "" {
			configValue := c.instance.Get(mapstructureTag)
			if configValue != nil && !fieldValue.Type().AssignableTo(reflect.TypeOf(configValue)) {
				return fmt.Errorf("❌ Type mismatch for field %s: expected %s, got %T",
					field.Name, fieldValue.Type(), configValue)
			}
		}
	}

	return nil
}

func (c *Config) Export(path string) error {
	if path == "" {
		return fmt.Errorf("❌ Export path is empty")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("❌ Failed to create directory %s: %w", dir, err)
	}

	if err := c.instance.WriteConfigAs(path); err != nil {
		return fmt.Errorf("❌ Failed to write config to %s: %w", path, err)
	}

	return nil
}

func (c *Config) Snapshot() (map[string]any, error) {
	c.watchMutex.RLock()
	defer c.watchMutex.RUnlock()

	snapshot := make(map[string]any)
	if err := c.instance.Unmarshal(&snapshot); err != nil {
		return nil, fmt.Errorf("❌ Failed to create snapshot: %w", err)
	}

	c.snapshot = snapshot
	return snapshot, nil
}

func (c *Config) Restore() error {
	if c.snapshot == nil {
		return fmt.Errorf("❌ No snapshot available to restore")
	}

	return c.RestoreFrom(c.snapshot)
}

func (c *Config) RestoreFrom(snapshot map[string]any) error {
	if snapshot == nil {
		return fmt.Errorf("❌ Snapshot is nil")
	}

	c.watchMutex.Lock()
	defer c.watchMutex.Unlock()

	for k, v := range snapshot {
		c.instance.Set(k, v)
	}

	c.snapshot = snapshot
	return nil
}

func (c *Config) Get(key string) any {
	c.watchMutex.RLock()
	defer c.watchMutex.RUnlock()

	return c.instance.Get(key)
}

func (c *Config) Set(key string, value any) {
	c.watchMutex.Lock()
	defer c.watchMutex.Unlock()

	c.instance.Set(key, value)
}

func CreateConfig(opts ConfigOptions) (*viper.Viper, error) {
	configPaths := getConfigFilePaths(opts)
	if opts.LoadAll {
		configPaths = getAllConfigFilePaths(opts)
	}
	if len(configPaths) == 0 {
		return nil, fmt.Errorf("❌ No valid configuration files found in path: %s", opts.BasePath)
	}

	v := viper.New()
	v.SetConfigType(opts.FileType)

	for _, configPath := range configPaths {
		tempV := viper.New()
		tempV.SetConfigFile(configPath)
		if err := tempV.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("❌ Error reading config file %s: %w", configPath, err)
		}

		for _, key := range tempV.AllKeys() {
			v.Set(key, tempV.Get(key))
		}
	}

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	if opts.EnvPrefix != "" {
		v.SetEnvPrefix(opts.EnvPrefix)
	}
	v.AutomaticEnv()

	// Override with environment variables (higher priority than config files)
	applyEnvOverrides(v, opts.EnvPrefix)

	return v, nil
}

// applyEnvOverrides checks all config keys and overrides with environment variables if they exist.
// This ensures environment variables have higher priority than config file values.
func applyEnvOverrides(v *viper.Viper, envPrefix string) {
	replacer := strings.NewReplacer(".", "_")

	for _, key := range v.AllKeys() {
		// Convert config key to env var name: database.host -> DATABASE_HOST
		envKey := strings.ToUpper(replacer.Replace(key))
		if envPrefix != "" {
			envKey = envPrefix + "_" + envKey
		}

		// Check if environment variable exists and override
		if envValue := os.Getenv(envKey); envValue != "" {
			v.Set(key, envValue)
		}
	}
}

func getConfigFilePaths(opts ConfigOptions) (configFiles []string) {
	env := env_mode.Mode()
	fileNames := []string{
		opts.FileName,
		fmt.Sprintf("%s.local", opts.FileName),
		fmt.Sprintf("%s.%s", opts.FileName, env),
		fmt.Sprintf("%s.%s.local", opts.FileName, env),
	}

	switch env {
	case env_mode.DevMode:
		fileNames = append(fileNames, fmt.Sprintf("%s.dev", opts.FileName))
		fileNames = append(fileNames, fmt.Sprintf("%s.dev.local", opts.FileName))
		fileNames = append(fileNames, fmt.Sprintf("%s.development", opts.FileName))
		fileNames = append(fileNames, fmt.Sprintf("%s.development.local", opts.FileName))
	case env_mode.ProMode:
		fileNames = append(fileNames, fmt.Sprintf("%s.pro", opts.FileName))
		fileNames = append(fileNames, fmt.Sprintf("%s.pro.local", opts.FileName))
		fileNames = append(fileNames, fmt.Sprintf("%s.prod", opts.FileName))
		fileNames = append(fileNames, fmt.Sprintf("%s.prod.local", opts.FileName))
		fileNames = append(fileNames, fmt.Sprintf("%s.production", opts.FileName))
		fileNames = append(fileNames, fmt.Sprintf("%s.production.local", opts.FileName))
	case env_mode.TestMode:
		fileNames = append(fileNames, fmt.Sprintf("%s.test", opts.FileName))
		fileNames = append(fileNames, fmt.Sprintf("%s.test.local", opts.FileName))
	}

	for _, fileName := range fileNames {
		file := filepath.Join(opts.BasePath, fmt.Sprintf("%s.%s", fileName, opts.FileType))
		if isDir, exists, _ := utils.Exists(file); exists && !isDir {
			configFiles = append(configFiles, file)
		}
	}

	return configFiles
}

func getAllConfigFilePaths(opts ConfigOptions) (configFiles []string) {
	baseNames := getConfigBaseNames(opts.BasePath, opts.FileType)
	if len(baseNames) == 0 {
		return nil
	}

	sort.Strings(baseNames)
	baseNames = moveConfigFirst(baseNames)
	seen := make(map[string]struct{}, len(baseNames))
	for _, baseName := range baseNames {
		tempOpts := opts
		tempOpts.FileName = baseName
		tempOpts.LoadAll = false
		for _, path := range getConfigFilePaths(tempOpts) {
			if _, exists := seen[path]; exists {
				continue
			}
			seen[path] = struct{}{}
			configFiles = append(configFiles, path)
		}
	}

	return configFiles
}

func getConfigBaseNames(basePath, fileType string) []string {
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return nil
	}

	suffix := "." + fileType
	seen := make(map[string]struct{})
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, suffix) {
			continue
		}
		base := strings.TrimSuffix(name, suffix)
		base = stripConfigSuffix(base)
		if base == "" {
			continue
		}
		seen[base] = struct{}{}
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	return names
}

func stripConfigSuffix(name string) string {
	name = strings.TrimSuffix(name, ".local")
	return trimEnvSuffix(name)
}

func trimEnvSuffix(name string) string {
	envSuffixes := []string{
		".dev",
		".development",
		".pro",
		".prod",
		".production",
		".test",
	}
	for _, suffix := range envSuffixes {
		if strings.HasSuffix(name, suffix) {
			return strings.TrimSuffix(name, suffix)
		}
	}
	return name
}

func moveConfigFirst(names []string) []string {
	configIndex := -1
	for i, name := range names {
		if name == "config" {
			configIndex = i
			break
		}
	}

	if configIndex <= 0 {
		return names
	}

	out := make([]string, 0, len(names))
	out = append(out, "config")
	out = append(out, names[:configIndex]...)
	out = append(out, names[configIndex+1:]...)
	return out
}
