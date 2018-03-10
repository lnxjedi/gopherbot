// The memes plugin is an example of how you can use the robot for fun things
// like generating Internet meme images. TODO: It could really use a re-write to
// make memes configurable instead of hard-coded.
package memes

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/lnxjedi/gopherbot/bot"
)

var (
	gobot   bot.Robot
	botName string
)

type MemeConfig struct {
	Username string
	Password string
	Memes    map[string]string
}

func memegen(r *bot.Robot, command string, args ...string) (retval bot.PlugRetVal) {
	var m *MemeConfig
	ret := r.GetPluginConfig(&m) // make m point to a valid, thread-safe MemeConfig
	if ret != bot.Ok || m.Password == "" {
		if command != "init" {
			r.Reply("I couldn't remember my password for the meme generator")
		}
	}
	switch command {
	case "init":
		// ignore
	default:
		url, err := createMeme(m, command, args[0], args[1])
		if err == nil {
			r.Say(url)
		} else {
			r.Reply("Sorry, something went wrong. Check the logs?")
			r.Log(bot.Error, fmt.Errorf("Generating a meme: %v", err))
		}
	}
	return
}

// Compose imgflip meme - thanks to Adam Georgeson for this function
func createMeme(m *MemeConfig, templateId, topText, bottomText string) (string, error) {
	values := url.Values{}
	values.Set("template_id", templateId)
	values.Set("username", m.Username)
	values.Set("password", m.Password)
	values.Set("text0", topText)
	values.Set("text1", bottomText)
	resp, err := http.PostForm("https://api.imgflip.com/caption_image", values)

	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return "", err
	}

	if !data["success"].(bool) {
		return "", errors.New(data["error_message"].(string))
	}

	url := data["data"].(map[string]interface{})["url"].(string)

	return url, nil
}

func init() {
	bot.RegisterPlugin("memes", bot.PluginHandler{
		DefaultConfig: defaultConfig,
		Handler:       memegen,
		Config:        &MemeConfig{},
	})
}
