package bot

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

const (
	oauth2DatumSchemaVersion = 1
	oauth2ExpiryLeeway       = 30 * time.Second
	oauth2HTTPTimeout        = 20 * time.Second
)

var oauth2HTTPClientFactory = func() *http.Client {
	return &http.Client{Timeout: oauth2HTTPTimeout}
}

type oauth2TokenHTTPResponse struct {
	AccessToken           string      `json:"access_token"`
	RefreshToken          string      `json:"refresh_token"`
	TokenType             string      `json:"token_type"`
	Scope                 interface{} `json:"scope"`
	ExpiresIn             int         `json:"expires_in"`
	RefreshTokenExpiresIn int         `json:"refresh_token_expires_in"`
	Error                 string      `json:"error"`
	ErrorDescription      string      `json:"error_description"`
	ErrorURI              string      `json:"error_uri"`
	Message               string      `json:"message"`
	Interval              int         `json:"interval"`
}

type oauth2ClientCredentials struct {
	ClientID     string
	ClientSecret string
}

func normalizeOAuth2ProviderKey(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}

var oauth2DatumKeySanitizer = regexp.MustCompile(`[^A-Za-z0-9_]`)

func oauth2DatumKeySegment(value string) string {
	value = normalizeOAuth2ProviderKey(value)
	if value == "" {
		return "provider"
	}
	value = oauth2DatumKeySanitizer.ReplaceAllString(value, "_")
	if value == "" {
		return "provider"
	}
	return value
}

func getOAuth2ProviderConfig(provider string) (OAuth2ProviderConfig, bool) {
	key := normalizeOAuth2ProviderKey(provider)
	currentCfg.RLock()
	cfg, ok := currentCfg.oauth2Providers[key]
	currentCfg.RUnlock()
	return cfg, ok
}

func oauth2UserDatumKey(provider, user string) string {
	sum := sha1.Sum([]byte(strings.ToLower(strings.TrimSpace(user))))
	return fmt.Sprintf("bot:oauth2:v1:%s:user:%s", oauth2DatumKeySegment(provider), hex.EncodeToString(sum[:]))
}

func getOAuth2CredentialParameterSet(name string) (ParameterSet, bool) {
	setName := strings.TrimSpace(name)
	if setName == "" {
		return ParameterSet{}, false
	}
	currentCfg.RLock()
	ps, ok := currentCfg.parameterSets[setName]
	currentCfg.RUnlock()
	return ps, ok
}

func oauth2GetParameterValue(params []Parameter, name string) string {
	for _, param := range params {
		if strings.EqualFold(strings.TrimSpace(param.Name), name) {
			return strings.TrimSpace(param.Value)
		}
	}
	return ""
}

func oauth2ResolveClientCredentials(provider OAuth2ProviderConfig) (oauth2ClientCredentials, robot.RetVal, string) {
	setName := strings.TrimSpace(provider.CredentialParameterSet)
	if setName == "" {
		return oauth2ClientCredentials{}, robot.OAuth2ConfigError, "provider missing CredentialParameterSet"
	}
	ps, ok := getOAuth2CredentialParameterSet(setName)
	if !ok {
		return oauth2ClientCredentials{}, robot.OAuth2ConfigError, fmt.Sprintf("provider credential ParameterSet %q not found", setName)
	}
	creds := oauth2ClientCredentials{
		ClientID:     oauth2GetParameterValue(ps.Parameters, "CLIENT_ID"),
		ClientSecret: oauth2GetParameterValue(ps.Parameters, "CLIENT_SECRET"),
	}
	if creds.ClientID == "" {
		return oauth2ClientCredentials{}, robot.OAuth2ConfigError, fmt.Sprintf("provider credential ParameterSet %q is missing CLIENT_ID", setName)
	}
	return creds, robot.Ok, ""
}

func oauth2TimePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	u := t.UTC()
	return &u
}

func oauth2ResolveTime(expiresAt string, expiresIn int, now time.Time) (*time.Time, error) {
	if strings.TrimSpace(expiresAt) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(expiresAt))
		if err != nil {
			return nil, err
		}
		return oauth2TimePtr(parsed), nil
	}
	if expiresIn > 0 {
		return oauth2TimePtr(now.Add(time.Duration(expiresIn) * time.Second)), nil
	}
	return nil, nil
}

func oauth2NormalizeScopes(raw interface{}) []string {
	switch v := raw.(type) {
	case nil:
		return nil
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		fields := strings.FieldsFunc(v, func(r rune) bool {
			return r == ' ' || r == ','
		})
		out := make([]string, 0, len(fields))
		for _, field := range fields {
			field = strings.TrimSpace(field)
			if field != "" {
				out = append(out, field)
			}
		}
		return out
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					out = append(out, s)
				}
			}
		}
		return out
	default:
		return nil
	}
}

func oauth2TokenUsable(state *oauth2UserLink, now time.Time) bool {
	if state == nil || strings.TrimSpace(state.Token.AccessToken) == "" {
		return false
	}
	if state.Token.ExpiresAt == nil || state.Token.ExpiresAt.IsZero() {
		return true
	}
	return now.Add(oauth2ExpiryLeeway).Before(state.Token.ExpiresAt.UTC())
}

func oauth2SetError(state *oauth2UserLink, code robot.RetVal, msg string, reauth bool, now time.Time) {
	state.Status.LastErrorCode = int(code)
	state.Status.LastError = strings.TrimSpace(msg)
	state.Status.LastErrorAt = oauth2TimePtr(now)
	state.Status.ReauthRequired = reauth
}

func oauth2ClearError(state *oauth2UserLink) {
	state.Status.LastErrorCode = 0
	state.Status.LastError = ""
	state.Status.LastErrorAt = nil
	state.Status.ReauthRequired = false
}

func oauth2BasicAuthValue(clientID, clientSecret string) string {
	raw := clientID + ":" + clientSecret
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(raw))
}

func oauth2AuthMethod(provider OAuth2ProviderConfig) string {
	method := strings.TrimSpace(strings.ToLower(provider.TokenEndpointAuthMethod))
	if method == "" {
		return "client_secret_post"
	}
	return method
}

func oauth2ApplyEndpointDefaults(values url.Values, endpoint OAuth2EndpointConfig) {
	for key, value := range endpoint.Parameters {
		if strings.TrimSpace(key) == "" || value == "" {
			continue
		}
		if _, ok := values[key]; !ok {
			values.Set(key, value)
		}
	}
}

func oauth2DoFormPost(endpoint OAuth2EndpointConfig, headers map[string]string, values url.Values) ([]byte, int, error) {
	oauth2ApplyEndpointDefaults(values, endpoint)
	body := values.Encode()
	req, err := http.NewRequest(http.MethodPost, endpoint.URL, bytes.NewBufferString(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for key, value := range endpoint.Headers {
		if key != "" && value != "" {
			req.Header.Set(key, value)
		}
	}
	for key, value := range headers {
		if key != "" && value != "" {
			req.Header.Set(key, value)
		}
	}
	client := oauth2HTTPClientFactory()
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return payload, resp.StatusCode, nil
}

func oauth2DecodeJSON(payload []byte, out interface{}) error {
	if len(bytes.TrimSpace(payload)) == 0 {
		return io.EOF
	}
	return json.Unmarshal(payload, out)
}

func oauth2ProviderErrorMessage(resp oauth2TokenHTTPResponse) string {
	parts := []string{}
	if resp.Error != "" {
		parts = append(parts, resp.Error)
	}
	if resp.ErrorDescription != "" {
		parts = append(parts, resp.ErrorDescription)
	}
	if resp.Message != "" {
		parts = append(parts, resp.Message)
	}
	if resp.ErrorURI != "" {
		parts = append(parts, resp.ErrorURI)
	}
	if len(parts) == 0 {
		return "provider returned an empty error response"
	}
	return strings.Join(parts, " - ")
}

func oauth2ReauthError(providerErr string) bool {
	switch strings.ToLower(strings.TrimSpace(providerErr)) {
	case "invalid_grant", "expired_token", "access_denied", "invalid_refresh_token":
		return true
	default:
		return false
	}
}

func oauth2ProviderConfigValid(provider OAuth2ProviderConfig) robot.RetVal {
	if provider.Key == "" || strings.TrimSpace(provider.CredentialParameterSet) == "" {
		return robot.OAuth2ConfigError
	}
	return robot.Ok
}

func oauth2RefreshUserToken(provider OAuth2ProviderConfig, state *oauth2UserLink, now time.Time) robot.RetVal {
	if ret := oauth2ProviderConfigValid(provider); ret != robot.Ok {
		oauth2SetError(state, ret, "provider missing key or credential ParameterSet", false, now)
		return ret
	}
	if strings.TrimSpace(provider.Token.URL) == "" {
		oauth2SetError(state, robot.OAuth2ConfigError, "provider missing token endpoint URL", false, now)
		return robot.OAuth2ConfigError
	}
	if strings.TrimSpace(state.Token.RefreshToken) == "" {
		oauth2SetError(state, robot.OAuth2ReauthRequired, "token expired and no refresh token is stored", true, now)
		return robot.OAuth2ReauthRequired
	}
	creds, ret, msg := oauth2ResolveClientCredentials(provider)
	if ret != robot.Ok {
		oauth2SetError(state, ret, msg, false, now)
		return ret
	}

	values := url.Values{}
	values.Set("grant_type", "refresh_token")
	values.Set("refresh_token", state.Token.RefreshToken)
	for key, value := range provider.Token.RefreshParameters {
		if strings.TrimSpace(key) != "" && value != "" {
			values.Set(key, value)
		}
	}

	headers := map[string]string{}
	switch oauth2AuthMethod(provider) {
	case "client_secret_basic":
		if creds.ClientSecret == "" {
			oauth2SetError(state, robot.OAuth2ConfigError, "provider requires client_secret_basic but CLIENT_SECRET is empty", false, now)
			return robot.OAuth2ConfigError
		}
		headers["Authorization"] = oauth2BasicAuthValue(creds.ClientID, creds.ClientSecret)
	case "client_id_only":
		values.Set("client_id", creds.ClientID)
	case "none":
		// nothing
	default:
		values.Set("client_id", creds.ClientID)
		if creds.ClientSecret != "" {
			values.Set("client_secret", creds.ClientSecret)
		}
	}

	payload, statusCode, err := oauth2DoFormPost(provider.Token.OAuth2EndpointConfig, headers, values)
	if err != nil {
		oauth2SetError(state, robot.OAuth2RefreshFailed, fmt.Sprintf("refresh HTTP request failed: %v", err), false, now)
		return robot.OAuth2RefreshFailed
	}

	var tokenResp oauth2TokenHTTPResponse
	if err := oauth2DecodeJSON(payload, &tokenResp); err != nil {
		oauth2SetError(state, robot.OAuth2RefreshFailed, fmt.Sprintf("refresh response decode failed (%d): %v", statusCode, err), false, now)
		return robot.OAuth2RefreshFailed
	}
	if tokenResp.Error != "" {
		msg := oauth2ProviderErrorMessage(tokenResp)
		if oauth2ReauthError(tokenResp.Error) {
			oauth2SetError(state, robot.OAuth2ReauthRequired, msg, true, now)
			return robot.OAuth2ReauthRequired
		}
		oauth2SetError(state, robot.OAuth2RefreshFailed, msg, false, now)
		return robot.OAuth2RefreshFailed
	}
	if strings.TrimSpace(tokenResp.AccessToken) == "" {
		oauth2SetError(state, robot.OAuth2RefreshFailed, fmt.Sprintf("refresh response missing access_token (status %d)", statusCode), false, now)
		return robot.OAuth2RefreshFailed
	}

	expiresAt, err := oauth2ResolveTime("", tokenResp.ExpiresIn, now)
	if err != nil {
		oauth2SetError(state, robot.OAuth2RefreshFailed, fmt.Sprintf("refresh expires_in conversion failed: %v", err), false, now)
		return robot.OAuth2RefreshFailed
	}
	refreshExpiresAt, err := oauth2ResolveTime("", tokenResp.RefreshTokenExpiresIn, now)
	if err != nil {
		oauth2SetError(state, robot.OAuth2RefreshFailed, fmt.Sprintf("refresh token expiry conversion failed: %v", err), false, now)
		return robot.OAuth2RefreshFailed
	}

	state.Token.AccessToken = tokenResp.AccessToken
	if tokenResp.RefreshToken != "" {
		if tokenResp.RefreshToken != state.Token.RefreshToken {
			state.Grant.RotationCounter++
		}
		state.Token.RefreshToken = tokenResp.RefreshToken
	}
	if strings.TrimSpace(tokenResp.TokenType) != "" {
		state.Token.TokenType = strings.TrimSpace(tokenResp.TokenType)
	}
	if scopes := oauth2NormalizeScopes(tokenResp.Scope); len(scopes) > 0 {
		state.Token.Scope = scopes
	}
	state.Token.ExpiresAt = expiresAt
	if refreshExpiresAt != nil {
		state.Token.RefreshExpiresAt = refreshExpiresAt
	}
	state.Grant.LastRefreshAt = now.UTC()
	oauth2ClearError(state)
	return robot.Ok
}

func (r Robot) GetOAuth2Token(provider, user string) (token string, ret robot.RetVal) {
	cfg, ok := getOAuth2ProviderConfig(provider)
	if !ok {
		return "", robot.OAuth2ProviderNotFound
	}
	if ret = oauth2ProviderConfigValid(cfg); ret != robot.Ok {
		return "", ret
	}
	user = strings.ToLower(strings.TrimSpace(user))
	if user == "" {
		return "", robot.OAuth2InvalidLinkRequest
	}
	key := oauth2UserDatumKey(cfg.Key, user)
	now := time.Now().UTC()

	var state oauth2UserLink
	_, exists, ret := checkoutDatum(key, &state, false)
	if ret != robot.Ok {
		return "", ret
	}
	if !exists {
		return "", robot.OAuth2UserNotLinked
	}
	if state.Status.ReauthRequired {
		return "", robot.OAuth2ReauthRequired
	}
	if oauth2TokenUsable(&state, now) {
		return state.Token.AccessToken, robot.Ok
	}

	lockToken, exists, ret := checkoutDatum(key, &state, true)
	if ret != robot.Ok {
		return "", ret
	}
	if !exists {
		checkinDatum(key, lockToken)
		return "", robot.OAuth2UserNotLinked
	}
	if state.Status.ReauthRequired {
		checkinDatum(key, lockToken)
		return "", robot.OAuth2ReauthRequired
	}
	if oauth2TokenUsable(&state, now) {
		checkinDatum(key, lockToken)
		return state.Token.AccessToken, robot.Ok
	}

	ret = oauth2RefreshUserToken(cfg, &state, now)
	updateRet := updateDatum(key, lockToken, state)
	if updateRet != robot.Ok {
		Log(robot.Error, "oauth2: failed updating token state for provider=%s user=%s: %s", cfg.Key, user, updateRet)
		if ret == robot.Ok {
			ret = updateRet
		}
	}
	if ret != robot.Ok {
		Log(robot.Warn, "oauth2: token retrieval failed for provider=%s user=%s: %s (%s)", cfg.Key, user, ret, state.Status.LastError)
		return "", ret
	}
	return state.Token.AccessToken, robot.Ok
}

func (r Robot) LinkOAuth2User(link *robot.OAuth2LinkRequest) robot.RetVal {
	if link == nil {
		return robot.OAuth2InvalidLinkRequest
	}
	cfg, ok := getOAuth2ProviderConfig(link.Provider)
	if !ok {
		return robot.OAuth2ProviderNotFound
	}
	if ret := oauth2ProviderConfigValid(cfg); ret != robot.Ok {
		return ret
	}
	user := strings.ToLower(strings.TrimSpace(link.User))
	if user == "" || strings.TrimSpace(link.AccessToken) == "" {
		return robot.OAuth2InvalidLinkRequest
	}
	now := time.Now().UTC()
	expiresAt, err := oauth2ResolveTime(link.ExpiresAt, link.ExpiresIn, now)
	if err != nil {
		return robot.OAuth2InvalidLinkRequest
	}
	refreshExpiresAt, err := oauth2ResolveTime(link.RefreshExpiresAt, link.RefreshExpiresIn, now)
	if err != nil {
		return robot.OAuth2InvalidLinkRequest
	}
	state := oauth2UserLink{
		SchemaVersion: oauth2DatumSchemaVersion,
		ProviderKey:   cfg.Key,
		Username:      user,
		Subject: oauth2Subject{
			ID:    strings.TrimSpace(link.SubjectID),
			Login: strings.TrimSpace(link.SubjectLogin),
			Name:  strings.TrimSpace(link.SubjectName),
			Email: strings.TrimSpace(link.SubjectEmail),
		},
		Token: oauth2TokenState{
			AccessToken:      strings.TrimSpace(link.AccessToken),
			RefreshToken:     strings.TrimSpace(link.RefreshToken),
			TokenType:        strings.TrimSpace(link.TokenType),
			Scope:            oauth2NormalizeScopes(link.Scope),
			ExpiresAt:        expiresAt,
			RefreshExpiresAt: refreshExpiresAt,
		},
		Grant: oauth2GrantState{
			Type:     strings.TrimSpace(link.GrantType),
			LinkedAt: now,
		},
	}
	if state.Token.TokenType == "" {
		state.Token.TokenType = "Bearer"
	}
	key := oauth2UserDatumKey(cfg.Key, user)
	lockToken, _, ret := checkoutDatum(key, &oauth2UserLink{}, true)
	if ret != robot.Ok {
		return ret
	}
	return updateDatum(key, lockToken, state)
}

func (r Robot) UnlinkOAuth2User(provider, user string) robot.RetVal {
	cfg, ok := getOAuth2ProviderConfig(provider)
	if !ok {
		return robot.OAuth2ProviderNotFound
	}
	user = strings.ToLower(strings.TrimSpace(user))
	if user == "" {
		return robot.OAuth2InvalidLinkRequest
	}
	return deleteDatum(oauth2UserDatumKey(cfg.Key, user))
}
