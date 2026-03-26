package bot

import "time"

type OAuth2ProviderConfig struct {
	Key                     string                    `yaml:"-"`
	DisplayName             string                    `yaml:"DisplayName"`
	ClientID                string                    `yaml:"ClientID"`
	ClientSecret            string                    `yaml:"ClientSecret"`
	DefaultScopes           []string                  `yaml:"DefaultScopes"`
	TokenEndpointAuthMethod string                    `yaml:"TokenEndpointAuthMethod"`
	DeviceAuthorization     OAuth2EndpointConfig      `yaml:"DeviceAuthorization"`
	Token                   OAuth2TokenEndpointConfig `yaml:"Token"`
	UserInfo                OAuth2EndpointConfig      `yaml:"UserInfo"`
}

type OAuth2EndpointConfig struct {
	URL        string            `yaml:"URL"`
	Headers    map[string]string `yaml:"Headers"`
	Parameters map[string]string `yaml:"Parameters"`
}

type OAuth2TokenEndpointConfig struct {
	OAuth2EndpointConfig `yaml:",inline"`
	RefreshParameters    map[string]string `yaml:"RefreshParameters"`
}

type oauth2Subject struct {
	ID    string `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type oauth2TokenState struct {
	AccessToken      string     `json:"access_token"`
	RefreshToken     string     `json:"refresh_token,omitempty"`
	TokenType        string     `json:"token_type,omitempty"`
	Scope            []string   `json:"scope,omitempty"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	RefreshExpiresAt *time.Time `json:"refresh_expires_at,omitempty"`
}

type oauth2GrantState struct {
	Type            string    `json:"type,omitempty"`
	LinkedAt        time.Time `json:"linked_at"`
	LastRefreshAt   time.Time `json:"last_refresh_at,omitempty"`
	RotationCounter int       `json:"rotation_counter,omitempty"`
}

type oauth2StatusState struct {
	ReauthRequired bool       `json:"reauth_required,omitempty"`
	LastErrorCode  int        `json:"last_error_code,omitempty"`
	LastError      string     `json:"last_error,omitempty"`
	LastErrorAt    *time.Time `json:"last_error_at,omitempty"`
}

type oauth2UserLink struct {
	SchemaVersion int               `json:"schema_version"`
	ProviderKey   string            `json:"provider_key"`
	Username      string            `json:"username"`
	Subject       oauth2Subject     `json:"subject,omitempty"`
	Token         oauth2TokenState  `json:"token"`
	Grant         oauth2GrantState  `json:"grant"`
	Status        oauth2StatusState `json:"status,omitempty"`
}
