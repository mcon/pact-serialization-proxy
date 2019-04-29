package main

import (
	"bytes"
	"fmt"
	"io/ioutil"

	"net/http"
	"net/url"

	"github.com/Jeffail/gabs"
	"github.com/gin-gonic/gin"
	"github.com/jhump/protoreflect/dynamic"
)

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
