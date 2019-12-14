package controllers

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"net/http"
	"net/url"

	"github.com/Jeffail/gabs"
	"github.com/gin-gonic/gin"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/mcon/pact-serialization-proxy/cmd/proxy-server/descriptorlogic"
	"github.com/mcon/pact-serialization-proxy/cmd/proxy-server/state"
)

type Dependencies struct {
	HttpClient IHttpClient
}

func RealDependencies() *Dependencies {
	return &Dependencies{
		HttpClient: http.DefaultClient,
	}
}

type IHttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func passThrough(c *gin.Context, deps *Dependencies) (*http.Response, error) {
	reqUrl, err := url.Parse(state.ParsedArgs.RubyCoreUrl + c.Request.URL.Path)
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

func (deps *Dependencies) HandleInteractionsDelete(c *gin.Context) {
	state.UrlResponseProtoMap = make(map[string]*gabs.Container)

	response, _ := passThrough(c, deps)

	resp := new(bytes.Buffer)
	resp.ReadFrom(response.Body)
	c.Writer.WriteString(resp.String())
	c.Status(response.StatusCode)
	for k, vArr := range response.Header {
		for _, v := range vArr {
			c.Writer.Header().Add(k, v)
		}
	}
}

func (deps *Dependencies) HandleGetVerification(c *gin.Context) {
	response, _ := passThrough(c, deps)

	resp := new(bytes.Buffer)
	resp.ReadFrom(response.Body)
	c.Writer.WriteString(resp.String())
	c.Status(response.StatusCode)
	for k, vArr := range response.Header {
		for _, v := range vArr {
			c.Writer.Header().Add(k, v)
		}
	}
}

func (deps *Dependencies) HandleInteractions(c *gin.Context) {
	reqUrl, err := url.Parse(state.ParsedArgs.RubyCoreUrl + c.Request.URL.Path)
	jsonBytes, err := ioutil.ReadAll(c.Request.Body)

	reader := bytes.NewBuffer(jsonBytes)
	requestBody := ioutil.NopCloser(reader)
	fmt.Println(err)
	req := &http.Request{
		URL:    reqUrl,
		Method: "POST",
		Header: c.Request.Header,
		Body:   requestBody}
	response, err := deps.HttpClient.Do(req)

	jsonParsed, err := gabs.ParseJSON(jsonBytes)

	state.Lock.Lock()
	defer state.Lock.Unlock()
	path, ok := jsonParsed.Path("request.path").Data().(string)
	fmt.Println("Added path: " + path)
	state.UrlResponseProtoMap[path] = jsonParsed

	fmt.Println(ok)
	fmt.Println(err)

	resp := new(bytes.Buffer)
	resp.ReadFrom(response.Body)

	c.Writer.WriteString(resp.String())
}

func (deps *Dependencies) HandleVerificationDynamicEndpoints(c *gin.Context) {
	// TODO: Support custom serialization of request body
	reqBody, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.Abort()
	}
	ul, err := url.ParseRequestURI(state.ParsedArgs.RubyCoreUrl + strings.TrimLeft(c.Request.URL.RequestURI(), "/"))
	if err != nil {
		c.Abort()
	}
	reader := bytes.NewBuffer(reqBody)
	requestBody := ioutil.NopCloser(reader)

	req := &http.Request{
		URL:    ul,
		Method: c.Request.Method,
		Header: c.Request.Header,
		Body:   requestBody}
	response, err := deps.HttpClient.Do(req)
	responseReader := response.Body.(io.Reader)
	contentLength := response.ContentLength

	fmt.Println(c.Request.URL.Path)
	fmt.Println(state.UrlResponseProtoMap)
	lookedupInteraction := state.UrlResponseProtoMap["/"+strings.TrimLeft(c.Request.URL.Path, "/")]
	if lookedupInteraction != nil && lookedupInteraction.ExistsP("response.encoding") {
		fmt.Println("Doing conversion to proto")
		msgDescriptor, err := descriptorlogic.GetMessageDescriptorFromBody(lookedupInteraction)
		if err != nil {
			c.AbortWithError(500, err)
			return
		}

		responseBody, err := ioutil.ReadAll(response.Body)
		if err != nil {
			c.AbortWithError(500, err)
			return
		}

		protoMessage := dynamic.NewMessage(msgDescriptor)
		protoMessage.Unmarshal(responseBody)

		encoded, err := protoMessage.MarshalJSONIndent()
		if err != nil {
			c.AbortWithError(500, err)
			return
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
}

func (deps *Dependencies) HandleDynamicEndpoints(c *gin.Context) {
	// TODO: Support custom serialization of request body
	ul, err := url.ParseRequestURI(state.ParsedArgs.RubyCoreUrl + c.Request.URL.Path)
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
	response, err := deps.HttpClient.Do(req)
	if err != nil {
		c.Abort()
		return
	}

	lookedupInteraction := state.UrlResponseProtoMap[c.Request.URL.Path]
	responseJson, err := ioutil.ReadAll(response.Body)
	if err != nil {
		_ = c.AbortWithError(500, err)
		return
	}
	msgDescriptor, err := descriptorlogic.GetMessageDescriptorFromBody(lookedupInteraction)
	if err != nil {
		_ = c.AbortWithError(500, err)
		return
	}

	protoJsonResp, err := descriptorlogic.JsonBytesToProtobufBytes(responseJson, msgDescriptor)
	if err != nil {
		_ = c.AbortWithError(500, err)
		return
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
}

func (deps *Dependencies) WritePactToFile(c *gin.Context) {
	response, err := passThrough(c, deps)
	if err != nil {
		_ = c.AbortWithError(500, err)
		return
	}

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		_ = c.AbortWithError(500, err)
		return
	}

	jsonParsed, err := gabs.ParseJSON(data)
	if err != nil {
		_ = c.AbortWithError(500, err)
		return
	}

	interactions := jsonParsed.Path("interactions")
	interactions_children, err := interactions.Children()
	if err != nil {
		_ = c.AbortWithError(500, err)
		return
	}

	for _, child := range interactions_children {
		path := child.Path("request.path").Data().(string)

		// TODO: A single path could have many different binary encodings (e.g. 400 could return different data structure to 200) - also, request/response different too
		pathSerialization := state.UrlResponseProtoMap[path]
		if pathSerialization != nil {
			encoding := pathSerialization.Path("response.encoding")
			child, err = child.SetP(encoding.Data(), "response.encoding")
			if err != nil {
				_ = c.AbortWithError(500, err)
				return
			}
		}
	}

	jsonParsed.Delete("interactions")
	jsonParsed.Array("interactions")
	for _, child := range interactions_children {
		err = jsonParsed.ArrayAppend(child.Data(), "interactions")
	}
	if err != nil {
		_ = c.AbortWithError(500, err)
		return
	}

	consumer_name := jsonParsed.Path("consumer.name").Data().(string)

	outputted_json := jsonParsed.EncodeJSON()
	fmt.Println(jsonParsed)
	fileDest := state.ParsedArgs.PactDir + consumer_name + ".proto.json"
	fmt.Println(fileDest)
	err = ioutil.WriteFile(fileDest, outputted_json, 0777)
	if err != nil {
		_ = c.AbortWithError(500, err)
		return
	}

	c.Data(200, "application/json", outputted_json)
}
