package conf

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/flarexio/identity/conf"
)

var (
	Path string
)

type Config struct {
	Keys        KeyConfig             `yaml:"keys"`
	Persistence PersistenceConfig     `yaml:"persistence"`
	Identity    IdentityConfig        `yaml:"identity"`
	Passkeys    conf.PasskeysProvider `yaml:"passkeys"`
}

type KeyConfig struct {
	Google  GoogleKeyConfig  `yaml:"google"`
	Session SessionKeyConfig `yaml:"session"`
}

type GoogleKeyConfig struct {
	ProjectID string `yaml:"projectID"`
	Location  string `yaml:"location"`
	KeyRing   string `yaml:"keyRing"`
	Key       string `yaml:"key"`
}

type SessionKeyConfig struct {
	Key [32]byte `yaml:"key"`
}

func (key GoogleKeyConfig) Path() string {
	return fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s",
		key.ProjectID, key.Location, key.KeyRing, key.Key)
}

type PersistenceDriver int

const (
	PersistenceDriverBadger PersistenceDriver = iota
	PersistenceDriverSolana
	PersistenceDriverComposite
)

func ParsePersistenceDriver(value string) (PersistenceDriver, error) {
	switch value {
	case "badger":
		return PersistenceDriverBadger, nil
	case "solana":
		return PersistenceDriverSolana, nil
	case "composite":
		return PersistenceDriverComposite, nil
	default:
		return -1, fmt.Errorf("unknown persistence driver")
	}
}

type PersistenceConfig struct {
	Driver    PersistenceDriver
	Badger    *BadgerPersistenceConfig
	Solana    *SolanaPersistenceConfig
	Composite *CompositePersistenceConfig
}

func (cfg *PersistenceConfig) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		Driver    string                      `yaml:"driver"`
		Badger    *BadgerPersistenceConfig    `yaml:"badger"`
		Solana    *SolanaPersistenceConfig    `yaml:"solana"`
		Composite *CompositePersistenceConfig `yaml:"composite"`
	}

	if err := value.Decode(&raw); err != nil {
		return err
	}

	driver, err := ParsePersistenceDriver(raw.Driver)
	if err != nil {
		return err
	}

	cfg.Driver = driver
	cfg.Badger = raw.Badger
	cfg.Solana = raw.Solana
	cfg.Composite = raw.Composite

	return nil
}

type CompositePersistenceConfig struct {
	Main  PersistenceConfig `yaml:"main"`
	Cache PersistenceConfig `yaml:"cache"`
}

type BadgerPersistenceConfig struct {
	Name  string
	Path  string
	InMem bool
}

func (cfg *BadgerPersistenceConfig) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		Name  string `yaml:"name"`
		Path  string `yaml:"path"`
		InMem bool   `yaml:"inmem"`
	}

	if err := value.Decode(&raw); err != nil {
		return err
	}

	cfg.Name = raw.Name
	cfg.InMem = raw.InMem

	cfg.Path = raw.Path
	if raw.Path == "" {
		cfg.Path = Path
	}

	return nil
}

type SolanaPersistenceConfig struct {
	RPC     string
	Program string
	Path    string
	Account string
}

func (cfg *SolanaPersistenceConfig) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		RPC     string `yaml:"rpc"`
		Program string `yaml:"program"`
		Path    string `yaml:"path"`
		Account string `yaml:"account"`
	}

	if err := value.Decode(&raw); err != nil {
		return err
	}

	cfg.RPC = raw.RPC
	cfg.Program = raw.Program

	cfg.Path = raw.Path
	if raw.Path == "" {
		cfg.Path = Path
	}

	cfg.Account = raw.Account

	return nil
}

type IdentityConfig struct {
	BaseURL string    `yaml:"baseURL"`
	JWT     JWTConfig `yaml:"jwt"`
}

type JWTConfig struct {
	Secret    []byte   `yaml:"secret"`
	Audiences []string `yaml:"audiences"`
}

func (cfg *JWTConfig) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		Secret    string   `yaml:"secret"`
		Audiences []string `yaml:"audiences"`
	}

	if err := value.Decode(&raw); err != nil {
		return err
	}

	cfg.Secret = []byte(raw.Secret)
	cfg.Audiences = raw.Audiences

	return nil
}
