package state

import (
	"sync"

	"github.com/Jeffail/gabs"
	"github.com/mkideal/cli"
)

type CliArgs struct {
	cli.Helper
	Verificaion bool   `cli:"verification" usage:"set if the server is being used in pact verification"`
	PactDir     string `cli:"*pact-dir" usage:"directory to store pact: --pact-dir <directory>"`
	LogDir      string `cli:"*log-dir" usage:"directory to store process log: --log-dir <directory>"`
	Port        int    `cli:"*port" usage:"port on which to run the server: --port <port>"`
	Host        string `cli:"host" usage:"host name on which to run the server: --pact-dir <directory>" dft:"localhost"`
	RubyCoreUrl string `cli:"*ruby-core-url" usage:"URL where the Ruby core is running --ruby-core-url <url>"`
	// TODO: Should add support for SSL
}

var ParsedArgs = new(CliArgs)
var RubyCoreUrl = "http://localhost:8888"
var UrlResponseProtoMap = map[string]*gabs.Container{}
var Lock = sync.Mutex{}
