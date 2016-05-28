package knock

import (
	"strings"

	"github.com/parsley42/gopherbot/bot"
)

type Joke struct {
	First  string
	Second string
}

type Config struct {
	Jokes    []Joke   // The actual jokes, first and second parts
	Openings []string // Stuff the robot says before starting the joke
	Phooey   []string // Ways the robot complains if the user doesn't respond correctly
}

func knock(r bot.Robot, c interface{}, command string, args ...string) {
	j := c.(*Config) // get access to a copy of the plugin's config
	switch command {
	case "init":
		// Ignore, this plugin has no start-up
	case "knock":
		if len(j.Jokes) == 0 {
			r.Reply("Sorry, I don't know any jokes :-(")
			return
		}
		//
		joke := &j.Jokes[r.RandomInt(len(j.Jokes))]
		r.Pause(0.5)
		r.Say(r.RandomString(j.Openings))
		r.Pause(1.2)
		r.Reply("Knock knock")
		for i := 0; i < 2; i++ {
			matched, timedOut, _, err := r.WaitForReply("whosthere", 14)
			if timedOut {
				r.Reply(r.RandomString(j.Phooey))
				return
			}
			if err != nil {
				r.Reply("... wait, sorry - my joke algorithm broke!")
			}
			if !matched {
				switch i {
				case 0:
					r.Pause(0.5)
					r.Reply("(Uh, didn't you mean to say \"who's there?\")")
				case 1:
					r.Pause(0.5)
					r.Reply(r.RandomString(j.Phooey))
					return
				}
			} else {
				break // matched
			}
		}
		r.Pause(0.5)
		r.Say(joke.First)
		for i := 0; i < 2; i++ {
			matched, timedOut, reply, err := r.WaitForReply("who", 14)
			if timedOut {
				r.Reply(r.RandomString(j.Phooey))
				return
			}
			if err != nil {
				r.Reply("... wait, sorry - my joke algorithm broke!")
			}
			if !matched {
				switch i {
				case 0:
					r.Pause(0.5)
					r.Reply("(Uh, didn't you mean to say \"" + joke.First + " who?\")")
				case 1:
					r.Pause(0.5)
					r.Reply(r.RandomString(j.Phooey))
					return
				}
			} else {
				// Did the user reply correctly with <j.First> who?
				if strings.HasPrefix(strings.ToLower(reply), strings.ToLower(joke.First)) {
					r.Say(joke.Second)
					return
				} else {
					switch i {
					case 1:
						r.Pause(0.5)
						r.Reply("(Uh, didn't you mean to say \"" + joke.First + " who?\")")
					case 2:
						r.Pause(0.5)
						r.Reply(r.RandomString(j.Phooey))
						return
					}
				}
			}
		}
	}
}

func init() {
	bot.RegisterPluginV1("knock", bot.PluginV1{
		Config{},
		knock,
	})
}
