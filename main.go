package main

import (
	"fmt"
	"os"

	"github.com/jordan-wright/email"
	"gopkg.in/alecthomas/kingpin.v2"
	"net"
	"net/smtp"
	"io/ioutil"
	"crypto/tls"
	"crypto/x509"
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

	sslInsecure = kingpin.Flag("insecure-skip-verify", "Disable TLS certificate authentication").Envar("EMAIL_INSECURE").Default("false").Bool()
	sslCA = kingpin.Flag("cacert", "Specify a custom CA certificate to verify against").Envar("EMAIL_CACERT").String()
)

var Version = "0.0.0-dev"

func main() {
	kingpin.CommandLine.HelpFlag.Short('h')
	kingpin.Version(Version)
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
		tlsConf := new(tls.Config)
		tlsConf.InsecureSkipVerify = *sslInsecure
		fmt.Println(*sslInsecure)
		if *sslCA != "" {
			certs := x509.NewCertPool()

			pemData, err := ioutil.ReadFile(*sslCA)
			if err != nil {
				println("Error loading custom root CA:", *sslCA)
				return err
			}
			certs.AppendCertsFromPEM(pemData)
			tlsConf.RootCAs = certs
		}


		sendPool, perr := email.NewPool(
			net.JoinHostPort(*host, fmt.Sprintf("%v", *port)),
			*poolsize,
			smtp.PlainAuth("", *username, *password, *host),
			tlsConf,
		)
		if perr != nil {
			println("Error creating email pool:", perr.Error())
			return perr
		}
		//defer sendPool.Close()

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
	    println("Error sending mail:", err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}
