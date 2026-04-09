package util

import "regexp"

var idRegex = regexp.MustCompile(`^<(.*)>$`)

// ExtractID checks a user/channel string against the pattern '<internalID>'
// and if it matches returns the internalID,true; otherwise returns the
// unmodified string,false.
func ExtractID(u string) (string, bool) {
	matches := idRegex.FindStringSubmatch(u)
	if len(matches) > 0 {
		return matches[1], true
	}
	return u, false
}
