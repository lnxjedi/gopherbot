package dynamobrain

import "github.com/lnxjedi/gopherbot/v2/bot"

func init() {
	bot.RegisterPreload("brains/dynamodb.so")
	bot.RegisterSimpleBrain("dynamo", provider)
}
