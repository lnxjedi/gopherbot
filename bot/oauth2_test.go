package bot

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

func setupOAuth2BrainTest(t *testing.T, providers map[string]OAuth2ProviderConfig, parameterSets map[string]ParameterSet) *memBrain {
	t.Helper()

	oldBrain := interfaces.brain
	testBrain := &memBrain{memories: make(map[string]*[]byte)}
	interfaces.brain = testBrain
	t.Cleanup(func() {
		interfaces.brain = oldBrain
	})

	cryptKey.Lock()
	oldKey := append([]byte(nil), cryptKey.key...)
	oldInitialized := cryptKey.initialized
	oldInitializing := cryptKey.initializing
	cryptKey.key = []byte("0123456789abcdef0123456789abcdef")
	cryptKey.initialized = true
	cryptKey.initializing = false
	cryptKey.Unlock()
	t.Cleanup(func() {
		cryptKey.Lock()
		cryptKey.key = oldKey
		cryptKey.initialized = oldInitialized
		cryptKey.initializing = oldInitializing
		cryptKey.Unlock()
	})

	currentCfg.Lock()
	oldProviders := currentCfg.oauth2Providers
	oldParameterSets := currentCfg.parameterSets
	currentCfg.oauth2Providers = providers
	currentCfg.parameterSets = parameterSets
	currentCfg.Unlock()
	t.Cleanup(func() {
		currentCfg.Lock()
		currentCfg.oauth2Providers = oldProviders
		currentCfg.parameterSets = oldParameterSets
		currentCfg.Unlock()
	})

	done := make(chan struct{})
	go func() {
		runBrain()
		close(done)
	}()
	t.Cleanup(func() {
		brainQuit()
		<-done
	})

	return testBrain
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (rt roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return rt(req)
}

func TestLinkOAuth2UserAndGetOAuth2TokenReturnsStoredAccessToken(t *testing.T) {
	providers := map[string]OAuth2ProviderConfig{
		"github-enterprise": {
			Key:                    "github-enterprise",
			CredentialParameterSet: "github_oauth",
		},
	}
	testBrain := setupOAuth2BrainTest(t, providers, map[string]ParameterSet{
		"github_oauth": {
			Parameters: []Parameter{
				{Name: "CLIENT_ID", Value: "client-id"},
				{Name: "CLIENT_SECRET", Value: "client-secret"},
			},
		},
	})

	r := Robot{}
	ret := r.LinkOAuth2User(&robot.OAuth2LinkRequest{
		Provider:     "github-enterprise",
		User:         "Alice",
		AccessToken:  "stored-access-token",
		TokenType:    "Bearer",
		SubjectLogin: "alice-gh",
	})
	if ret != robot.Ok {
		t.Fatalf("LinkOAuth2User ret = %v, want Ok", ret)
	}

	token, ret := r.GetOAuth2Token("github-enterprise", "alice")
	if ret != robot.Ok {
		t.Fatalf("GetOAuth2Token ret = %v, want Ok", ret)
	}
	if token != "stored-access-token" {
		t.Fatalf("GetOAuth2Token token = %q, want %q", token, "stored-access-token")
	}

	found := false
	for key := range testBrain.memories {
		if strings.HasPrefix(key, "bot:oauth2:v1:github_enterprise:user:") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("stored datum key did not use sanitized provider segment; keys=%v", testBrain.memories)
	}
}

func TestGetOAuth2TokenRefreshesExpiredTokenAndRotatesRefreshToken(t *testing.T) {
	var seenForm url.Values
	oldClientFactory := oauth2HTTPClientFactory
	oauth2HTTPClientFactory = func() *http.Client {
		return &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPost {
					t.Fatalf("token endpoint method = %s, want POST", req.Method)
				}
				body, err := io.ReadAll(req.Body)
				if err != nil {
					t.Fatalf("ReadAll request body failed: %v", err)
				}
				seenForm, err = url.ParseQuery(string(body))
				if err != nil {
					t.Fatalf("ParseQuery failed: %v", err)
				}
				payload, err := json.Marshal(map[string]interface{}{
					"access_token":  "fresh-access-token",
					"refresh_token": "rotated-refresh-token",
					"expires_in":    3600,
				})
				if err != nil {
					t.Fatalf("Marshal response failed: %v", err)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(bytes.NewReader(payload)),
				}, nil
			}),
		}
	}
	t.Cleanup(func() {
		oauth2HTTPClientFactory = oldClientFactory
	})

	providers := map[string]OAuth2ProviderConfig{
		"github": {
			Key:                    "github",
			CredentialParameterSet: "github_oauth",
			Token: OAuth2TokenEndpointConfig{
				OAuth2EndpointConfig: OAuth2EndpointConfig{
					URL: "https://oauth2.invalid/token",
				},
			},
		},
	}
	setupOAuth2BrainTest(t, providers, map[string]ParameterSet{
		"github_oauth": {
			Parameters: []Parameter{
				{Name: "CLIENT_ID", Value: "client-id"},
				{Name: "CLIENT_SECRET", Value: "client-secret"},
			},
		},
	})

	r := Robot{}
	ret := r.LinkOAuth2User(&robot.OAuth2LinkRequest{
		Provider:     "github",
		User:         "alice",
		AccessToken:  "expired-access-token",
		RefreshToken: "old-refresh-token",
		TokenType:    "Bearer",
		Scope:        []string{"repo"},
		ExpiresAt:    time.Now().Add(-time.Minute).UTC().Format(time.RFC3339),
		GrantType:    "device_code",
	})
	if ret != robot.Ok {
		t.Fatalf("LinkOAuth2User ret = %v, want Ok", ret)
	}

	token, ret := r.GetOAuth2Token("github", "alice")
	if ret != robot.Ok {
		t.Fatalf("GetOAuth2Token ret = %v, want Ok", ret)
	}
	if token != "fresh-access-token" {
		t.Fatalf("GetOAuth2Token token = %q, want %q", token, "fresh-access-token")
	}
	if got := seenForm.Get("grant_type"); got != "refresh_token" {
		t.Fatalf("grant_type = %q, want %q", got, "refresh_token")
	}
	if got := seenForm.Get("refresh_token"); got != "old-refresh-token" {
		t.Fatalf("refresh_token = %q, want %q", got, "old-refresh-token")
	}
	if got := seenForm.Get("client_id"); got != "client-id" {
		t.Fatalf("client_id = %q, want %q", got, "client-id")
	}
	if got := seenForm.Get("client_secret"); got != "client-secret" {
		t.Fatalf("client_secret = %q, want %q", got, "client-secret")
	}

	var state oauth2UserLink
	_, exists, ret := checkoutDatum(oauth2UserDatumKey("github", "alice"), &state, false)
	if ret != robot.Ok || !exists {
		t.Fatalf("checkout refreshed datum ret=%v exists=%t, want Ok/true", ret, exists)
	}
	if state.Token.AccessToken != "fresh-access-token" {
		t.Fatalf("stored access token = %q, want %q", state.Token.AccessToken, "fresh-access-token")
	}
	if state.Token.RefreshToken != "rotated-refresh-token" {
		t.Fatalf("stored refresh token = %q, want %q", state.Token.RefreshToken, "rotated-refresh-token")
	}
	if state.Token.TokenType != "Bearer" {
		t.Fatalf("stored token type = %q, want %q", state.Token.TokenType, "Bearer")
	}
	if len(state.Token.Scope) != 1 || state.Token.Scope[0] != "repo" {
		t.Fatalf("stored scopes = %v, want [repo]", state.Token.Scope)
	}
	if state.Grant.RotationCounter != 1 {
		t.Fatalf("rotation counter = %d, want 1", state.Grant.RotationCounter)
	}
	if state.Grant.LastRefreshAt.IsZero() {
		t.Fatal("LastRefreshAt was not set")
	}
	if state.Status.ReauthRequired {
		t.Fatal("ReauthRequired = true, want false")
	}
	if state.Status.LastErrorCode != 0 || state.Status.LastError != "" || state.Status.LastErrorAt != nil {
		t.Fatalf("unexpected error status after refresh: %+v", state.Status)
	}
	if state.Token.ExpiresAt == nil || !state.Token.ExpiresAt.After(time.Now().UTC()) {
		t.Fatalf("ExpiresAt = %v, want a future timestamp", state.Token.ExpiresAt)
	}
}

func TestGetOAuth2TokenReturnsConfigErrorWhenCredentialParameterSetIsMissing(t *testing.T) {
	setupOAuth2BrainTest(t, map[string]OAuth2ProviderConfig{
		"github": {
			Key:                    "github",
			CredentialParameterSet: "github_oauth",
			Token: OAuth2TokenEndpointConfig{
				OAuth2EndpointConfig: OAuth2EndpointConfig{
					URL: "https://oauth2.invalid/token",
				},
			},
		},
	}, nil)

	ret := Robot{}.LinkOAuth2User(&robot.OAuth2LinkRequest{
		Provider:     "github",
		User:         "alice",
		AccessToken:  "expired-access-token",
		RefreshToken: "old-refresh-token",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(-time.Minute).UTC().Format(time.RFC3339),
	})
	if ret != robot.Ok {
		t.Fatalf("LinkOAuth2User ret = %v, want Ok", ret)
	}

	token, ret := Robot{}.GetOAuth2Token("github", "alice")
	if ret != robot.OAuth2ConfigError {
		t.Fatalf("GetOAuth2Token ret = %v, want OAuth2ConfigError", ret)
	}
	if token != "" {
		t.Fatalf("GetOAuth2Token token = %q, want empty string", token)
	}
}
