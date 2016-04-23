package bot

import (
	"regexp"
)

func setFormat(format string) MessageFormat {
	switch format {
	case "fixed":
		return Fixed
	default:
		return Variable
	}
}

// messageAppliesToPlugin checks the user and channel against the plugin's
// configuration to determine if the message should be evaluated. Used by
// both handleMessage and the help builtin.
// TODO: add logic for checking plugin.Users[]
func (b *robot) messageAppliesToPlugin(user, channel, message string, plugin Plugin) bool {
	ok := false
	directMsg := false
	if len(channel) == 0 {
		directMsg = true
	}
	if len(plugin.Channels) > 0 {
		if !directMsg {
			for _, pchannel := range plugin.Channels {
				if pchannel == channel {
					ok = true
				}
			}
		} else { // direct message
			if !plugin.DisallowDirect {
				ok = true
			}
		}
	} else {
		if directMsg {
			if !plugin.DisallowDirect {
				ok = true
			}
		} else {
			ok = true
		}
	}
	return ok
}

func (b *robot) updateRegexes() {
	preString := `^(`
	if b.alias != 0 {
		preString += string(b.alias) + "|"
	}
	preString += `(?:@?(?i)` + b.name + `[:,]{0,1}\s*))(.+)$`
	b.Log(Debug, "preString is", preString)
	re, err := regexp.Compile(preString)
	if err == nil {
		b.lock.Lock()
		b.preRegex = re
		b.lock.Unlock()
	}
	postString := `^([^,@]+),?\s*((?i)@?` + b.name + `)([.?! ])?$`
	b.Log(Debug, "postString is", postString)
	re, err = regexp.Compile(postString)
	if err == nil {
		b.lock.Lock()
		b.postRegex = re
		b.lock.Unlock()
	}
}
