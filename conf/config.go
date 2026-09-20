package conf

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/flarexio/identity/conf"
)

var (
	Path string
)

type Config struct {
	Keys        KeyConfig             `yaml:"keys"`
	Persistence PersistenceConfig     `yaml:"persistence"`
	Backup      BackupConfig          `yaml:"backup"`
	JWT         JWTConfig             `yaml:"jwt"`
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

type JWTConfig struct {
	Issuer   string `yaml:"issuer"`
	Audience string `yaml:"audience"`
	JWKsURL  string `yaml:"jwksURL"`
}

// DefaultBackupKeep is how many scheduled runs are left at the destination
// when the config does not say. Four weeks of six-hourly runs is long enough
// that a corruption introduced over a weekend is still recoverable from.
const DefaultBackupKeep = 28

type BackupDestinationDriver int

// BackupDestinationNone is the zero value, so a config that schedules backups
// without naming somewhere to put them fails as exactly that rather than as a
// directory with no path.
const (
	BackupDestinationNone BackupDestinationDriver = iota
	BackupDestinationDir
	BackupDestinationGCS
)

func ParseBackupDestinationDriver(value string) (BackupDestinationDriver, error) {
	switch value {
	case "dir":
		return BackupDestinationDir, nil

	case "gcs":
		return BackupDestinationGCS, nil

	default:
		return 0, fmt.Errorf("invalid backup destination driver: %s", value)
	}
}

// BackupConfig schedules the backup the running service takes of itself. A
// zero Interval leaves it off, which is the default: shipping snapshots
// somewhere needs a destination the operator chose.
type BackupConfig struct {
	Interval    time.Duration
	Keep        int
	Destination BackupDestinationConfig
}

func (cfg BackupConfig) Enabled() bool {
	return cfg.Interval > 0
}

func (cfg *BackupConfig) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		Interval    string                  `yaml:"interval"`
		Keep        int                     `yaml:"keep"`
		Destination BackupDestinationConfig `yaml:"destination"`
	}

	if err := value.Decode(&raw); err != nil {
		return err
	}

	if raw.Interval != "" {
		interval, err := time.ParseDuration(raw.Interval)
		if err != nil {
			return fmt.Errorf("backup interval: %w", err)
		}

		cfg.Interval = interval
	}

	cfg.Keep = raw.Keep
	if raw.Keep == 0 {
		cfg.Keep = DefaultBackupKeep
	}

	cfg.Destination = raw.Destination

	return nil
}

type BackupDestinationConfig struct {
	Driver BackupDestinationDriver
	Dir    *DirBackupDestinationConfig
	GCS    *GCSBackupDestinationConfig
}

func (cfg *BackupDestinationConfig) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		Driver string                      `yaml:"driver"`
		Dir    *DirBackupDestinationConfig `yaml:"dir"`
		GCS    *GCSBackupDestinationConfig `yaml:"gcs"`
	}

	if err := value.Decode(&raw); err != nil {
		return err
	}

	// An absent destination is not an error while backups are off; serve
	// checks for one only when the interval asks for it.
	if raw.Driver == "" {
		return nil
	}

	driver, err := ParseBackupDestinationDriver(raw.Driver)
	if err != nil {
		return err
	}

	cfg.Driver = driver
	cfg.Dir = raw.Dir
	cfg.GCS = raw.GCS

	return nil
}

type DirBackupDestinationConfig struct {
	Path string `yaml:"path"`
}

type GCSBackupDestinationConfig struct {
	Bucket string `yaml:"bucket"`
	Prefix string `yaml:"prefix"`
}
