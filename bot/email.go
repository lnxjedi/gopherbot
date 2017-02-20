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
// robot.. It relies on both the robot and the user having an email address.
// For the robot, this can be conifigured in gopherbot.conf, Email attribute.
// For the user, this should be provided by the chat protocol, or in
// gopherbot.conf. (TODO: not yet implemented)
// It returns an error and RetVal != 0 if there's a problem.
func (r *Robot) Email(subject string, messageBody *bytes.Buffer) (ret RetVal) {
	var mailFrom, botName, mailTo string

	mailAttr := r.GetBotAttribute("email")
	if ret != Ok {
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
	mailAttr = r.GetSenderAttribute("email")
	if mailAttr.RetVal != Ok {
		return NoUserEmail
	}
	mailTo = mailAttr.Attribute

	e := email.NewEmail()
	from := fmt.Sprintf("%s <%s>", botName, mailFrom)
	e.From = from
	e.To = []string{mailTo}
	e.Subject = subject
	e.Text = messageBody.Bytes()

	var a smtp.Auth
	if b.mailConf.Authtype == "plain" {
		host := strings.Split(b.mailConf.Mailhost, ":")[0]
		a = smtp.PlainAuth("", b.mailConf.User, b.mailConf.Password, host)
		r.Log(Debug, fmt.Sprintf("Sending authenticated email to \"%s\" from \"%s\" via \"%s\" with user: %s, password: %s and host: %s",
			mailTo,
			from,
			b.mailConf.Mailhost,
			b.mailConf.User,
			b.mailConf.Password,
			host,
		))
	} else {
		r.Log(Debug, fmt.Sprintf("Sending unauthenticated email to \"%s\" from \"%s\" via \"%s\"",
			mailTo,
			from,
			b.mailConf.Mailhost,
		))
	}

	err := e.Send(b.mailConf.Mailhost, a)
	if err != nil {
		err = fmt.Errorf("Sending email: %v", err)
		r.Log(Error, err)
		return MailError
	}
	return Ok
}
