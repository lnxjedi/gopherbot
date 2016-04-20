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

func (b *Bot) updateRegexes() {
	preString := `^(`
	if b.alias != 0 {
		preString += string(b.alias) + "|"
	}
	preString += `(?:@?(?i)` + b.name + `[:,]{0,1}\s*))(.+)$`
	b.Log(Debug, "preString is", preString)
	re, err := regexp.Compile(preString)
	if err == nil {
		b.Lock()
		b.preRegex = re
		b.Unlock()
	}
	postString := `^([^,@]+),?\s*((?i)@?` + b.name + `)([.?! ])?$`
	b.Log(Debug, "postString is", postString)
	re, err = regexp.Compile(postString)
	if err == nil {
		b.Lock()
		b.postRegex = re
		b.Unlock()
	}
}
