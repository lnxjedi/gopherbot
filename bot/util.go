package bot

import (
	"regexp"
)

func (b *Bot) updateRegexes() {
	preString := `^(`
	if b.alias != 0 {
		preString += string(b.alias) + "|"
	}
	preString += `(?:@?(?i)` + b.name + `[:,]{0,1}\s*))(.+)$`
	re, err := regexp.Compile(preString)
	if err == nil {
		b.Lock()
		b.preRegex = re
		b.Unlock()
	}
	postString := `^([^,@]+),?\s*((?i)@?` + b.name + `)([.?! ])?$`
	b.Debug("postString is", postString)
	re, err = regexp.Compile(postString)
	if err == nil {
		b.Lock()
		b.postRegex = re
		b.Unlock()
	}
}
