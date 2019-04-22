package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"sync"

	"net/http"
	"net/url"

	"github.com/Jeffail/gabs"
	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
)

var RubyCoreUrl = "http://localhost:8888"
var urlResponseProtoMap = map[string]*gabs.Container{}
var lock = sync.Mutex{}

//sd
// TODO 1: Commit this to a repo
// TODO 2: Clean up validation logic, factor out into files, and add tests
// TODO 3: Add ability to read and write pact files
// TODO 4: Hack up the ability to act in mock verification

func testFileDescriptorLogic() {
	dat, err := ioutil.ReadFile("/home/matt/RiderProjects/pact-protobuf-tester/Provider/Contracts/sample.desc.test")

	fdc := &descriptor.FileDescriptorSet{}
	if err == nil {
		err = proto.Unmarshal(dat, fdc)
	} else {
		fmt.Println("Reading file error")
	}
	var d *desc.FileDescriptor
	if err == nil {
		d, err = desc.CreateFileDescriptorFromSet(fdc)
	} else {
		fmt.Println(err)
		fmt.Println("Unmarshalling error")
	}

	if err == nil {
		messages := d.GetMessageTypes()
		for _, v := range messages {
			fmt.Println(v.GetName())
		}
		//d.FindMessage
		// fmt.Println("Managed to find message: ", md)
	} else {
		fmt.Println("Failed to create FileDescriptor from set")
	}
	//dynamic.Message
}

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
	runHost()
}

func handleInteractionsDelete(c *gin.Context) {
	urlResponseProtoMap = make(map[string]*gabs.Container)

	reqUrl, err := url.Parse(c.Request.URL.Path)
	req := &http.Request{
		URL:    reqUrl,
		Method: c.Request.Method,
		Header: c.Request.Header,
		Body:   c.Request.Body}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		c.Abort()
		return
	}

	for k, vArr := range response.Header {
		for _, v := range vArr {
			c.Writer.Header().Add(k, v)
		}
	}
}

func handleGetVerification(c *gin.Context) {
	reqUrl, err := url.Parse(c.Request.URL.Path)
	req := &http.Request{
		URL:    reqUrl,
		Method: c.Request.Method,
		Header: c.Request.Header,
		Body:   c.Request.Body}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		c.Abort()
		return
	}

	for k, vArr := range response.Header {
		for _, v := range vArr {
			c.Writer.Header().Add(k, v)
		}
	}
}

func handleInteractions(c *gin.Context) {
	reqHdrs := c.Request.Header
	fmt.Println(reqHdrs)

	reqUrl, err := url.Parse(RubyCoreUrl + c.Request.URL.Path)
	jsonBytes, err := ioutil.ReadAll(c.Request.Body)

	reader := bytes.NewBuffer(jsonBytes)
	requestBody := ioutil.NopCloser(reader)
	fmt.Println(err)
	req := &http.Request{
		URL:    reqUrl,
		Method: "POST",
		Header: c.Request.Header,
		Body:   requestBody}
	response, err := http.DefaultClient.Do(req)

	jsonParsed, err := gabs.ParseJSON(jsonBytes)

	lock.Lock()
	defer lock.Unlock()
	path, ok := jsonParsed.Path("request.path").Data().(string)
	fmt.Println("Added path: " + path)
	urlResponseProtoMap[path] = jsonParsed

	fmt.Println(ok)
	fmt.Println(err)

	resp := new(bytes.Buffer)
	resp.ReadFrom(response.Body)

	c.Writer.WriteString(resp.String())
}

func handleDynamicEndpoints(c *gin.Context) {
	// c.JSON(200, gin.H{
	// 	"message": "wohooooo",
	// })
	ul, err := url.ParseRequestURI(RubyCoreUrl + c.Request.URL.Path)
	if err != nil {
		c.Abort()
	}
	jsonBytes, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.Abort()
	}
	reader := bytes.NewBuffer(jsonBytes)
	requestBody := ioutil.NopCloser(reader)

	req := &http.Request{
		URL:    ul,
		Method: c.Request.Method,
		Header: c.Request.Header,
		Body:   requestBody}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		c.Abort()
		return
	}

	lookedupInteraction := urlResponseProtoMap[c.Request.URL.Path]
	responseJson, err := ioutil.ReadAll(response.Body)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	msgDescriptor, err := getMessageDescriptorFromBody(lookedupInteraction)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	protoMessage := dynamic.NewMessage(msgDescriptor)
	decodedJson, _ := gabs.ParseJSON(responseJson)
	fmt.Println(decodedJson)
	protoMessage.UnmarshalJSON(responseJson)

	protoJsonRep, err := protoMessage.Marshal()
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	c.Data(200, "application/octet-stream", protoJsonRep)

	// TODO: If encoding.type exists and is "protobuf" then enforce that route is a key
	// in the map - then try serialization.

	for k, vArr := range response.Header {
		for _, v := range vArr {
			c.Writer.Header().Add(k, v)
		}
	}

	c.DataFromReader(response.StatusCode, response.ContentLength,
		"application/json", response.Body, map[string]string{})
}

func writePactToFile(c *gin.Context) {
	// Call mock service, also save our pact output with additional bits in it when we receive response.
}

func runHost() {
	r := gin.Default()
	r.DELETE("/interactions", handleInteractionsDelete)
	r.GET("/interactions/verification", handleGetVerification)
	r.POST("/interactions", handleInteractions)
	r.POST("/pact", writePactToFile)
	r.NoRoute(handleDynamicEndpoints)
	r.Run() // listen and serve on 0.0.0.0:8080
}
