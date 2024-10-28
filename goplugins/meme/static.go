package meme

import "github.com/lnxjedi/gopherbot/robot"

func init() {
	robot.RegisterPlugin("memes", memehandler)
}
