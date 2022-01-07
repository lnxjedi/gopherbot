package duo

import "github.com/lnxjedi/gopherbot/v2/bot"

func init() {
	bot.RegisterPlugin("duo", duohandler)
}
