package main

import (
	"fmt"
	"os"

	"github.com/jordan-wright/email"
	"gopkg.in/alecthomas/kingpin.v2"
	"net"
	"net/smtp"
	"io/ioutil"
)

var (
	username = kingpin.Flag("username", "Username to authenticate to the SMTP server with").Envar("EMAIL_USERNAME").String()
	password = kingpin.Flag("password", "Password to authenticate to the SMTP server with").Envar("EMAIL_PASSWORD").String()

	//usetls = kingpin.Flag("use-tls", "Use TLS to authenticate").Envar("EMAIL_USETLS").Bool()
	host =kingpin.Flag("host", "Hostname").Envar("EMAIL_HOST").String()
	port = kingpin.Flag("port", "Port number").Envar("EMAIL_PORT").Default("25").Uint16()

	attachments = kingpin.Flag("attach", "Files to attach to the email.").Envar("EMAIL_ATTACH").ExistingFiles()

	subject = kingpin.Flag("subject", "Subject line of email.").Envar("EMAIL_SUBJECT").String()
	body = kingpin.Flag("body", "Body of email. Read from stdin if blank.").Envar("EMAIL_BODY").String()

	from = kingpin.Flag("from", "From address for email").Envar("EMAIL_FROM").String()
	to = kingpin.Arg("to", "Email recipients").Strings()

	timeout = kingpin.Flag("timeout", "Timeout for mail sending").Envar("EMAIL_TIMEOUT").Duration()
	poolsize = kingpin.Flag("concurrent-sends", "Max concurrent email send jobs").Envar("EMAIL_CONCURRENT_SENDS").Default("1").Int()
)

func main() {
	kingpin.Parse()

	if *timeout == 0 {
		*timeout = -1
	}

	var bodytxt []byte
	if *body == "" {
		var err error
		bodytxt, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			println(err)
			os.Exit(1)
		}
	} else {
		bodytxt = []byte(*body)
	}

	err:= func() error {
		sendPool := email.NewPool(
			net.JoinHostPort(*host, fmt.Sprintf("%v", *port)),
			*poolsize,
			smtp.PlainAuth("", *username, *password, *host),
		)
		defer sendPool.Close()

		for _, recipient := range *to {
			m := email.NewEmail()
			m.From = *from
			m.To = []string{recipient}
			m.Subject = *subject
			m.Text = bodytxt

			for _, filename := range *attachments {
				_, err := m.AttachFile(filename)
				if err != nil {
					println(err)
					return err
				}
			}

			if err := sendPool.Send(m, *timeout); err != nil {
				println("Error sending mail:", recipient, err.Error())
			}
		}
		return nil
	}()

	if err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
