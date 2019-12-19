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
	"github.com/mcon/pact-serialization-proxy/cmd/proxy-server/state"
	"github.com/mkideal/cli"
)

func main() {
	cli.Run(state.ParsedArgs, func(ctx *cli.Context) error {
		state.ParsedArgs = ctx.Argv().(*state.CliArgs)
		if state.ParsedArgs.Verificaion {
			loadInteractionsFromPactFile()
		}

		deps := controllers.RealDependencies()
		SetupRouter(deps).Run(fmt.Sprintf("%s:%d", state.ParsedArgs.Host, state.ParsedArgs.Port))
		return nil
	})
}

func loadInteractionsFromPactFile() {
	dat, err := ioutil.ReadFile(state.ParsedArgs.PactDir)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	pactContract := serialization.PactContract{}
	err = json.Unmarshal(dat, pactContract)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	state.UrlResponseProtoMap = domain.CreateInteractionLookupFromContract(&pactContract)
}

func SetupRouter(deps *controllers.Dependencies) *gin.Engine {
	r := gin.Default()
	r.DELETE("/interactions", deps.HandleInteractionDelete)
	r.GET("/interactions/verification", deps.HandleGetVerification)
	r.POST("/interactions", deps.HandleInteractionAdd)
	r.POST("/pact", deps.WritePactToFile)
	if state.ParsedArgs.Verificaion {
		r.NoRoute(deps.HandleVerificationDynamicEndpoints)
	} else {
		r.NoRoute(deps.HandleDynamicEndpoints)
	}
	// TODO: Need to support provider states - this will entail performing some matching on the request in order to work
	// out which registered interaction a request made by the application under test pertains to (given the serialization
	// for different interactions for a given endpoint may vary).
	return r
}
