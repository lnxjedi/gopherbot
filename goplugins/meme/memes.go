package meme

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/lnxjedi/gopherbot/bot"
	"github.com/lnxjedi/gopherbot/robot"
)

var (
	gobot   bot.Robot
	botName string
)

type MemeConfig struct {
	Username string
	Password string
}

func memegen(r robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	var m *MemeConfig
	r.GetTaskConfig(&m) // make m point to a valid, thread-safe MemeConfig
	if len(m.Password) == 0 {
		m.Password = r.GetSecret("PASSWORD")
	}
	if len(m.Username) == 0 || len(m.Password) == 0 {
		if command != "init" {
			r.Reply("I couldn't remember my username or password for the meme generator")
		}
	}
	switch command {
	case "init":
		// ignore
	default:
		var top, bottom string
		if len(args[1]) > 0 {
			top = args[0]
			bottom = args[1]
		} else {
			top = ""
			bottom = args[0]
		}
		url, err := createMeme(m, command, top, bottom)
		if err == nil {
			r.Say(url)
		} else {
			r.Reply("Sorry, something went wrong. Check the logs?")
			r.Log(robot.Error, "Generating a meme: %v", err)
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

var memehandler = robot.PluginHandler{
	Handler: memegen,
	Config:  &MemeConfig{},
}
