# Emailcli

Because surprisingly, everything else out there just barely fails to
be useful to me.

This utility does exactly one thing: wrap a Golang email library in a
command line interface.

## Install

    go get github.com/wrouesnel/emailcli

## Usage

```
email --username test@gmail.com --password somepassword \
    --host smtp.gmail.com --port 587 \
    --subject "Test mail" \
    --body "Test Body" test@gmail.com
```

For security, it also supports reading settings from environment
variables:
```
export EMAIL_PASSWORD=somepassword
email --username test@gmail.com \
    --host smtp.gmail.com --port 587 \
    --subject "Test mail" \
    --body "Test Body" test@gmail.com
```

All command line variables can be used as environment variables by
appending EMAIL_ to the parameter name and capitalizing.