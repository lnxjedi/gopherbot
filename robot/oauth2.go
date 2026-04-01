package robot

// IdentityCredential is a language-neutral credential envelope for extension
// authors. The engine fills both the raw value and the most common header
// presentation so callers can use whichever is more convenient.
type IdentityCredential struct {
	Type        string            `json:"type"`
	Value       string            `json:"value"`
	Scheme      string            `json:"scheme,omitempty"`
	HeaderName  string            `json:"header_name,omitempty"`
	HeaderValue string            `json:"header_value,omitempty"`
	ExpiresAt   string            `json:"expires_at,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// OAuth2IdentityLinkRequest defines the token metadata needed to link a user
// account to an OAuth2-backed identity provider.
type OAuth2IdentityLinkRequest struct {
	Provider         string
	User             string
	AccessToken      string
	RefreshToken     string
	TokenType        string
	Scope            []string
	ExpiresIn        int
	ExpiresAt        string
	RefreshExpiresIn int
	RefreshExpiresAt string
	GrantType        string
	SubjectID        string
	SubjectLogin     string
	SubjectName      string
	SubjectEmail     string
}
