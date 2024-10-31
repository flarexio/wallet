package conf

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

var (
	Path string
)

type Config struct {
	Keys         KeyConfig         `yaml:"keys"`
	Persistences PersistenceConfig `yaml:"persistences"`
}

type KeyConfig struct {
	Google GoogleKeyConfig `yaml:"google"`
}

type GoogleKeyConfig struct {
	ProjectID string `yaml:"projectID"`
	Location  string `yaml:"location"`
	KeyRing   string `yaml:"keyRing"`
	Key       string `yaml:"key"`
}

func (key GoogleKeyConfig) Path() string {
	return fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s",
		key.ProjectID, key.Location, key.KeyRing, key.Key)
}

type PersistenceConfig struct {
	Cache CacheConfig `yaml:"cache"`
	Main  MainConfig  `yaml:"main"`
}

type CacheConfig struct {
	Enabled bool
	Name    string
	Path    string
	InMem   bool
}

func (cfg *CacheConfig) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		Enabled bool   `yaml:"enabled"`
		Name    string `yaml:"name"`
		Path    string `yaml:"path"`
		InMem   bool   `yaml:"inmem"`
	}

	if err := value.Decode(&raw); err != nil {
		return err
	}

	cfg.Enabled = raw.Enabled
	cfg.Name = raw.Name
	cfg.InMem = raw.InMem

	cfg.Path = raw.Path
	if raw.Path == "" {
		cfg.Path = Path
	}

	return nil
}

type MainConfig struct {
	Enabled bool
	RPC     string
	Program string
	Path    string
	Account string
}

func (cfg *MainConfig) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		Enabled bool   `yaml:"enabled"`
		RPC     string `yaml:"rpc"`
		Program string `yaml:"program"`
		Path    string `yaml:"path"`
		Account string `yaml:"account"`
	}

	if err := value.Decode(&raw); err != nil {
		return err
	}

	cfg.Enabled = raw.Enabled
	cfg.RPC = raw.RPC
	cfg.Program = raw.Program

	cfg.Path = raw.Path
	if raw.Path == "" {
		cfg.Path = Path
	}

	cfg.Account = raw.Account

	return nil
}
