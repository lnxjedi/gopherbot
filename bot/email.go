package bot

import (
	"bytes"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/jordan-wright/email"
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
// It returns an error and RetVal != 0 if there's a problem.
func (r *Robot) Email(subject string, messageBody *bytes.Buffer, html ...bool) (ret RetVal) {
	mailAttr := r.GetSenderAttribute("email")
	if mailAttr.RetVal != Ok {
		return NoUserEmail
	}
	return r.realEmail(subject, mailAttr.Attribute, messageBody, html...)
}

// EmailUser is a method for sending an email to a specified user. See Email.
func (r *Robot) EmailUser(user, subject string, messageBody *bytes.Buffer, html ...bool) (ret RetVal) {
	mailAttr := r.GetUserAttribute(user, "email")
	if mailAttr.RetVal != Ok {
		return NoUserEmail
	}
	return r.realEmail(subject, mailAttr.Attribute, messageBody, html...)
}

// EmailAddress is a method for sending an email to a specified address. See Email.
func (r *Robot) EmailAddress(address, subject string, messageBody *bytes.Buffer, html ...bool) (ret RetVal) {
	return r.realEmail(subject, address, messageBody, html...)
}

func (r *Robot) realEmail(subject, mailTo string, messageBody *bytes.Buffer, html ...bool) (ret RetVal) {
	var mailFrom, botName string

	mailAttr := r.GetBotAttribute("email")
	if mailAttr.RetVal != Ok || mailAttr.Attribute == "" {
		Log(Error, "Email send requested but robot has no Email set in config")
		return NoBotEmail
	}
	mailFrom = mailAttr.Attribute
	// We can live without a full name
	botAttr := r.GetBotAttribute("fullName")
	if botAttr.RetVal != Ok {
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
	if botCfg.mailConf.Authtype == "plain" {
		host := strings.Split(botCfg.mailConf.Mailhost, ":")[0]
		a = smtp.PlainAuth("", botCfg.mailConf.User, botCfg.mailConf.Password, host)
		Log(Debug, fmt.Sprintf("Sending authenticated email to \"%s\" from \"%s\" via \"%s\" with user: %s, password: xxxx, and host: %s",
			mailTo,
			from,
			botCfg.mailConf.Mailhost,
			botCfg.mailConf.User,
			host,
		))
	} else {
		Log(Debug, fmt.Sprintf("Sending unauthenticated email to \"%s\" from \"%s\" via \"%s\"",
			mailTo,
			from,
			botCfg.mailConf.Mailhost,
		))
	}

	err := e.Send(botCfg.mailConf.Mailhost, a)
	if err != nil {
		err = fmt.Errorf("Sending email: %v", err)
		Log(Error, err)
		return MailError
	}
	return Ok
}
