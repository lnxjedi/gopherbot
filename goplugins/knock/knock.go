package knock

import (
	"fmt"
	"strings"
	"sync"

	"github.com/parsley42/gopherbot/bot"
)

type Joke struct {
	First  string
	Second string
}

var Jokes []Joke
var lock sync.Mutex
var configured bool

var openings = []string{
	"Ok, I know a good one!",
	"Hrm... ok, this is one of my favorites...",
	"I'll see if I can think of one...",
	"Another robot told me this one, tell me if you think it's funny",
	"I found lame joke on the Internet - but it's kinda funny when a robot tells it",
	"I'll ask Watson(tm) if he knows any good ones and get back to you in a jiffy...",
	"Hang on while I Google that for you (just kidding ;-)",
	"Sure - Siri told me this one, but I think it's kind of dumb",
	"Ok, here's a funny one I found in Hillary's email...",
	"Yeah! I LOVE telling jokes!",
	"Alright - I'll see if I can make my voice sound funny",
}

var phooey = []string{
	"Ah, you're no fun",
	"What, don't you like a good knock-knock joke?",
	"Ok, maybe another time",
}

func knock(r bot.Robot, command string, args ...string) {
	switch command {
	case "init":
		lock.Lock()
		defer lock.Unlock()
		err := r.GetPluginConfig(&Jokes)
		if err == nil {
			r.Log(bot.Info, fmt.Sprintf("Knock-knock plugin successfully loaded %d jokes.", len(Jokes)))
			configured = true
		} else {
			configured = false
			r.Log(bot.Error, fmt.Errorf("Loading jokes: %v", err))
		}
	case "knock":
		if !configured {
			r.Reply("Sorry, I don't know any jokes :-(")
			return
		}
		//
		lock.Lock()
		j := Jokes[r.RandomInt(len(Jokes))]
		lock.Unlock()
		r.Pause(0.5)
		r.Say(r.RandomString(openings))
		r.Pause(1.2)
		r.Reply("Knock knock")
		for i := 0; i < 2; i++ {
			matched, timedOut, _, err := r.WaitForReply("whosthere", 14)
			fmt.Printf("Return from WFR - matched: %v, timedOut: %v, err: %v", matched, timedOut, err)
			if timedOut {
				r.Reply(r.RandomString(phooey))
				return
			}
			if err != nil {
				r.Reply("... wait, sorry - my joke algorithm broke!")
			}
			if !matched {
				switch i {
				case 0:
					fmt.Println("Not matched")
					r.Pause(0.5)
					r.Reply("(Uh, didn't you mean to say \"who's there?\")")
				case 1:
					r.Reply(r.RandomString(phooey))
					return
				}
			} else {
				break // matched
			}
		}
		r.Pause(0.5)
		r.Say(j.First)
		for i := 0; i < 2; i++ {
			matched, timedOut, reply, err := r.WaitForReply("who", 14)
			if timedOut {
				r.Reply(r.RandomString(phooey))
				return
			}
			if err != nil {
				r.Reply("... wait, sorry - my joke algorithm broke!")
			}
			if !matched {
				switch i {
				case 0:
					r.Pause(0.5)
					r.Reply("(Uh, didn't you mean to say \"" + strings.Title(j.First) + " who?\")")
				case 1:
					r.Reply(r.RandomString(phooey))
					return
				}
			} else {
				// Did the user reply correctly with <j.First> who?
				if strings.HasPrefix(strings.ToLower(reply), strings.ToLower(j.First)) {
					r.Say(j.Second)
					return
				} else {
					switch i {
					case 1:
						r.Pause(0.5)
						r.Reply("(Uh, didn't you mean to say \"" + strings.Title(j.First) + " who?\")")
					case 2:
						r.Pause(0.5)
						r.Reply(r.RandomString(phooey))
						return
					}
				}
			}
		}
	}
}

func init() {
	bot.RegisterPlugin("knock", knock)
}
