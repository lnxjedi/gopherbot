package bot

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

type githubUserInfo struct {
	ID    interface{} `json:"id"`
	Login string      `json:"login"`
	Name  string      `json:"name"`
	Email string      `json:"email"`
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

func githubFetchUserInfo(provider OAuth2ProviderConfig, token string) (*githubUserInfo, error) {
	if strings.TrimSpace(provider.UserInfo.URL) == "" {
		return nil, fmt.Errorf("provider user info URL is not configured")
	}
	req, err := http.NewRequest(http.MethodGet, provider.UserInfo.URL, nil)
	if err != nil {
		return nil, err
	}
	for key, value := range provider.UserInfo.Headers {
		if key != "" && value != "" {
			req.Header.Set(key, value)
		}
	}
	req.Header.Set("Authorization", oauth2BearerHeader(token))
	client := oauth2HTTPClientFactory()
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

func githubLinkHasStoredUser(provider OAuth2ProviderConfig, user string) (bool, robot.RetVal) {
	key := oauth2UserDatumKey(provider.Key, user)
	var state oauth2UserLink
	_, exists, ret := checkoutDatum(key, &state, false)
	return exists, ret
}

func githubLinkPlugin(r robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	if command == "init" {
		return robot.Normal
	}
	provider, ok := getOAuth2ProviderConfig("github")
	if !ok {
		r.Say(githubLinkRetMessage(robot.OAuth2ProviderNotFound))
		return robot.Fail
	}
	if ret := oauth2ProviderConfigValid(provider); ret != robot.Ok {
		r.Say(githubLinkRetMessage(ret))
		return robot.Fail
	}
	user := strings.ToLower(strings.TrimSpace(r.GetMessage().User))
	if user == "" {
		r.Say("I couldn't determine which robot user is asking.")
		return robot.Fail
	}

	switch command {
	case "whoami":
		token, ret := r.GetOAuth2Token(provider.Key, user)
		if ret != robot.Ok {
			r.Say(githubLinkRetMessage(ret))
			return robot.Fail
		}
		info, err := githubFetchUserInfo(provider, token)
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
		exists, ret := githubLinkHasStoredUser(provider, user)
		if ret != robot.Ok {
			r.Log(robot.Error, "github-link: unlink check failed for %s: %s", user, ret)
			r.Say("I had trouble checking your stored GitHub link.")
			return robot.Fail
		}
		if !exists {
			r.Say("I don't have a linked GitHub account for you.")
			return robot.Fail
		}
		ret = r.UnlinkOAuth2User(provider.Key, user)
		if ret != robot.Ok {
			r.Log(robot.Error, "github-link: unlink failed for %s: %s", user, ret)
			r.Say("I couldn't unlink your GitHub account right now.")
			return robot.Fail
		}
		r.Say("I removed your linked GitHub account.")
		return robot.Normal
	case "link":
		if token, ret := r.GetOAuth2Token(provider.Key, user); ret == robot.Ok && strings.TrimSpace(token) != "" {
			r.Say("You already have a linked GitHub account. Use 'unlink-github' first if you want to replace it.")
			return robot.Normal
		}
		device, ret := requestOAuth2DeviceAuthorization(provider, nil)
		if ret != robot.Ok {
			r.Log(robot.Error, "github-link: device authorization start failed for %s: %s", user, ret)
			r.Say(githubLinkRetMessage(ret))
			return robot.Fail
		}

		directBot := r.Direct()
		if strings.TrimSpace(r.GetMessage().Channel) != "" {
			r.Say("I'll message you directly to finish linking your GitHub account.")
		}
		linkText := fmt.Sprintf("To link GitHub, visit %s and enter code `%s`.", device.VerificationURI, device.UserCode)
		if device.VerificationURIComplete != "" {
			linkText += fmt.Sprintf("\n\nShortcut: %s", device.VerificationURIComplete)
		}
		directBot.Say(linkText)

		interval := time.Duration(device.Interval) * time.Second
		if interval <= 0 {
			interval = 5 * time.Second
		}
		timeout := time.Duration(device.ExpiresIn) * time.Second
		if timeout <= 0 {
			timeout = 5 * time.Minute
		}
		deadline := time.Now().Add(timeout)

		for time.Now().Before(deadline) {
			tokenResp, ret := exchangeOAuth2DeviceCode(provider, device.DeviceCode)
			if ret != robot.Ok {
				directBot.Say("There was a problem talking to GitHub while waiting for authorization.")
				return robot.Fail
			}
			if tokenResp.Error != "" {
				switch strings.ToLower(strings.TrimSpace(tokenResp.Error)) {
				case "authorization_pending":
					r.Pause(interval.Seconds())
					continue
				case "slow_down":
					if tokenResp.Interval > 0 {
						interval = time.Duration(tokenResp.Interval) * time.Second
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
					r.Log(robot.Error, "github-link: unexpected device token error for %s: %s", user, oauth2ProviderErrorMessage(*tokenResp))
					directBot.Say("GitHub returned an unexpected authorization error.")
					return robot.Fail
				}
			}
			if strings.TrimSpace(tokenResp.AccessToken) == "" {
				directBot.Say("GitHub returned an empty access token.")
				return robot.Fail
			}
			info, err := githubFetchUserInfo(provider, tokenResp.AccessToken)
			if err != nil {
				r.Log(robot.Error, "github-link: user info lookup failed for %s: %v", user, err)
				directBot.Say("GitHub authorization succeeded, but I couldn't verify the linked account profile.")
				return robot.Fail
			}
			ret = r.LinkOAuth2User(&robot.OAuth2LinkRequest{
				Provider:         provider.Key,
				User:             user,
				AccessToken:      tokenResp.AccessToken,
				RefreshToken:     tokenResp.RefreshToken,
				TokenType:        tokenResp.TokenType,
				Scope:            oauth2NormalizeScopes(tokenResp.Scope),
				ExpiresIn:        tokenResp.ExpiresIn,
				RefreshExpiresIn: tokenResp.RefreshTokenExpiresIn,
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
			if strings.TrimSpace(r.GetMessage().Channel) != "" {
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

func init() {
	robot.RegisterPlugin("github-link", robot.PluginHandler{Handler: githubLinkPlugin})
}
