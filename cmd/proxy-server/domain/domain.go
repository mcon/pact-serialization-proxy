package domain

import (
	"fmt"
	"github.com/mcon/pact-serialization-proxy/cmd/proxy-server/serialization"
	"github.com/mkideal/cli"
	"sync"
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

type UniqueInteractionIdentifier struct {
	method string
	path   string
	query  string
}

func CreateUniqueInteractionIdentifier(method string, path string, query string) UniqueInteractionIdentifier {
	return UniqueInteractionIdentifier{
		method: method,
		path:   path,
		query:  query,
	}
}

func CreateUniqueInteractionIdentifierFromInteraction(interaction *serialization.ProviderServiceInteraction) UniqueInteractionIdentifier {
	return UniqueInteractionIdentifier{
		method: interaction.Request.Method,
		path:   interaction.Request.Path,
		query:  interaction.Request.Query,
	}
}

// In general, there can be multiple interactions per endpoint, whose serialization can differ - in the longer term
// this will have to be a map[UniqueInteractionIdentifier] -> []*ProviderServiceInteraction.
type InteractionLookup struct {
	_map map[UniqueInteractionIdentifier]*serialization.ProviderServiceInteraction
	lock sync.Mutex
}

func (il InteractionLookup) Get(identifier UniqueInteractionIdentifier) (*serialization.ProviderServiceInteraction, bool) {
	il.lock.Lock()
	defer il.lock.Unlock()

	value, success := il._map[identifier]
	return value, success
}
func (il InteractionLookup) Set(identifier UniqueInteractionIdentifier, interaction *serialization.ProviderServiceInteraction) error {
	il.lock.Lock()
	defer il.lock.Unlock()

	// Don't try to overwrite
	_, failure := il._map[identifier]
	if failure {
		return fmt.Errorf("key %T already in map", identifier)
	}

	il._map[identifier] = interaction
	fmt.Println("Added path: ", identifier)
	return nil
}
func CreateEmptyInteractionLookup() InteractionLookup {
	return InteractionLookup{
		_map: map[UniqueInteractionIdentifier]*serialization.ProviderServiceInteraction{},
		lock: sync.Mutex{},
	}
}

func CreateInteractionLookupFromContract(contract *serialization.PactContract) InteractionLookup {
	interactionLookup := CreateEmptyInteractionLookup()
	for _, interaction := range contract.Interactions {
		key := CreateUniqueInteractionIdentifierFromInteraction(&interaction)
		err := interactionLookup.Set(key, &interaction)

		// A valid PactContract shouldn't repeat any interactions, this err should therefore never be non-nil
		if err != nil {
			panic(err)
		}
	}
	return interactionLookup
}
