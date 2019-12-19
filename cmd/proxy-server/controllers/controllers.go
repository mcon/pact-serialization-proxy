package controllers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mcon/pact-serialization-proxy/cmd/proxy-server/domain"
	"github.com/mcon/pact-serialization-proxy/cmd/proxy-server/pactContractHandler"
	"github.com/mcon/pact-serialization-proxy/cmd/proxy-server/serialization"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/mcon/pact-serialization-proxy/cmd/proxy-server/descriptorlogic"
)

// Mimic a dependency-injected controller setup to allow for testing.
type fileWriter func(filename string, data []byte, perm os.FileMode) error
type Dependencies struct {
	HttpClient        IHttpClient
	FileWriter        fileWriter
	InteractionLookup domain.InteractionLookup
	CliArgs           *domain.CliArgs
}

func RealDependencies(args *domain.CliArgs) *Dependencies {
	return &Dependencies{
		HttpClient:        http.DefaultClient,
		FileWriter:        ioutil.WriteFile,
		InteractionLookup: domain.CreateEmptyInteractionLookup(),
		CliArgs:           args,
	}
}

type IHttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func passThrough(c *gin.Context, deps *Dependencies) (*http.Response, error) {
	reqUrl, err := url.Parse(deps.CliArgs.RubyCoreUrl + c.Request.URL.Path)
	req := &http.Request{
		URL:    reqUrl,
		Method: c.Request.Method,
		Header: c.Request.Header,
		Body:   c.Request.Body}
	response, err := deps.HttpClient.Do(req)
	if err != nil {
		c.Abort()
		return nil, err
	}

	return response, nil
}

func (deps *Dependencies) handleInteractionDeleteInner(c *gin.Context) error {
	deps.InteractionLookup = domain.CreateEmptyInteractionLookup()

	response, _ := passThrough(c, deps)

	resp := new(bytes.Buffer)
	_, err := resp.ReadFrom(response.Body)
	if err != nil {
		return err
	}
	_, err = c.Writer.WriteString(resp.String())
	if err != nil {
		return err
	}
	c.Status(response.StatusCode)
	for k, vArr := range response.Header {
		for _, v := range vArr {
			c.Writer.Header().Add(k, v)
		}
	}
	return nil
}

func (deps *Dependencies) HandleInteractionDelete(c *gin.Context) {
	err := deps.handleInteractionDeleteInner(c)
	if err != nil {
		_ = c.AbortWithError(500, err)
	}
}

func (deps *Dependencies) handleGetVerificationInner(c *gin.Context) error {
	response, _ := passThrough(c, deps)

	resp := new(bytes.Buffer)
	_, err := resp.ReadFrom(response.Body)
	if err != nil {
		return err
	}
	_, err = c.Writer.WriteString(resp.String())
	if err != nil {
		return err
	}
	c.Status(response.StatusCode)
	for k, vArr := range response.Header {
		for _, v := range vArr {
			c.Writer.Header().Add(k, v)
		}
	}
	return nil
}

func (deps *Dependencies) HandleGetVerification(c *gin.Context) {
	err := deps.handleGetVerificationInner(c)
	if err != nil {
		_ = c.AbortWithError(500, err)
	}
}
func (deps *Dependencies) handleInteractionAddInner(c *gin.Context) error {
	reqUrl, err := url.Parse(deps.CliArgs.RubyCoreUrl + c.Request.URL.Path)
	if err != nil {
		return err
	}
	jsonBytes, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		return err
	}

	reader := bytes.NewBuffer(jsonBytes)
	requestBody := ioutil.NopCloser(reader)
	req := &http.Request{
		URL:    reqUrl,
		Method: "POST",
		Header: c.Request.Header,
		Body:   requestBody}
	response, err := deps.HttpClient.Do(req)
	if err != nil {
		return err
	}

	var unmarshalledInteraction = new(serialization.ProviderServiceInteraction)
	err = json.Unmarshal(jsonBytes, unmarshalledInteraction)
	if err != nil {
		return err
	}

	urlIdentifier := domain.CreateUniqueInteractionIdentifierFromInteraction(unmarshalledInteraction)
	err = deps.InteractionLookup.Set(urlIdentifier, unmarshalledInteraction)
	if err != nil {
		return err
	}

	resp := new(bytes.Buffer)
	_, err = resp.ReadFrom(response.Body)
	if err != nil {
		return err
	}
	_, err = c.Writer.WriteString(resp.String())
	if err != nil {
		return err
	}
	return nil
}

func (deps *Dependencies) HandleInteractionAdd(c *gin.Context) {
	err := deps.handleInteractionAddInner(c)
	if err != nil {
		_ = c.AbortWithError(500, err)
	}
}

func (deps *Dependencies) handleVerificationDynamicEndpointsInner(c *gin.Context) error {
	// TODO: Support custom serialization of request body
	reqBody, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		return err
	}
	ul, err := url.ParseRequestURI(deps.CliArgs.RubyCoreUrl + strings.TrimLeft(c.Request.URL.RequestURI(), "/"))
	if err != nil {
		return err
	}
	reader := bytes.NewBuffer(reqBody)
	requestBody := ioutil.NopCloser(reader)

	req := &http.Request{
		URL:    ul,
		Method: c.Request.Method,
		Header: c.Request.Header,
		Body:   requestBody}
	response, err := deps.HttpClient.Do(req)
	if err != nil {
		return err
	}
	responseReader := response.Body.(io.Reader)
	contentLength := response.ContentLength

	fmt.Println(c.Request.URL.Path)
	fmt.Println(deps.InteractionLookup)
	interactionKey := domain.CreateUniqueInteractionIdentifier(
		c.Request.Method,
		"/"+strings.TrimLeft(c.Request.URL.Path, "/"),
		c.Request.URL.RawQuery)
	lookedUpInteraction, success := deps.InteractionLookup.Get(interactionKey)
	if !success {
		return errors.New(fmt.Sprintf("Failed to look up interaction: %T", lookedUpInteraction))
	}
	if lookedUpInteraction.Response.Encoding.Type == "protobuf" {
		fmt.Println("Doing conversion to proto")
		msgDescriptor, err := descriptorlogic.GetMessageDescriptorFromBody(&lookedUpInteraction.Response.Encoding, c.Request.URL.Path)
		if err != nil {
			return err
		}

		responseBody, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return err
		}

		protoMessage := dynamic.NewMessage(msgDescriptor)
		err = protoMessage.Unmarshal(responseBody)
		if err != nil {
			return err
		}

		encoded, err := protoMessage.MarshalJSONIndent()
		if err != nil {
			return err
		}
		responseReader = bytes.NewReader(encoded)
		fmt.Println("Encoded contents:")
		fmt.Println(encoded)
		fmt.Println(len(encoded))
		contentLength = int64(cap(encoded))
	}

	for k, vArr := range response.Header {
		for _, v := range vArr {
			c.Writer.Header().Add(k, v)
		}
	}

	// ContentLength is set by DataFromReader below, and gin doesn't support overwritingof header values.
	c.Writer.Header().Del("content-length")

	c.DataFromReader(response.StatusCode, contentLength,
		strings.Join(response.Header["Content-Type"], "; "), responseReader, map[string]string{})
	return nil
}

func (deps *Dependencies) HandleVerificationDynamicEndpoints(c *gin.Context) {
	err := deps.handleVerificationDynamicEndpointsInner(c)
	if err != nil {
		_ = c.AbortWithError(500, err)
	}
}

func (deps *Dependencies) handleDynamicEndpointsInner(c *gin.Context) error {
	ul, err := url.ParseRequestURI(deps.CliArgs.RubyCoreUrl + c.Request.URL.Path)
	if err != nil {
		return err
	}
	jsonBytes, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		return err
	}
	reader := bytes.NewBuffer(jsonBytes)
	requestBody := ioutil.NopCloser(reader)

	req := &http.Request{
		URL:    ul,
		Method: c.Request.Method,
		Header: c.Request.Header,
		Body:   requestBody}
	response, err := deps.HttpClient.Do(req)
	if err != nil {
		return err
	}

	interactionKey := domain.CreateUniqueInteractionIdentifier(
		c.Request.Method,
		c.Request.URL.Path,
		c.Request.URL.RawQuery)
	lookedUpInteraction, success := deps.InteractionLookup.Get(interactionKey)
	if !success {
		return errors.New(fmt.Sprintf("Failed to look up interaction: %T", lookedUpInteraction))
	}
	responseJson, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	msgDescriptor, err := descriptorlogic.GetMessageDescriptorFromBody(&lookedUpInteraction.Response.Encoding, c.Request.URL.Path)
	if err != nil {
		return err
	}

	protoJsonResp, err := descriptorlogic.JsonBytesToProtobufBytes(responseJson, msgDescriptor)
	if err != nil {
		return err
	}
	c.Data(200, "application/octet-stream", protoJsonResp)

	// TODO: If encoding.type exists and is "protobuf" then enforce that route is a key
	// in the map - then try serialization.

	for k, vArr := range response.Header {
		for _, v := range vArr {
			c.Writer.Header().Add(k, v)
		}
	}

	c.DataFromReader(response.StatusCode, response.ContentLength,
		"application/json", response.Body, map[string]string{})
	return nil
}

func (deps *Dependencies) HandleDynamicEndpoints(c *gin.Context) {
	// TODO: Support custom serialization of request body
	err := deps.handleDynamicEndpointsInner(c)
	if err != nil {
		_ = c.AbortWithError(500, err)
	}
}

func (deps *Dependencies) writePactToFileInner(c *gin.Context) error {
	response, err := passThrough(c, deps)
	if err != nil {
		return err
	}

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	// Note: the Pact core is ignorant of the `Encoding` fields, and so these are looked up in our interaction map
	contract := serialization.PactContract{}
	err = json.Unmarshal(data, contract)
	if err != nil {
		return err
	}

	pactContractHandler.PopulateContractFromInteractions(&contract, deps.InteractionLookup)

	outputtedJson, err := json.Marshal(contract)
	if err != nil {
		return err
	}
	// TODO: This is sensitive to there being a trailing '/' at the end of the PactDir, but otherwise *should* work on
	// both unix and Windows
	fileDest := deps.CliArgs.PactDir + contract.Consumer + ".proto.json"
	_, err = fmt.Printf("Writing pact file to %s\n", fileDest)
	if err != nil {
		return err
	}
	err = deps.FileWriter(fileDest, outputtedJson, 0777)
	if err != nil {
		return err
	}

	c.Data(200, "application/json", outputtedJson)
	return nil
}

func (deps *Dependencies) WritePactToFile(c *gin.Context) {
	err := deps.writePactToFileInner(c)
	if err != nil {
		_ = c.AbortWithError(500, err)
	}
}
