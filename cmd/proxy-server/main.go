package main

import (
	"errors"
	"fmt"
	"sync"

	"github.com/Jeffail/gabs"
	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/desc"
)

var RubyCoreUrl = "http://localhost:8888"
var urlResponseProtoMap = map[string]*gabs.Container{}
var lock = sync.Mutex{}

//sd
// TODO 1: Commit this to a repo
// TODO 2: Clean up validation logic, factor out into files, and add tests
// TODO 3: Add ability to read and write pact files
// TODO 4: Hack up the ability to act in mock verification

func getMessageDescriptorFromBody(interaction *gabs.Container) (messageDescriptor *desc.MessageDescriptor, err error) {
	fmt.Println(interaction)
	fdcContainer := interaction.Path("response.encoding.description.fileDescriptorSet")
	fmt.Println(fdcContainer)
	fdcBytes := make([]byte, 0, 100000)

	fmt.Println(fdcContainer.String())
	children, _ := fdcContainer.Children()
	for _, child := range children {
		floatRepr := child.Data().(float64)
		fdcBytes = append(fdcBytes, byte(floatRepr))
	}

	fdc := &descriptor.FileDescriptorSet{}
	err = proto.Unmarshal(fdcBytes, fdc)
	if err != nil {
		return nil, err
	}

	var d *desc.FileDescriptor
	d, err = desc.CreateFileDescriptorFromSet(fdc)
	if err != nil {
		return nil, err
	}

	messages := d.GetMessageTypes()
	for _, msg := range messages {
		fmt.Println(msg.GetName(), interaction.Path("response.encoding.description.messageName").Data().(string))
		if msg.GetName() == interaction.Path("response.encoding.description.messageName").Data().(string) {
			return msg, nil
		}
	}
	return nil, errors.New("Expected route was not found in interactions: " + interaction.Path("request.path").String())
	//dynamic.Message
}

func main() {
	runWebHost()
}

func writePactToFile(c *gin.Context) {
	// Call mock service, also save our pact output with additional bits in it when we receive response.
}

func runWebHost() {
	r := gin.Default()
	r.DELETE("/interactions", handleInteractionsDelete)
	r.GET("/interactions/verification", handleGetVerification)
	r.POST("/interactions", handleInteractions)
	r.POST("/pact", writePactToFile)
	r.NoRoute(handleDynamicEndpoints)
	r.Run() // listen and serve on 0.0.0.0:8080
}
