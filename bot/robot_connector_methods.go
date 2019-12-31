package bot

import (
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
)

// GetMessage returns a pointer to the message struct
func (r Robot) GetMessage() *robot.Message {
	return r.Message
}

// GetUserAttribute returns a AttrRet with
// - The string Attribute of a user, or "" if unknown/error
// - A RetVal which is one of Ok, UserNotFound, AttributeNotFound
// Current attributes:
// name(handle), fullName, email, firstName, lastName, phone, internalID
// TODO: supplement data with gopherbot.yaml user's table, if an
// admin wants to supplment whats available from the protocol.
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

// GetSenderAttribute returns a AttrRet with
// - The string Attribute of the sender, or "" if unknown/error
// - A RetVal which is one of Ok, UserNotFound, AttributeNotFound
// Current attributes:
// name(handle), fullName, email, firstName, lastName, phone, internalID
// TODO: (see above)
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
