package entrypoint

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/smtp"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	"github.com/jordan-wright/email"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/wrouesnel/emailcli/version"
	"go.uber.org/zap"
)

type EmailOptions struct {
	Username                string        `help:"Username to authenticate to the SMTP server with"`
	Password                string        `help:"Password to authenticate to the SMTP server with"`
	Host                    string        `help:"Hostname"`
	Port                    uint16        `help:"Port number" default:"25"`
	TLSHost                 string        `help:"Hostname to use for verifying TLS (default to host if blank)" default:""`
	Attachments             []string      `help:"Files to attach to the email" type:"existingfile"`
	Subject                 string        `help:"Subject line of the email"`
	Body                    string        `help:"Body of email. Read from stdin if blank."`
	From                    string        `help:"From address for the email."`
	To                      []string      `help:"Email recipients." arg:""`
	Timeout                 time.Duration `help:"Timeout for mail sending."`
	PoolSize                uint          `name:"concurrent-sends" help:"Maximum concurrent send jobs." default:"1"`
	TLSInsecureSkipVerify   bool          `name:"insecure-skip-verify" help:"Disable TLS certificate authentication" default:"false"`
	TLSCertificateAuthority string        `name:"cacert" help:"Specify a custom CA certificate to verify against."`
	HelloHostname           string        `name:"hello-hostname" help:"Hostname to use for SMTP HELO request" default:"localhost"`
}

type Options struct {
	Logging struct {
		Level  string `help:"logging level" default:"warn"`
		Format string `help:"logging format (${enum})" enum:"console,json" default:"console"`
	} `embed:"" prefix:"logging."`

	Version bool `help:"Print the version and exit"`

	Email EmailOptions `embed:""`
}

type LaunchArgs struct {
	StdIn  io.Reader
	StdOut io.Writer
	StdErr io.Writer
	Env    map[string]string
	Args   []string
}

// Entrypoint implements the actual functionality of the program so it can be called inline from testing.
// env is normally passed the environment variable array.
//
//nolint:funlen,gocognit,gocyclo,cyclop,maintidx
func Entrypoint(args LaunchArgs) int {
	var err error
	options := Options{}

	deferredLogs := []string{}

	// Command line parsing can now happen
	parser := lo.Must(kong.New(&options, kong.Description(version.Description),
		kong.DefaultEnvars(version.EnvPrefix)))
	_, err = parser.Parse(args.Args)
	if err != nil {
		_, _ = fmt.Fprintf(args.StdErr, "Argument error: %s", err.Error())
		return 1
	}

	// Initialize logging as soon as possible
	logConfig := zap.NewProductionConfig()
	if err := logConfig.Level.UnmarshalText([]byte(options.Logging.Level)); err != nil {
		deferredLogs = append(deferredLogs, err.Error())
	}
	logConfig.Encoding = options.Logging.Format

	logger, err := logConfig.Build()
	if err != nil {
		// Error unhandled since this is a very early failure
		for _, line := range deferredLogs {
			_, _ = io.WriteString(args.StdErr, line)
		}
		_, _ = io.WriteString(args.StdErr, "Failure while building logger")
		return 1
	}

	// Install as the global logger
	zap.ReplaceGlobals(logger)

	logger.Info("Launched with command line", zap.Strings("cmdline", args.Args))

	if options.Version {
		lo.Must(fmt.Fprintf(args.StdOut, "%s", version.Version))
		return 0
	}

	logger.Info("Version Info", zap.String("version", version.Version),
		zap.String("name", version.Name),
		zap.String("description", version.Description),
		zap.String("env_prefix", version.EnvPrefix))

	appCtx, cancelFn := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM)

	go func() {
		for sig := range sigCh {
			logger.Info("Caught signal", zap.String("signal", sig.String()))
			cancelFn()
			return
		}
	}()

	logger.Info("Starting command")
	err = SendEmail(appCtx, options.Email)

	logger.Debug("Finished command")
	if err != nil {
		logger.Error("Command exited with error", zap.Error(err))
		return 1
	}
	logger.Info("Command exited successfully")
	return 0
}

// SendEmail implements the eactual email sending.
func SendEmail(ctx context.Context, options EmailOptions) error {
	logger := zap.L().With(
		zap.String("smtp_host", options.Host),
		zap.Uint16("smtp_port", options.Port),
		zap.String("smtp_username", options.Username))

	if options.Timeout == 0 {
		options.Timeout = -1
	}
	logger.Debug("Timeout set", zap.Duration("timeout", options.Timeout))

	if options.Password == "" {
		logger.Warn("Supplied SMTP password is blank!")
	}

	var bodytxt []byte
	if options.Body == "" {
		logger.Debug("Reading body text from stdin")
		var err error
		bodytxt, err = io.ReadAll(os.Stdin)
		if err != nil {
			return errors.Wrap(err, "Error reading from stdin")
		}
	} else {
		logger.Debug("Reading body text from options")
		bodytxt = []byte(options.Body)
	}

	err := func() error {
		tlsConf := new(tls.Config)
		if options.TLSHost != "" {
			logger.Debug("TLS Host set from options", zap.String("tls_host", options.TLSHost))
			tlsConf.ServerName = options.TLSHost
		} else {
			logger.Debug("TLS Host set to server hostname", zap.String("tls_host", options.Host))
			tlsConf.ServerName = options.Host
		}
		tlsConf.InsecureSkipVerify = options.TLSInsecureSkipVerify
		if tlsConf.InsecureSkipVerify {
			logger.Warn("Skipping certificate verification by user request")
		}

		if options.TLSCertificateAuthority != "" {
			logger.Debug("Loading certificate pool from file", zap.String("cacerts", options.TLSCertificateAuthority))
			certs := x509.NewCertPool()

			pemData, err := os.ReadFile(options.TLSCertificateAuthority)
			if err != nil {
				return errors.Wrapf(err, "Error loading custom root CA: %s", options.TLSCertificateAuthority)
			}

			certs.AppendCertsFromPEM(pemData)
			tlsConf.RootCAs = certs
		} else {
			logger.Debug("Using default certificate pool")
		}

		logger.Debug("Initialize email pool")
		sendPool, perr := email.NewPool(
			net.JoinHostPort(options.Host, fmt.Sprintf("%v", options.Port)),
			int(options.PoolSize),
			smtp.PlainAuth("", options.Username, options.Password, options.Host),
			tlsConf,
		)
		if perr != nil {
			return errors.Wrap(perr, "Error creating email pool")
		}
		sendPool.SetHelloHostname(options.HelloHostname)
		//defer sendPool.Close()

		logger.Info("Sending email to recipients", zap.Int("num_recipients", len(options.To)))
		numSuccessful := 0
		numFailed := 0
		for _, recipient := range options.To {
			logger.Info("Sending email", zap.String("recipient", recipient))
			m := email.NewEmail()
			m.From = options.From
			m.To = []string{recipient}
			m.Subject = options.Subject
			m.Text = bodytxt

			for _, filename := range options.Attachments {
				_, err := m.AttachFile(filename)
				if err != nil {
					logger.Error("Error attaching file", zap.String("filename", filename), zap.Error(err))
					return err
				}
			}

			if err := sendPool.Send(m, options.Timeout); err != nil {
				logger.Warn("Failed", zap.String("recipient", recipient), zap.Error(err))
				numFailed += 1
			} else {
				logger.Info("Success", zap.String("recipient", recipient))
				numSuccessful += 1
			}

		}

		if numFailed == len(options.To) {
			return errors.New("No emails were sent successfully")
		}

		logger.Warn("Some emails failed to send", zap.Int("success", numSuccessful), zap.Int("failed", numFailed), zap.Int("num_recipients", len(options.To)))

		return nil
	}()

	if err != nil {
		return errors.Wrap(err, "Error ending mail")
	}
	return nil
}
