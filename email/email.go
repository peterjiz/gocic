package email

import (
	"bytes"
	"crypto/tls"
	"log"
	"net/mail"
	"net/smtp"
	"strconv"
	"text/template"
	"strings"
)

type EmailUser struct {
	Username    string
	Password    string
	EmailServer string
	Port        int
}

type SmtpTemplateData struct {
	From    string
	To      string
	Subject string
	Body    string
	EndName string
}

//http://nathanleclaire.com/blog/2013/12/17/sending-email-from-gmail-using-golang/
//https://gist.github.com/nathanleclaire/8662755
//https://gist.github.com/chrisgillis/10888032
func (emailSender *EmailUser) SendEmail(from mail.Address, recipients []string, subject, body, endname string) error {
	const emailTemplate = `From: {{.From}}
To: {{.To}}
Subject: {{.Subject}}

{{.Body}}

{{.EndName}}
`
	var err error
	var doc bytes.Buffer

	// Set the context for the email template.
	context := &SmtpTemplateData{
		from.String(),
		strings.Join(recipients, " ; "),
		subject,
		body,
		endname,
	}

	// Create a new template for our SMTP message.
	t := template.New("emailTemplate")
	if t, err = t.Parse(emailTemplate); err != nil {
		log.Print("error trying to parse mail template ", err)
		return err
	}

	// Apply the values we have initialized in our struct context to the template.
	if err = t.Execute(&doc, context); err != nil {
		log.Print("error trying to execute mail template ", err)
		return err
	}

	// Authenticate with Email Server (analagous to logging in to your Email account in the browser)
	auth := smtp.PlainAuth("", emailSender.Username, emailSender.Password, emailSender.EmailServer)

	if emailSender.Port != 465 {
		// Actually perform the step of sending the email - (3) above
		err = smtp.SendMail(emailSender.EmailServer+":"+strconv.Itoa(emailSender.Port), auth, emailSender.Username, recipients, doc.Bytes())
		if err != nil {
			log.Print("ERROR: attempting to send a mail ", err)
			return err
		}
	} else {
		// TLS config
		tlsconfig := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         emailSender.EmailServer,
		}

		// Here is the key, you need to call tls.Dial instead of smtp.Dial
		// for smtp servers running on 465 that require an ssl connection
		// from the very beginning (no starttls)
		conn, err := tls.Dial("tcp", emailSender.EmailServer+":"+strconv.Itoa(emailSender.Port), tlsconfig)
		if err != nil {
			return err
		}

		c, err := smtp.NewClient(conn, emailSender.EmailServer)
		if err != nil {
			return err
		}

		// Auth
		if err = c.Auth(auth); err != nil {
			return err
		}

		// To && From
		if err = c.Mail(from.Address); err != nil {
			return err
		}

		for _, receiver := range recipients {
			if err = c.Rcpt(receiver); err != nil {
				return err
			}
		}

		// Data
		w, err := c.Data()
		if err != nil {
			return err
		}

		_, err = w.Write(doc.Bytes())
		if err != nil {
			return err
		}

		err = w.Close()
		if err != nil {
			return err
		}

		c.Quit()
	}

	return nil

}
