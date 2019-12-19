package domain

import (
	"fmt"
	"github.com/mcon/pact-serialization-proxy/cmd/proxy-server/serialization"
	"sync"
)

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
