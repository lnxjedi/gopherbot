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
	protocol := protocolFromIncoming(r.Incoming, r.Protocol)
	user := u
	var pui *UserInfo
	var dui *DirectoryUser
	protocolMapped := false
	if pm, exists := r.maps.userProto[protocol]; exists {
		if pu, ok := pm[u]; ok {
			pui = pu
			user = "<" + pu.UserID + ">"
			protocolMapped = true
		}
	}
	if d, ok := r.maps.user[u]; ok {
		dui = d
	}
	if pui != nil || dui != nil {
		var attr string
		switch a {
		case "name", "username", "handle", "user":
			if pui != nil {
				attr = pui.UserName
			} else {
				attr = dui.UserName
			}
		case "id", "internalid", "protocolid":
			if protocolMapped {
				attr = pui.UserID
			}
		case "mail", "email":
			if pui != nil {
				attr = pui.Email
			} else {
				attr = dui.Email
			}
		case "fullname", "realname":
			if pui != nil {
				attr = pui.FullName
			} else {
				attr = dui.FullName
			}
		case "firstname", "givenname":
			if pui != nil {
				attr = pui.FirstName
			} else {
				attr = dui.FirstName
			}
		case "lastname", "surname":
			if pui != nil {
				attr = pui.LastName
			} else {
				attr = dui.LastName
			}
		case "phone":
			if pui != nil {
				attr = pui.Phone
			} else {
				attr = dui.Phone
			}
		case "":
			w := getLockedWorker(r.tid)
			w.Unlock()
			w.Log(robot.Error, "empty attribute in call to GetUserAttribute")
			return &robot.AttrRet{"", robot.AttributeNotFound}
		}
		if len(attr) > 0 {
			return &robot.AttrRet{attr, robot.Ok}
		}
	}
	conn := getConnectorForProtocol(protocol)
	if conn == nil {
		return &robot.AttrRet{"", robot.Failed}
	}
	attr, ret := conn.GetProtocolUserAttribute(user, a)
	return &robot.AttrRet{attr, ret}
}

// see robot/robot.go
func (r Robot) GetSenderAttribute(a string) *robot.AttrRet {
	a = strings.ToLower(a)
	protocol := protocolFromIncoming(r.Incoming, r.Protocol)
	var pui *UserInfo
	var dui *DirectoryUser
	if pm, exists := r.maps.userProto[protocol]; exists {
		pui = pm[r.User]
	}
	dui, _ = r.maps.user[r.User]
	switch a {
	case "name", "username", "handle", "user":
		return &robot.AttrRet{r.User, robot.Ok}
	case "id", "internalid", "protocolid":
		return &robot.AttrRet{r.ProtocolUser, robot.Ok}
	}
	if pui != nil || dui != nil {
		var attr string
		switch a {
		case "mail", "email":
			if pui != nil {
				attr = pui.Email
			} else {
				attr = dui.Email
			}
		case "fullname", "realname":
			if pui != nil {
				attr = pui.FullName
			} else {
				attr = dui.FullName
			}
		case "firstname", "givenname":
			if pui != nil {
				attr = pui.FirstName
			} else {
				attr = dui.FirstName
			}
		case "lastname", "surname":
			if pui != nil {
				attr = pui.LastName
			} else {
				attr = dui.LastName
			}
		case "phone":
			if pui != nil {
				attr = pui.Phone
			} else {
				attr = dui.Phone
			}
		case "":
			w := getLockedWorker(r.tid)
			w.Unlock()
			w.Log(robot.Error, "empty attribute in call to GetUserAttribute")
			return &robot.AttrRet{"", robot.AttributeNotFound}
		}
		if len(attr) > 0 {
			return &robot.AttrRet{attr, robot.Ok}
		}
	}
	user := r.ProtocolUser
	if len(user) == 0 {
		user = r.User
	}
	conn := getConnectorForProtocol(protocol)
	if conn == nil {
		return &robot.AttrRet{"", robot.Failed}
	}
	attr, ret := conn.GetProtocolUserAttribute(user, a)
	return &robot.AttrRet{attr, ret}
}
