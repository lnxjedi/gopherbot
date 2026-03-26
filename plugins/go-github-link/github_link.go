package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

var defaultConfig = []byte(`
Commands:
- Regex: '(?i:link[- ]github)'
  Command: link
  Usage: "link-github"
  Summary: "Link your GitHub account to the robot with a device authorization code."
  Examples:
  - "(alias) link-github"
  - "(bot) link-github"
- Regex: '(?i:unlink[- ]github)'
  Command: unlink
  Usage: "unlink-github"
  Summary: "Remove your linked GitHub account from the robot."
  Examples:
  - "(alias) unlink-github"
- Regex: '(?i:github[- ]whoami)'
  Command: whoami
  Usage: "github-whoami"
  Summary: "Verify the GitHub account currently linked to your robot user."
  Examples:
  - "(alias) github-whoami"
AllowDirect: true
`)

const (
	githubProviderKey      = "github"
	githubOAuthHTTPTimeout = 20 * time.Second
	githubDeviceAuthURL    = "https://github.com/login/device/code"
	githubTokenURL         = "https://github.com/login/oauth/access_token"
	githubUserInfoURL      = "https://api.github.com/user"
)

var githubDefaultScopes = []string{"read:user", "repo", "workflow"}

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

type oauth2DeviceAuthorizationResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
	Error                   string `json:"error"`
	ErrorDescription        string `json:"error_description"`
	Message                 string `json:"message"`
}

type githubUserInfo struct {
	ID    interface{} `json:"id"`
	Login string      `json:"login"`
	Name  string      `json:"name"`
	Email string      `json:"email"`
}

type githubCredentialResult struct {
	ClientID     string
	ClientSecret string
	Ret          robot.RetVal
}

type githubDeviceAuthorizationResult struct {
	Response *oauth2DeviceAuthorizationResponse
	Ret      robot.RetVal
}

type githubTokenExchangeResult struct {
	Response *oauth2TokenHTTPResponse
	Ret      robot.RetVal
}

func Configure() *[]byte {
	return &defaultConfig
}

func githubLinkRetMessage(ret robot.RetVal) string {
	switch ret {
	case robot.OAuth2ProviderNotFound:
		return "GitHub OAuth is not configured for this robot yet."
	case robot.OAuth2UserNotLinked:
		return "I don't have a linked GitHub account for you yet."
	case robot.OAuth2ReauthRequired:
		return "Your linked GitHub account needs to be linked again."
	case robot.OAuth2RefreshFailed:
		return "I couldn't refresh your GitHub token right now."
	case robot.OAuth2ConfigError:
		return "The robot's GitHub OAuth provider configuration is incomplete."
	default:
		return "I ran into a GitHub OAuth error."
	}
}

func githubClientCredentials(r robot.Robot) githubCredentialResult {
	result := githubCredentialResult{
		ClientID:     strings.TrimSpace(r.GetParameter("CLIENT_ID")),
		ClientSecret: strings.TrimSpace(r.GetParameter("CLIENT_SECRET")),
		Ret:          robot.Ok,
	}
	if result.ClientID == "" || result.ClientSecret == "" {
		result.Ret = robot.OAuth2ConfigError
	}
	return result
}

func oauth2DoFormPost(endpointURL string, headers map[string]string, values url.Values) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodPost, endpointURL, bytes.NewBufferString(values.Encode()))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for key, value := range headers {
		if key != "" && value != "" {
			req.Header.Set(key, value)
		}
	}
	client := &http.Client{Timeout: githubOAuthHTTPTimeout}
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

func oauth2NormalizeScopes(raw interface{}) []string {
	switch v := raw.(type) {
	case nil:
		return nil
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		fields := strings.FieldsFunc(v, func(r rune) bool { return r == ' ' || r == ',' })
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
			s, ok := item.(string)
			if !ok {
				continue
			}
			s = strings.TrimSpace(s)
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func oauth2BearerHeader(token string) string {
	return "Bearer " + strings.TrimSpace(token)
}

func requestOAuth2DeviceAuthorization(clientID string, scopes []string) githubDeviceAuthorizationResult {
	if strings.TrimSpace(clientID) == "" {
		return githubDeviceAuthorizationResult{Ret: robot.OAuth2ConfigError}
	}
	if len(scopes) == 0 {
		scopes = append([]string(nil), githubDefaultScopes...)
	}
	values := url.Values{}
	values.Set("client_id", clientID)
	if len(scopes) > 0 {
		values.Set("scope", strings.Join(scopes, " "))
	}
	payload, _, err := oauth2DoFormPost(githubDeviceAuthURL, map[string]string{
		"Accept": "application/json",
	}, values)
	if err != nil {
		return githubDeviceAuthorizationResult{Ret: robot.Failed}
	}
	var resp oauth2DeviceAuthorizationResponse
	if err := oauth2DecodeJSON(payload, &resp); err != nil {
		return githubDeviceAuthorizationResult{Ret: robot.Failed}
	}
	if resp.Error != "" || resp.DeviceCode == "" {
		return githubDeviceAuthorizationResult{Ret: robot.Failed}
	}
	if resp.Interval <= 0 {
		resp.Interval = 5
	}
	return githubDeviceAuthorizationResult{Response: &resp, Ret: robot.Ok}
}

func exchangeOAuth2DeviceCode(clientID, clientSecret, deviceCode string) githubTokenExchangeResult {
	if strings.TrimSpace(clientID) == "" || strings.TrimSpace(clientSecret) == "" {
		return githubTokenExchangeResult{Ret: robot.OAuth2ConfigError}
	}
	if strings.TrimSpace(deviceCode) == "" {
		return githubTokenExchangeResult{Ret: robot.OAuth2ConfigError}
	}
	values := url.Values{}
	values.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	values.Set("device_code", deviceCode)
	values.Set("client_id", clientID)
	values.Set("client_secret", clientSecret)
	payload, _, err := oauth2DoFormPost(githubTokenURL, map[string]string{
		"Accept": "application/json",
	}, values)
	if err != nil {
		return githubTokenExchangeResult{Ret: robot.Failed}
	}
	var resp oauth2TokenHTTPResponse
	if err := oauth2DecodeJSON(payload, &resp); err != nil {
		return githubTokenExchangeResult{Ret: robot.Failed}
	}
	return githubTokenExchangeResult{Response: &resp, Ret: robot.Ok}
}

func githubFetchUserInfo(token string) (*githubUserInfo, error) {
	req, err := http.NewRequest(http.MethodGet, githubUserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", oauth2BearerHeader(token))
	client := &http.Client{Timeout: githubOAuthHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("github user info request failed with status %d", resp.StatusCode)
	}
	var info githubUserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

func githubSubjectID(raw interface{}) string {
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return fmt.Sprintf("%.0f", v)
	default:
		return ""
	}
}

func PluginHandler(r robot.Robot, command string, args ...string) robot.TaskRetVal {
	if command == "init" {
		return robot.Normal
	}
	msg := r.GetMessage()
	if msg == nil {
		r.Say("I couldn't determine which robot user is asking.")
		return robot.Fail
	}
	user := strings.ToLower(strings.TrimSpace(msg.User))
	if user == "" {
		r.Say("I couldn't determine which robot user is asking.")
		return robot.Fail
	}

	switch command {
	case "whoami":
		token, ret := r.GetOAuth2Token(githubProviderKey, user)
		if ret != robot.Ok {
			r.Say(githubLinkRetMessage(ret))
			return robot.Fail
		}
		info, err := githubFetchUserInfo(token)
		if err != nil {
			r.Log(robot.Error, "github-link: whoami user info lookup failed for %s: %v", user, err)
			r.Say("I got a GitHub token, but I couldn't verify your GitHub profile right now.")
			return robot.Fail
		}
		reply := fmt.Sprintf("Your linked GitHub account is '%s'", info.Login)
		if info.Name != "" {
			reply += fmt.Sprintf(", name '%s'", info.Name)
		}
		if info.Email != "" {
			reply += fmt.Sprintf(", email '%s'", info.Email)
		}
		r.Say(reply)
		return robot.Normal
	case "unlink":
		ret := r.UnlinkOAuth2User(githubProviderKey, user)
		if ret != robot.Ok {
			r.Log(robot.Error, "github-link: unlink failed for %s: %s", user, ret)
			r.Say("I couldn't unlink your GitHub account right now.")
			return robot.Fail
		}
		r.Say("I removed any stored GitHub link for your robot user.")
		return robot.Normal
	case "link":
		creds := githubClientCredentials(r)
		if creds.Ret != robot.Ok {
			r.Say("The GitHub link plugin needs a ParameterSet providing CLIENT_ID and CLIENT_SECRET.")
			return robot.Fail
		}
		token, ret := r.GetOAuth2Token(githubProviderKey, user)
		switch ret {
		case robot.Ok:
			if strings.TrimSpace(token) != "" {
				r.Say("You already have a linked GitHub account. Use 'unlink-github' first if you want to replace it.")
				return robot.Normal
			}
		case robot.OAuth2UserNotLinked, robot.OAuth2ReauthRequired:
			// Continue into the link flow.
		default:
			r.Say(githubLinkRetMessage(ret))
			return robot.Fail
		}
		device := requestOAuth2DeviceAuthorization(creds.ClientID, nil)
		if device.Ret != robot.Ok || device.Response == nil {
			r.Log(robot.Error, "github-link: device authorization start failed for %s: %s", user, device.Ret)
			r.Say(githubLinkRetMessage(device.Ret))
			return robot.Fail
		}
		directBot := r.Direct()
		if strings.TrimSpace(msg.Channel) != "" {
			r.Say("I'll message you directly to finish linking your GitHub account.")
		}
		linkText := fmt.Sprintf("To link GitHub, visit %s and enter code `%s`.", device.Response.VerificationURI, device.Response.UserCode)
		if device.Response.VerificationURIComplete != "" {
			linkText += fmt.Sprintf("\n\nShortcut: %s", device.Response.VerificationURIComplete)
		}
		directBot.Say(linkText)

		interval := time.Duration(device.Response.Interval) * time.Second
		if interval <= 0 {
			interval = 5 * time.Second
		}
		timeout := time.Duration(device.Response.ExpiresIn) * time.Second
		if timeout <= 0 {
			timeout = 5 * time.Minute
		}
		deadline := time.Now().Add(timeout)
		for time.Now().Before(deadline) {
			tokenResp := exchangeOAuth2DeviceCode(creds.ClientID, creds.ClientSecret, device.Response.DeviceCode)
			if tokenResp.Ret != robot.Ok || tokenResp.Response == nil {
				directBot.Say("There was a problem talking to GitHub while waiting for authorization.")
				return robot.Fail
			}
			if tokenResp.Response.Error != "" {
				switch strings.ToLower(strings.TrimSpace(tokenResp.Response.Error)) {
				case "authorization_pending":
					r.Pause(interval.Seconds())
					continue
				case "slow_down":
					if tokenResp.Response.Interval > 0 {
						interval = time.Duration(tokenResp.Response.Interval) * time.Second
					} else {
						interval += 5 * time.Second
					}
					r.Pause(interval.Seconds())
					continue
				case "expired_token":
					directBot.Say("The GitHub device code expired before authorization completed. Try 'link-github' again.")
					return robot.Fail
				case "access_denied":
					directBot.Say("GitHub authorization was denied.")
					return robot.Fail
				default:
					r.Log(robot.Error, "github-link: unexpected device token error for %s: %s", user, oauth2ProviderErrorMessage(*tokenResp.Response))
					directBot.Say("GitHub returned an unexpected authorization error.")
					return robot.Fail
				}
			}
			if strings.TrimSpace(tokenResp.Response.AccessToken) == "" {
				directBot.Say("GitHub returned an empty access token.")
				return robot.Fail
			}
			info, err := githubFetchUserInfo(tokenResp.Response.AccessToken)
			if err != nil {
				r.Log(robot.Error, "github-link: user info lookup failed for %s: %v", user, err)
				directBot.Say("GitHub authorization succeeded, but I couldn't verify the linked account profile.")
				return robot.Fail
			}
			ret = r.LinkOAuth2User(&robot.OAuth2LinkRequest{
				Provider:         githubProviderKey,
				User:             user,
				AccessToken:      tokenResp.Response.AccessToken,
				RefreshToken:     tokenResp.Response.RefreshToken,
				TokenType:        tokenResp.Response.TokenType,
				Scope:            oauth2NormalizeScopes(tokenResp.Response.Scope),
				ExpiresIn:        tokenResp.Response.ExpiresIn,
				RefreshExpiresIn: tokenResp.Response.RefreshTokenExpiresIn,
				GrantType:        "device_authorization",
				SubjectID:        githubSubjectID(info.ID),
				SubjectLogin:     info.Login,
				SubjectName:      info.Name,
				SubjectEmail:     info.Email,
			})
			if ret != robot.Ok {
				r.Log(robot.Error, "github-link: failed storing oauth link for %s: %s", user, ret)
				directBot.Say("GitHub authorization succeeded, but I couldn't store the linked account.")
				return robot.Fail
			}
			directBot.Say(fmt.Sprintf("GitHub linked successfully as '%s'.", info.Login))
			if strings.TrimSpace(msg.Channel) != "" {
				r.Say("Your GitHub account is linked.")
			}
			return robot.Normal
		}
		directBot.Say("Timed out waiting for GitHub authorization to complete.")
		return robot.Fail
	default:
		return robot.Fail
	}
}
