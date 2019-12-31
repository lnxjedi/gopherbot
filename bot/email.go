package bot

import (
	"bytes"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/jordan-wright/email"
	"github.com/lnxjedi/gopherbot/robot"
)

type botMailer struct {
	Mailhost string // host(:port) to send email through
	Authtype string // none, plain
	User     string // optional username for authenticated email
	Password string // optional password for authenticated email
}

// Email provides a simple interface for sending the user an email from the
// bot. It relies on both the robot and the user having an email address.
// For the robot, this can be conifigured in gopherbot.conf, Email attribute.
// For the user, this should be provided by the chat protocol, or in
// gopherbot.conf.
// It returns an error and b.RetVal != 0 if there's a problem.
func (r Robot) Email(subject string, messageBody *bytes.Buffer, html ...bool) (ret robot.RetVal) {
	mailAttr := r.GetSenderAttribute("email")
	if mailAttr.RetVal != robot.Ok {
		return robot.NoUserEmail
	}
	return r.realEmail(subject, mailAttr.Attribute, messageBody, html...)
}

// EmailUser is a method for sending an email to a specified user. See Email.
func (r Robot) EmailUser(user, subject string, messageBody *bytes.Buffer, html ...bool) (ret robot.RetVal) {
	mailAttr := r.GetUserAttribute(user, "email")
	if mailAttr.RetVal != robot.Ok {
		return robot.NoUserEmail
	}
	return r.realEmail(subject, mailAttr.Attribute, messageBody, html...)
}

// EmailAddress is a method for sending an email to a specified address. See Email.
func (r Robot) EmailAddress(address, subject string, messageBody *bytes.Buffer, html ...bool) (ret robot.RetVal) {
	return r.realEmail(subject, address, messageBody, html...)
}

func (r Robot) realEmail(subject, mailTo string, messageBody *bytes.Buffer, html ...bool) (ret robot.RetVal) {
	var mailFrom, botName string

	mailAttr := r.GetBotAttribute("email")
	if mailAttr.RetVal != robot.Ok || mailAttr.Attribute == "" {
		Log(robot.Error, "Email send requested but robot has no Email set in config")
		return robot.NoBotEmail
	}
	mailFrom = mailAttr.Attribute
	// We can live without a full name
	botAttr := r.GetBotAttribute("fullName")
	if botAttr.RetVal != robot.Ok {
		botName = "Gopherbot"
	} else {
		botName = botAttr.Attribute
	}

	e := email.NewEmail()
	from := fmt.Sprintf("%s <%s>", botName, mailFrom)
	e.From = from
	e.To = []string{mailTo}
	e.Subject = subject
	if len(html) > 0 && html[0] {
		e.HTML = messageBody.Bytes()
	} else {
		e.Text = messageBody.Bytes()
	}

	var a smtp.Auth
	if r.cfg.mailConf.Authtype == "plain" {
		host := strings.Split(r.cfg.mailConf.Mailhost, ":")[0]
		a = smtp.PlainAuth("", r.cfg.mailConf.User, r.cfg.mailConf.Password, host)
		Log(robot.Debug, "Sending authenticated email to \"%s\" from \"%s\" via \"%s\" with user: %s, password: xxxx, and host: %s",
			mailTo,
			from,
			r.cfg.mailConf.Mailhost,
			r.cfg.mailConf.User,
			host,
		)
	} else {
		Log(robot.Debug, "Sending unauthenticated email to \"%s\" from \"%s\" via \"%s\"",
			mailTo,
			from,
			r.cfg.mailConf.Mailhost,
		)
	}

	err := e.Send(r.cfg.mailConf.Mailhost, a)
	if err != nil {
		err = fmt.Errorf("Sending email: %v", err)
		Log(robot.Error, err.Error())
		return robot.MailError
	}
	return robot.Ok
}
