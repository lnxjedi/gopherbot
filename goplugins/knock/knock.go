package knock

// knock implements a simple demonstrator plugin for using Gopherbot's
// WaitForReply function to tell knock-knock jokes.

import (
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
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

var knockhandler = robot.PluginHandler{
	Handler: knock,
	Config:  &JokeConfig{},
}

func knock(r robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	var j *JokeConfig // get access to a copy of the plugin's config
	switch command {
	case "init":
		// Ignore, this plugin has no start-up
	case "knock":
		if ret := r.GetTaskConfig(&j); ret != robot.Ok {
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
			if ret == robot.Interrupted {
				r.Reply("Ok, I guess you don't like knock-knock jokes!")
				return
			}
			if ret == robot.TimeoutExpired {
				r.Reply(r.RandomString(j.Phooey))
				return
			}
			if ret == robot.ReplyNotMatched {
				switch i {
				case 0:
					r.Pause(0.5)
					r.Reply("(Uh, didn't you mean to say \"who's there?\")")
				case 1:
					r.Pause(0.5)
					r.Reply(r.RandomString(j.Phooey))
					return
				}
			} else if ret == robot.UseDefaultValue {
				r.Reply("Sheesh, are you kidding me? Ok, I'll assume you meant 'Who's there?'...")
				r.Pause(1)
				break
			} else if ret != robot.Ok {
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
			if ret == robot.Interrupted {
				r.Reply("Oooo, you're going to leave the joke unfinished? What about CLOSURE?!?")
				return
			}
			if ret == robot.TimeoutExpired {
				r.Reply(r.RandomString(j.Phooey))
				return
			}
			if ret == robot.UseDefaultValue {
				switch i {
				case 0:
					r.Reply("Ohhhhh no... you're going to have to spell it out, lazy bones!")
				case 1:
					r.Reply(r.RandomString(j.Phooey))
					return
				}
			} else if ret == robot.ReplyNotMatched {
				switch i {
				case 0:
					r.Pause(0.5)
					r.Reply("(Uh, didn't you mean to say \"" + joke.First + " who?\")")
				case 1:
					r.Pause(0.5)
					r.Reply(r.RandomString(j.Phooey))
					return
				}
			} else if ret == robot.Ok {
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
