package firestorebrain

import "github.com/lnxjedi/gopherbot/v2/bot"

func init() {
	bot.RegisterSimpleBrain("gcpfirestore", provider)
}
