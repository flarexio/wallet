package wallet

type Config struct {
	PasskeysConfig PasskeysConfig `yaml:"passkeys"`
}

type PasskeysConfig struct {
	BaseURL    string     `yaml:"baseURL"`
	TenantID   string     `yaml:"tenantID"`
	PasskeyAPI PasskeyAPI `yaml:"api"`
	Origins    []string   `yaml:"origins"`
}

type PasskeyAPI struct {
	ID     string `yaml:"id"`
	Secret string `yaml:"secret"`
}
