package pactContractHandler

import (
	"github.com/mcon/pact-serialization-proxy/cmd/proxy-server/domain"
	"github.com/mcon/pact-serialization-proxy/cmd/proxy-server/serialization"
)

func PopulateContractFromInteractions(contract *serialization.PactContract, interactionLookup domain.InteractionLookup) {
	for _, contractChildInteraction := range contract.Interactions {
		lookupKey := domain.CreateUniqueInteractionIdentifierFromInteraction(&contractChildInteraction)
		// TODO: A single path could have many different binary encodings (e.g. 400 could return different data structure to 200) - also, request/response different too
		locallyRecordedInteraction, success := interactionLookup.Get(lookupKey)
		if success {
			contractChildInteraction.Response.Encoding = locallyRecordedInteraction.Response.Encoding
			contractChildInteraction.Request.Encoding = locallyRecordedInteraction.Request.Encoding
		}
	}
}
