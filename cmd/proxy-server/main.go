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

// TODO 3: Add ability to read and write pact files
// TODO 4: Hack up the ability to act in mock verification

func main() {
	cli.Run(state.ParsedArgs, func(ctx *cli.Context) error {
		state.ParsedArgs = ctx.Argv().(*state.CliArgs)
		if state.ParsedArgs.Verificaion {
			loadInteractionsFromPactFile()
		}
		runWebHost()
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

func runWebHost() {
	r := gin.Default()
	r.DELETE("/interactions", controllers.HandleInteractionsDelete)
	r.GET("/interactions/verification", controllers.HandleGetVerification)
	r.POST("/interactions", controllers.HandleInteractions)
	r.POST("/pact", controllers.WritePactToFile)
	if state.ParsedArgs.Verificaion {
		r.NoRoute(controllers.HandleVerificationDynamicEndpoints)
	} else {
		r.NoRoute(controllers.HandleDynamicEndpoints)
	}
	r.Run(fmt.Sprintf("%s:%d", state.ParsedArgs.Host, state.ParsedArgs.Port))
}
