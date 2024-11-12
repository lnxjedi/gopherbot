package bot

import (
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
)

// see robot/robot.go
func (r Robot) GetMessage() *robot.Message {
	return r.Message
}

// see robot/robot.go
func (r Robot) GetUserAttribute(u, a string) *robot.AttrRet {
	a = strings.ToLower(a)
	var user string
	var ui *UserInfo
	var ok bool
	if ui, ok = r.maps.user[u]; ok {
		user = "<" + ui.UserID + ">"
	} else {
		user = u
	}
	if ui != nil {
		var attr string
		switch a {
		case "name", "username", "handle", "user":
			attr = ui.UserName
		case "id", "internalid", "protocolid":
			attr = ui.UserID
		case "mail", "email":
			attr = ui.Email
		case "fullname", "realname":
			attr = ui.FullName
		case "firstname", "givenname":
			attr = ui.FirstName
		case "lastname", "surname":
			attr = ui.LastName
		case "phone":
			attr = ui.Phone
		}
		if len(attr) > 0 {
			return &robot.AttrRet{attr, robot.Ok}
		}
	}
	attr, ret := interfaces.GetProtocolUserAttribute(user, a)
	return &robot.AttrRet{attr, ret}
}

// see robot/robot.go
func (r Robot) GetSenderAttribute(a string) *robot.AttrRet {
	a = strings.ToLower(a)
	var ui *UserInfo
	ui, _ = r.maps.user[r.User]
	switch a {
	case "name", "username", "handle", "user":
		return &robot.AttrRet{r.User, robot.Ok}
	case "id", "internalid", "protocolid":
		return &robot.AttrRet{r.ProtocolUser, robot.Ok}
	}
	if ui != nil {
		var attr string
		switch a {
		case "mail", "email":
			attr = ui.Email
		case "fullname", "realname":
			attr = ui.FullName
		case "firstname", "givenname":
			attr = ui.FirstName
		case "lastname", "surname":
			attr = ui.LastName
		case "phone":
			attr = ui.Phone
		}
		if len(attr) > 0 {
			return &robot.AttrRet{attr, robot.Ok}
		}
	}
	user := r.ProtocolUser
	if len(user) == 0 {
		user = r.User
	}
	attr, ret := interfaces.GetProtocolUserAttribute(user, a)
	return &robot.AttrRet{attr, ret}
}
