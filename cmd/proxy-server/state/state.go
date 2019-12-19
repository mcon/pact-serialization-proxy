package state

import (
	"github.com/mcon/pact-serialization-proxy/cmd/proxy-server/domain"
	"github.com/mkideal/cli"
)

type CliArgs struct {
	cli.Helper
	Verificaion bool   `cli:"verification" usage:"set if the server is being used in pact verification"`
	PactDir     string `cli:"*pact-dir" usage:"directory to store pact: --pact-dir <directory>"`
	LogDir      string `cli:"log-dir" usage:"directory to store process log: --log-dir <directory>"`
	Port        int    `cli:"*port" usage:"port on which to run the server: --port <port>"`
	Host        string `cli:"host" usage:"host name on which to run the server: --pact-dir <directory>" dft:"localhost"`
	// TODO: Should make this "OutputUrl", as it's not the ruby core when doing verification.
	RubyCoreUrl string `cli:"*ruby-core-url" usage:"URL where the Ruby core is running --ruby-core-url <url>"`
	// TODO: Should add support for SSL
}

var ParsedArgs = new(CliArgs)

var UrlResponseProtoMap = domain.InteractionLookup{}
