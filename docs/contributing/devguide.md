## Local Setup

How to run this repo locally (after cloning):

* First [install a recent version of go](https://go.dev/doc/install) (see [go.mod](../../go.mod) for the minimum version).

* Then run the following:

````bash
cd bitwarden-sdk-server/

# install go dependencies
go mod tidy

# launch server
go run main.go serve --insecure --hostname :9998 --debug

# alternatively you can launch the server with the delve debugger:
go install github.com/go-delve/delve/cmd/dlv@v1.26.1

dlv debug main.go -- serve --insecure --hostname :9998 --debug
# optionally set a breakpoint at a given function
break funcs listSecretsHandler
# and start code execution
continue
````
