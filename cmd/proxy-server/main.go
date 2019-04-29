package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/mcon/pact-serialization-proxy/cmd/proxy-server/controllers"
	"github.com/mcon/pact-serialization-proxy/cmd/proxy-server/state"
	"github.com/mkideal/cli"
)

//sd
// TODO 1: Commit this to a repo
// TODO 2: Clean up validation logic, factor out into files, and add tests
// TODO 3: Add ability to read and write pact files
// TODO 4: Hack up the ability to act in mock verification

func main() {
	cli.Run(state.ParsedArgs, func(ctx *cli.Context) error {
		argv := ctx.Argv().(*state.CliArgs)
		ctx.String("%s\n", argv.PactDir)
		runWebHost()
		return nil
	})
}

func runWebHost() {
	r := gin.Default()
	r.DELETE("/interactions", controllers.HandleInteractionsDelete)
	r.GET("/interactions/verification", controllers.HandleGetVerification)
	r.POST("/interactions", controllers.HandleInteractions)
	r.POST("/pact", controllers.WritePactToFile)
	r.NoRoute(controllers.HandleDynamicEndpoints)
	r.Run(fmt.Sprintf("%s:%d", state.ParsedArgs.Host, state.ParsedArgs.Port))
}
