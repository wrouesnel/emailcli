[![Build and Test](https://github.com/wrouesnel/emailcli/actions/workflows/integration.yml/badge.svg)](https://github.com/wrouesnel/emailcli/actions/workflows/integration.yml)
[![Release](https://github.com/wrouesnel/emailcli/actions/workflows/release.yml/badge.svg)](https://github.com/wrouesnel/emailcli/actions/workflows/release.yml)
[![Container Build](https://github.com/wrouesnel/emailcli/actions/workflows/container.yml/badge.svg)](https://github.com/wrouesnel/emailcli/actions/workflows/container.yml)
[![Coverage Status](https://coveralls.io/repos/github/wrouesnel/emailcli/badge.svg?branch=main)](https://coveralls.io/github/wrouesnel/emailcli?branch=main)


# Emailcli

Because surprisingly, everything else out there just barely fails to
be useful to me.

This utility does exactly one thing: wrap a Golang email library in a
command line interface.

## Install

Download a release binary from the [releases](https://github.com/wrouesnel/emailcli/releases/latest) page or use the container packaging:
    
    podman run -it --rm ghcr.io/wrouesnel/emailcli:latest

OR

    docker run -it --rm ghcr.io/wrouesnel/emailcli:latest

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
