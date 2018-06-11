// Package knock implements a simple demonstrator plugin for using Gopherbot's
// WaitForReply function to tell knock-knock jokes.
package knock

import (
	"strings"

	"github.com/lnxjedi/gopherbot/bot"
)

// Joke holds a knock-knock joke
type Joke struct {
	First  string
	Second string
}

// JokeConfig holds the config for all the jokes the robot knows
type JokeConfig struct {
	Jokes    []Joke   // The actual jokes, first and second parts
	Openings []string // Stuff the robot says before starting the joke
	Phooey   []string // Ways the robot complains if the user doesn't respond correctly
}

func knock(r *bot.Robot, command string, args ...string) (retval bot.TaskRetVal) {
	var j *JokeConfig // get access to a copy of the plugin's config
	switch command {
	case "init":
		// Ignore, this plugin has no start-up
	case "knock":
		if ret := r.GetTaskConfig(&j); ret != bot.Ok {
			r.Reply("Sorry, I couldn't find my joke book")
		}
		if len(j.Jokes) == 0 {
			r.Reply("Sorry, I don't know any jokes :-(")
			return
		}
		//
		joke := &j.Jokes[r.RandomInt(len(j.Jokes))]
		r.Pause(0.5)
		r.Say(r.RandomString(j.Openings))
		r.Pause(1.2)
		for i := 0; i < 2; i++ {
			_, ret := r.PromptForReply("whosthere", "Knock knock")
			if ret == bot.Interrupted {
				r.Reply("Ok, I guess you don't like knock-knock jokes!")
				return
			}
			if ret == bot.TimeoutExpired {
				r.Reply(r.RandomString(j.Phooey))
				return
			}
			if ret == bot.ReplyNotMatched {
				switch i {
				case 0:
					r.Pause(0.5)
					r.Reply("(Uh, didn't you mean to say \"who's there?\")")
				case 1:
					r.Pause(0.5)
					r.Reply(r.RandomString(j.Phooey))
					return
				}
			} else if ret == bot.UseDefaultValue {
				r.Reply("Sheesh, are you kidding me? Ok, I'll assume you meant 'Who's there?'...")
				r.Pause(1)
				break
			} else if ret != bot.Ok {
				r.Reply("Sorry, something broke")
				return
			} else {
				break // matched
			}
		}
		r.Pause(0.5)
		if joke.First == "Interrupting Cow" {
			go func() {
				r.Pause(3.5)
				r.Reply("MOOOOOOOO!!!")
			}()
			return
		}
		for i := 0; i < 2; i++ {
			reply, ret := r.PromptForReply("who", joke.First)
			if ret == bot.Interrupted {
				r.Reply("Oooo, you're going to leave the joke unfinished? What about CLOSURE?!?")
				return
			}
			if ret == bot.TimeoutExpired {
				r.Reply(r.RandomString(j.Phooey))
				return
			}
			if ret == bot.UseDefaultValue {
				switch i {
				case 0:
					r.Reply("Ohhhhh no... you're going to have to spell it out, lazy bones!")
				case 1:
					r.Reply(r.RandomString(j.Phooey))
					return
				}
			} else if ret == bot.ReplyNotMatched {
				switch i {
				case 0:
					r.Pause(0.5)
					r.Reply("(Uh, didn't you mean to say \"" + joke.First + " who?\")")
				case 1:
					r.Pause(0.5)
					r.Reply(r.RandomString(j.Phooey))
					return
				}
			} else if ret == bot.Ok {
				// Did the user reply correctly with <j.First> who?
				if strings.HasPrefix(strings.ToLower(reply), strings.ToLower(joke.First)) {
					r.Say(joke.Second)
					return
				}
				switch i {
				case 1:
					r.Pause(0.5)
					r.Reply("(Uh, didn't you mean to say \"" + joke.First + " who?\")")
				case 2:
					r.Pause(0.5)
					r.Reply(r.RandomString(j.Phooey))
					return
				}
			} else {
				r.Reply("... wait, sorry - my joke algorithm broke!")
				return
			}
		}
	}
	return
}

func init() {
	bot.RegisterPlugin("knock", bot.PluginHandler{
		DefaultConfig: defaultConfig,
		Handler:       knock,
		Config:        &JokeConfig{},
	})
}
