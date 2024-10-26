package wallet

type Config struct {
	Providers Providers `yaml:"providers"`
}

type Providers struct {
	Google   GoogleConfig   `yaml:"google"`
	Passkeys PasskeysConfig `yaml:"passkeys"`
}

type GoogleConfig struct {
	Client OAuthAPI `yaml:"client"`
}

type PasskeysConfig struct {
	BaseURL    string   `yaml:"baseURL"`
	TenantID   string   `yaml:"tenantID"`
	PasskeyAPI OAuthAPI `yaml:"api"`
	Origins    []string `yaml:"origins"`
}

type OAuthAPI struct {
	ID     string `yaml:"id"`
	Secret string `yaml:"secret"`
}
