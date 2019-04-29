package main

import (
	"github.com/gin-gonic/gin"
	"github.com/mcon/pact-serialization-proxy/cmd/proxy-server/controllers"
)

//sd
// TODO 1: Commit this to a repo
// TODO 2: Clean up validation logic, factor out into files, and add tests
// TODO 3: Add ability to read and write pact files
// TODO 4: Hack up the ability to act in mock verification

func main() {
	runWebHost()
}

func runWebHost() {
	r := gin.Default()
	r.DELETE("/interactions", controllers.HandleInteractionsDelete)
	r.GET("/interactions/verification", controllers.HandleGetVerification)
	r.POST("/interactions", controllers.HandleInteractions)
	r.POST("/pact", controllers.WritePactToFile)
	r.NoRoute(controllers.HandleDynamicEndpoints)
	r.Run() // listen and serve on 0.0.0.0:8080
}
