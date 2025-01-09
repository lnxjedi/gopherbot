package cloudflarekv

import "github.com/lnxjedi/gopherbot/v2/bot"

func init() {
	bot.RegisterSimpleBrain("cloudflare", provider)
}
