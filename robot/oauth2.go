package robot

// OAuth2LinkRequest defines the token metadata needed to link a user account
// to a configured OAuth2 provider.
type OAuth2LinkRequest struct {
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
