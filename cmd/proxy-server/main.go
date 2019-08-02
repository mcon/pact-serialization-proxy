package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/Jeffail/gabs"
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

	json, err := gabs.ParseJSON(dat)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	interactions := json.Path("interactions")
	fmt.Println(interactions)

	children, err := interactions.Children()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	for _, child := range children {
		state.UrlResponseProtoMap[child.Path("request.path").Data().(string)] = child
	}
}

func SetupRouter(deps *controllers.Dependencies) *gin.Engine {
	r := gin.Default()
	r.DELETE("/interactions", deps.HandleInteractionsDelete)
	r.GET("/interactions/verification", deps.HandleGetVerification)
	r.POST("/interactions", deps.HandleInteractions)
	r.POST("/pact", deps.WritePactToFile)
	if state.ParsedArgs.Verificaion {
		r.NoRoute(deps.HandleVerificationDynamicEndpoints)
	} else {
		r.NoRoute(deps.HandleDynamicEndpoints)
	}
	return r
}
