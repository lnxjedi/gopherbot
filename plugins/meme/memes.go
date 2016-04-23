package memes

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/parsley42/gopherbot/bot"
)

var (
	gobot   bot.Robot
	botName string
)

type MemeConfig struct {
	Username string
	Password string
}

var config MemeConfig
var configured bool = false

func memegen(r bot.Robot, channel, user, command string, args ...string) {
	switch command {
	case "start":
		gobot = r
		botName = user
		err := r.GetPluginConfig(&config)
		if err == nil {
			configured = true
		}
	case "simply":
		sendMeme(r, "61579", "ONE DOES NOT SIMPLY", args[0])
	case "prepare":
		sendMeme(r, "47779539", "You "+args[0], "PREPARE TO DIE")
	}
}

func sendMeme(r bot.Robot, templateId, topText, bottomText string) {
	url, err := createMeme(templateId, topText, bottomText)
	if err == nil {
		r.Say(url)
	} else {
		r.Log(bot.Error, fmt.Errorf("Generating a meme: %v", err))
	}
}

// Compose imgflip meme - thanks to Adam Georgeson for this function
func createMeme(templateId, topText, bottomText string) (string, error) {
	values := url.Values{}
	values.Set("template_id", templateId)
	values.Set("username", config.Username)
	values.Set("password", config.Password)
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
	bot.RegisterPlugin("memes", memegen)
}
