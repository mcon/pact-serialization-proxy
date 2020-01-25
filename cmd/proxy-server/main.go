package main

import (
	"encoding/json"
	"fmt"
	"github.com/mcon/pact-serialization-proxy/cmd/proxy-server/domain"
	"github.com/mcon/pact-serialization-proxy/cmd/proxy-server/serialization"
	"io/ioutil"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/mcon/pact-serialization-proxy/cmd/proxy-server/controllers"
	"github.com/mkideal/cli"
)

func main() {
	var ParsedArgs = new(domain.CliArgs)
	cli.Run(ParsedArgs, func(ctx *cli.Context) error {
		ParsedArgs = ctx.Argv().(*domain.CliArgs)
		interactionLookup := domain.CreateEmptyInteractionLookup()
		if ParsedArgs.Verificaion {
			interactionLookup = loadInteractionsFromPactFile(ParsedArgs)
		}

		deps := controllers.RealDependencies(ParsedArgs)
		deps.InteractionLookup = interactionLookup
		return SetupRouter(deps).Run(fmt.Sprintf("%s:%d", ParsedArgs.Host, ParsedArgs.Port))
	})
}

func loadInteractionsFromPactFile(args *domain.CliArgs) *domain.InteractionLookup {
	dat, err := ioutil.ReadFile(args.PactDir)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	pactContract := serialization.PactContract{}
	err = json.Unmarshal(dat, &pactContract)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return domain.CreateInteractionLookupFromContract(&pactContract)
}

func SetupRouter(deps *controllers.Dependencies) *gin.Engine {
	r := gin.Default()
	r.DELETE("interactions", deps.HandleInteractionDelete)
	r.GET("interactions/verification", deps.HandleGetVerification)
	r.POST("interactions", deps.HandleInteractionAdd)
	r.POST("pact", deps.WritePactToFile)
	if deps.CliArgs.Verificaion {
		r.NoRoute(deps.HandleVerificationDynamicEndpoints)
	} else {
		r.NoRoute(deps.HandleDynamicEndpoints)
	}
	// TODO: Need to support provider states - this will entail performing some matching on the request in order to work
	// out which registered interaction a request made by the application under test pertains to (given the serialization
	// for different interactions for a given endpoint may vary).
	// TODO: Currently match statements are not supported: match statements mutate the body of the ServiceProviderRequest
	// and at present the body is deserialized directly - this isn't an insurmountable problem if we demand that the
	// serialization for a given endpoint is determined, for provider verification by (method * path * providerState).
	// The Ruby core gets the providerState from the environment, maybe we should do the same.
	// TODO: To handle match statements on the consumer-contract-creation-side, we don't know the request serialization,
	// and so we should assume that's going to be the same for all requests. Response serialization can be determined
	// by the status code returned by the Ruby core (as the core is actually capable of properly matching requests
	// to their corresponding interactions).

	return r
}
