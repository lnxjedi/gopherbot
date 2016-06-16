package bot

import (
	"bytes"
	"fmt"
	"net/smtp"
)

// Email provides a simple interface for sending the user an email from the
// robot.. It relies on both the robot and the user having an email address.
// For the robot, this can be conifigured in gopherbot.conf, Email attribute.
// For the user, this should be provided by the chat protocol, or in
// gopherbot.conf. (TODO: not yet implemented)
// It returns an error and BotRetVal != 0 if there's a problem.
func (r *Robot) Email(subject string, messageBody *bytes.Buffer) (ret BotRetVal) {
	var mailFrom, botName, mailTo string

	mailAttr := r.GetBotAttribute("email")
	if ret != Ok {
		return NoBotEmail
	}
	mailFrom = mailAttr.Attribute
	// We can live without a full name
	botAttr := r.GetBotAttribute("fullName")
	if botAttr.BotRetVal != Ok {
		botName = "Gopherbot"
	} else {
		botName = botAttr.Attribute
	}
	mailAttr = r.GetSenderAttribute("email")
	if mailAttr.BotRetVal != Ok {
		return NoUserEmail
	}
	mailTo = mailAttr.Attribute
	var messageBuffer bytes.Buffer
	fmt.Fprintf(&messageBuffer, "From: %s <%s>\r\n", botName, mailFrom)
	fmt.Fprintf(&messageBuffer, "Subject: %s\r\n\r\n", subject)
	// Connect to the remote SMTP server.
	// TODO: make email configurable for authenticated, see Go source for SendMail
	c, err := smtp.Dial("127.0.0.1:25")
	if err != nil {
		err = fmt.Errorf("Sending email: %v", err)
		r.Log(Error, err)
		return MailError
	}
	if err := c.Mail(mailFrom); err != nil {
		err = fmt.Errorf("Setting sender to %s: %v", mailFrom, err)
		r.Log(Error, err)
		return MailError
	}
	if err := c.Rcpt(mailTo); err != nil {
		err = fmt.Errorf("Setting recipient to %s: %v", mailTo, err)
		r.Log(Error, err)
		return MailError
	}
	// Send the email body.
	wc, err := c.Data()
	if err != nil {
		err = fmt.Errorf("Starting message body: %v", err)
		r.Log(Error, err)
		return MailError
	}
	_, err = wc.Write(messageBuffer.Bytes())
	if err != nil {
		err = fmt.Errorf("Sending message body: %v", err)
		r.Log(Error, err)
		return MailError
	}
	_, err = wc.Write(messageBody.Bytes())
	if err != nil {
		err = fmt.Errorf("Sending message body: %v", err)
		r.Log(Error, err)
		return MailError
	}
	err = wc.Close()
	if err != nil {
		err = fmt.Errorf("Closing message body: %v", err)
		r.Log(Error, err)
		return MailError
	}
	err = c.Quit()
	if err != nil {
		err = fmt.Errorf("Closing mail connection: %v", err)
		r.Log(Error, err)
		return MailError
	}
	return Ok
}
