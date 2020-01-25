package pactContractHandler

import (
	"github.com/mcon/pact-serialization-proxy/cmd/proxy-server/domain"
	"github.com/mcon/pact-serialization-proxy/cmd/proxy-server/serialization"
)

func PopulateContractFromInteractions(contract *serialization.PactContract, interactionLookup *domain.InteractionLookup) {
	for i, _ := range contract.Interactions {
		lookupKey := domain.CreateUniqueInteractionIdentifierFromInteraction(&contract.Interactions[i])
		// TODO: A single path could have many different binary encodings (e.g. 400 could return different data structure to 200) - also, request/response different too
		locallyRecordedInteraction, success := interactionLookup.Get(lookupKey)
		if success {
			// TODO: If there's no encoding information, as it stands there'll be an empty Encoding field in the resulting contract
			contract.Interactions[i].Response.Encoding = locallyRecordedInteraction.Response.Encoding
			contract.Interactions[i].Request.Encoding = locallyRecordedInteraction.Request.Encoding
		}
	}
}
