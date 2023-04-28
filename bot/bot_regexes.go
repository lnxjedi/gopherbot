package bot

import (
	"regexp"
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
)

const escapeAliases = `*+^$?\[]{}`
const aliases = `&!;:-%#@~<>/`

func updateRegexes() {
	currentCfg.RLock()
	name := currentCfg.botinfo.UserName
	protoMention := currentCfg.botinfo.protoMention
	alias := currentCfg.alias
	currentCfg.RUnlock()
	preRegex, postRegex, bareRegex, errpre, errpost, errbare := updateRegexesWrapped(name, protoMention, alias)
	if errpre != nil {
		Log(robot.Error, "Compiling pre regex: %s", errpre)
	}
	if preRegex != nil {
		Log(robot.Debug, "Setting pre regex to: %s", preRegex)
	}
	if errpost != nil {
		Log(robot.Error, "Compiling post regex: %s", errpost)
	}
	if postRegex != nil {
		Log(robot.Debug, "Setting post regex to: %s", postRegex)
	}
	if errbare != nil {
		Log(robot.Error, "Compiling bare regex: %s", errbare)
	}
	if bareRegex != nil {
		Log(robot.Debug, "Setting bare regex to: %s", bareRegex)
	}
	regexes.Lock()
	regexes.preRegex = preRegex
	regexes.postRegex = postRegex
	regexes.bareRegex = bareRegex
	regexes.Unlock()
}

// TODO: write unit test. The regexes produced shouldn't be checked, but rather
// whether given strings do or don't match them. Note: this code is partially
// tested in TestBotName
func updateRegexesWrapped(name, mention string, alias rune) (preRe, postRe, bareRe *regexp.Regexp, errpre, errpost, errbare error) {
	preRe = nil
	postRe = nil
	if alias == 0 && len(name) == 0 {
		Log(robot.Error, "Robot has no name or alias, and will only respond to direct messages")
		return
	}
	names := []string{}
	barenames := []string{}
	if alias != 0 {
		if strings.ContainsRune(string(escapeAliases), alias) {
			names = append(names, `\`+string(alias))
			barenames = append(barenames, `\`+string(alias))
		} else {
			names = append(names, string(alias))
			barenames = append(barenames, string(alias))
		}
	}
	if len(name) > 0 {
		if len(mention) > 0 {
			names = append(names, `(?i:`+name+`)[:, ]`)
			barenames = append(barenames, `(?i:`+name+`\??)`)
		} else {
			names = append(names, `@?`+name+`[:, ]`)
			barenames = append(barenames, `@?`+name+`\??`)
		}
	}
	if len(mention) > 0 {
		names = append(names, `@`+mention+`[:, ]`)
		barenames = append(barenames, `@`+mention+`\??`)
	}
	preString := `^(?s)(?i:` + strings.Join(names, "|") + `\s*)(.*)$`
	preRe, errpre = regexp.Compile(preString)
	// NOTE: the preString regex matches a bare alias, but not a bare name
	if len(name) > 0 {
		postString := `^([^,@]+),\s+(?i:@?` + name + `)([.?!])?$`
		postRe, errpost = regexp.Compile(postString)
		bareString := `^@?(?i:` + strings.Join(barenames, "|") + `)$`
		bareRe, errbare = regexp.Compile(bareString)
	}
	return
}
