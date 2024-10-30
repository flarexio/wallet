package conf

import "fmt"

type Config struct {
	Keys KeyConfig `yaml:"keys"`
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
