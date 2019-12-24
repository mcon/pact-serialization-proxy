package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/mcon/pact-serialization-proxy/cmd/proxy-server/domain"
	"github.com/mcon/pact-serialization-proxy/cmd/proxy-server/serialization"
	"github.com/mkideal/cli"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"io/ioutil"

	"github.com/mcon/pact-serialization-proxy/cmd/proxy-server/controllers"
	"github.com/stretchr/testify/assert"
)

type fakeHttpClient struct {
	t *testing.T
	// For now, assume that a single URL will only be used with one HTTP method type
	pathToResponse map[string]*http.Response

	err             error
	endpointsCalled []string
	lastRequest     *http.Request
}

func (client *fakeHttpClient) Do(req *http.Request) (*http.Response, error) {
	client.endpointsCalled = append(client.endpointsCalled, req.URL.Path)
	if client.err != nil {
		return nil, client.err
	}
	client.lastRequest = req

	// Response must be consistent with the request that produced it
	response, success := client.pathToResponse[req.URL.Path]

	// Populate the correct content length here
	materializedResponse, err := ioutil.ReadAll(response.Body)
	response.ContentLength = int64(len(materializedResponse))
	response.Body = ioutil.NopCloser(bytes.NewReader(materializedResponse))
	if err != nil {
		panic(err)
	}
	if !success {
		panic(*req.URL)
	}

	return response, nil
}

func (client *fakeHttpClient) ResetCallsOccurred() {
	// Don't reset the pathToResponse map
	client.err = nil
	client.endpointsCalled = make([]string, 0)
}

func performRequest(r http.Handler, method, url string, contents io.Reader, headers http.Header) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, url, contents)
	req.Header = headers
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TODO: Should test default headers and passing headers through separately.
// TODO: Should test adding PUT, POST requests as well as GET

// Just check that the /interactions/verification endpoint that the Pact framework relies on is called on the ruby Core
func TestMainVerificationSuccessPassedThrough(t *testing.T) {
	fakeRubyCore := &fakeHttpClient{
		t:               t,
		endpointsCalled: make([]string, 0),
		pathToResponse: map[string]*http.Response{
			"//interactions/verification": {
				Body:    ioutil.NopCloser(strings.NewReader("")),
				Request: &http.Request{},
			}},
	}
	fakeDeps := &controllers.Dependencies{
		HttpClient: fakeRubyCore,
		CliArgs: &domain.CliArgs{
			Helper:      cli.Helper{},
			Verificaion: false,
			PactDir:     "",
			LogDir:      "",
			Port:        0,
			Host:        "",
			RubyCoreUrl: "http://localhost:1234/",
		},
		InteractionLookup: domain.CreateEmptyInteractionLookup(),
	}
	router := SetupRouter(fakeDeps)

	// Perform a GET request with that handler.
	response := performRequest(router, "GET", "/interactions/verification", strings.NewReader(""), http.Header{})

	// Assert we would call the corresponding point on the ruby core
	assert.Equal(t, []string{"//interactions/verification"}, fakeRubyCore.endpointsCalled)
	assert.Equal(t, make(http.Header), response.Header()) // No headers are set
	assert.Equal(t, http.StatusOK, response.Code)
}

// Just a sample proto encoding
func getFileDescriptorSetForUserType() *descriptor.FileDescriptorSet {
	base64FileDescriptor := "Cg5jb250cmFjdC5wcm90bxIIY29udHJhY3QiMQoGUGVyc29uEgwKBG5hbWUYASABKAkSCgoCaWQYAiABKAUSDQoFZW1haWwYAyABKAliBnByb3RvMw=="

	fileDescriptor := descriptor.FileDescriptorProto{}
	bytes, err := base64.StdEncoding.DecodeString(base64FileDescriptor)
	if err != nil {
		panic("Unable to decode base64 string to []byte")
	}
	err = proto.Unmarshal(bytes, &fileDescriptor)
	if err != nil {
		panic("Unable to unmarshal FileDescriptor")
	}

	// The Pact client library will send over the FileDescriptorSet (dependencies of Proto message definitions may span many files in general)
	return &descriptor.FileDescriptorSet{
		File: []*descriptor.FileDescriptorProto{&fileDescriptor},
	}
}

func getMessageDescriptorForUserType(set *descriptor.FileDescriptorSet) *desc.MessageDescriptor {
	fileDescriptor, err := desc.CreateFileDescriptorFromSet(set)
	if err != nil {
		panic("Unable to create FileDescriptor from FileDescriptorSet")
	}

	messages := fileDescriptor.GetMessageTypes()
	for _, msg := range messages {
		if msg.GetName() == "Person" {
			return msg
		}
	}
	return nil
}

func decodeUserMessage(data []byte) *dynamic.Message {
	fds := getFileDescriptorSetForUserType()
	messageDescriptor := getMessageDescriptorForUserType(fds)

	message := dynamic.NewMessage(messageDescriptor)
	err := message.Unmarshal(data)
	if err != nil {
		panic("Unable to unmarshall dynamic message")
	}

	return message
}

func getfloat64RepresenationOfProtoMessage(desc proto.Message) []float64 {
	encoded, err := proto.Marshal(desc)
	if err != nil {
		panic(desc)
	}

	// Probably should work whether a better way exists to get the encoding from JSON into the code
	fileDescriptorSetFloats := make([]float64, 0, 100000)
	for _, child := range encoded {
		fileDescriptorSetFloats = append(fileDescriptorSetFloats, float64(child))
	}
	return fileDescriptorSetFloats
}

func getStandardUserJsonString() *serialization.PactRequestBody {
	return serialization.CreatePactRequestBody(`{"name":"Joe Bloggs","email":"joe.bloggs@foobarmail.com"}`)
}

func getStandardProtobufInteraction() serialization.ProviderServiceInteraction {
	// No need to test serialization of types, as this is tested in pact_serialization_test.go
	fdsForUser := getFileDescriptorSetForUserType()
	encodingForInteractionRegistration := getfloat64RepresenationOfProtoMessage(fdsForUser)
	return serialization.ProviderServiceInteraction{
		Description:   "Successfully get a set of users",
		ProviderState: "Success state",
		Request: serialization.ProviderServiceRequest{
			Method:   "GET",
			Path:     "/users",
			Query:    "?type=verified",
			Encoding: serialization.SerializationEncoding{}, // No request body for GET
			Headers:  map[string]string{"Content-type": "application/octet-stream", "Arbitrary-header": "some-value"},
		},
		Response: serialization.ProviderServiceResponse{
			Status: 200,
			Encoding: serialization.SerializationEncoding{
				Type: "protobuf",
				Description: &serialization.ProtobufEncodingDescription{
					MessageName:       "Person",
					FileDescriptorSet: encodingForInteractionRegistration,
				},
			},
			Headers: nil,
			Body:    getStandardUserJsonString(),
		},
	}
}

func addStandardProtobufInteraction(
	t *testing.T, router *gin.Engine, fakeDeps *controllers.Dependencies, fakeRubyCore *fakeHttpClient) domain.UniqueInteractionIdentifier {
	interactionFromClient := getStandardProtobufInteraction()

	marshalledInteraction, err := json.Marshal(interactionFromClient)
	postRequestBodyReader := bytes.NewReader(marshalledInteraction)
	if err != nil {
		panic(err)
	}
	response := performRequest(router, "POST", "/interactions", postRequestBodyReader, http.Header{})

	// Assert we would call the corresponding point on the ruby core
	assert.Equal(t, []string{"//interactions"}, fakeRubyCore.endpointsCalled)
	// Content-type should be JSON now, as that's all the core can read
	assert.Equal(t, http.Header{}, response.Header())
	assert.Equal(t, http.StatusOK, response.Code)

	// Assert that we have an entry in our internal interactions map as expected
	interactionLookupKey := domain.CreateUniqueInteractionIdentifier(
		interactionFromClient.Request.Method, interactionFromClient.Request.Path, interactionFromClient.Request.Query)
	_, atteptedLookupSuccess := fakeDeps.InteractionLookup.Get(interactionLookupKey)
	assert.True(t, atteptedLookupSuccess, "Unable to look up expected interaction in global map")

	return interactionLookupKey
}

func getStandardJsonInteraction() serialization.ProviderServiceInteraction {
	return serialization.ProviderServiceInteraction{
		Description:   "Successfully get a set of users",
		ProviderState: "Success state",
		Request: serialization.ProviderServiceRequest{
			Method:   "GET",
			Path:     "/users-json-endpoint",
			Query:    "?type=verified",
			Encoding: serialization.SerializationEncoding{}, // No request body for GET
			Headers:  map[string]string{"Content-type": "application/json", "Arbitrary-header": "some-value"},
		},
		Response: serialization.ProviderServiceResponse{
			Status: 200,
			Encoding: serialization.SerializationEncoding{
				Type:        "",
				Description: nil,
			},
			Headers: nil,
			Body:    getStandardUserJsonString(),
		},
	}
}

func addStandardJsonInteraction(
	t *testing.T, router *gin.Engine, fakeDeps *controllers.Dependencies, fakeRubyCore *fakeHttpClient) domain.UniqueInteractionIdentifier {
	interactionFromClient := getStandardJsonInteraction()

	marshalledInteraction, err := json.Marshal(interactionFromClient)
	postRequestBodyReader := bytes.NewReader(marshalledInteraction)
	if err != nil {
		assert.Fail(t, "Unable to serialize message %T", interactionFromClient)
	}
	response := performRequest(router, "POST", "/interactions", postRequestBodyReader, http.Header{})

	// Assert we would call the corresponding point on the ruby core
	assert.Equal(t, []string{"//interactions"}, fakeRubyCore.endpointsCalled)
	// Content-type should be JSON now, as that's all the core can read
	assert.Equal(t, http.Header{}, response.Header())
	assert.Equal(t, http.StatusOK, response.Code)

	// Assert that we have an entry in our internal interactions map as expected
	interactionLookupKey := domain.CreateUniqueInteractionIdentifier(
		interactionFromClient.Request.Method, interactionFromClient.Request.Path, interactionFromClient.Request.Query)
	_, atteptedLookupSuccess := fakeDeps.InteractionLookup.Get(interactionLookupKey)
	assert.True(t, atteptedLookupSuccess, "Unable to look up expected interaction in global map")

	return interactionLookupKey
}

func checkRequestForUserProto(
	t *testing.T, router *gin.Engine, fakeDeps *controllers.Dependencies, fakeRubyCore *fakeHttpClient) {
	//assert.Equal(t, http.Header{"Content-type": {"application/json"}, "Arbitrary-header": {"some-value"}}, response.Header())
	headers := http.Header{"Content-type": {"application/octet-stream"}, "Arbitrary-header": {"some-value"}}
	response := performRequest(router, "GET", "/users?type=verified", strings.NewReader(""), headers)
	decodedMessage := decodeUserMessage(response.Body.Bytes())

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, http.Header{"Content-Type": {"application/octet-stream"}, "Content-Length": {"39"}}, response.Header())
	assert.Equal(t, "Joe Bloggs", decodedMessage.GetFieldByName("name"))
	assert.Equal(t, "joe.bloggs@foobarmail.com", decodedMessage.GetFieldByName("email"))

	assert.Equal(t, []string{"//users"}, fakeRubyCore.endpointsCalled)
	// TODO: Add test which verifies that the correct headers are sent to the Ruby core
}

func checkRequestForUserJson(
	t *testing.T, router *gin.Engine, fakeDeps *controllers.Dependencies, fakeRubyCore *fakeHttpClient) {
	headers := http.Header{"Content-type": {"application/json"}, "Arbitrary-header": {"some-value"}}
	response := performRequest(router, "GET", "/users-json-endpoint?type=verified", strings.NewReader(""), headers)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, http.Header{"Content-Type": {"application/json"}, "Content-Length": {"60"}}, response.Header())
	assert.Equal(t, "{\"name\": \"Joe Bloggs\", \"email\": \"joe.bloggs@foobarmail.com\"}", response.Body.String())

	assert.Equal(t, []string{"//users-json-endpoint"}, fakeRubyCore.endpointsCalled)
	// TODO: Add test which verifies that the correct headers are sent to the Ruby core
}

func getSamplePactContractDto(includeSerialization bool) serialization.PactContract {
	contract := serialization.PactContract{
		Consumer:     "consumer",
		Provider:     "provider",
		Interactions: []serialization.ProviderServiceInteraction{getStandardJsonInteraction(), getStandardProtobufInteraction()},
		Metadata:     serialization.PactContractMetadata{PactSpecificationVersion: "2.0.0"},
	}

	// Contract from Ruby core won't have this encoding information
	if !includeSerialization {
		for idx, _ := range contract.Interactions {
			contract.Interactions[idx].Request.Encoding = serialization.SerializationEncoding{}
			contract.Interactions[idx].Response.Encoding = serialization.SerializationEncoding{}
		}
	}
	return contract
}

func getSamplePactContract(includeSerialization bool) string {
	contract := getSamplePactContractDto(includeSerialization)

	contractBytes, err := json.Marshal(contract)
	if err != nil {
		panic(err)
	}

	return string(contractBytes)
}

func checkPactContractCreationRequest(
	t *testing.T, router *gin.Engine, fakeDeps *controllers.Dependencies, fakeRubyCore *fakeHttpClient) {
	headers := http.Header{"Content-type": {"application/json"}}
	response := performRequest(router, "POST", "/pact", strings.NewReader(""), headers)

	assert.Equal(t, http.StatusOK, response.Code)

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}
	contract := serialization.PactContract{}
	err = json.Unmarshal(data, &contract)
	if err != nil {
		panic(err)
	}

	// Contract should match what we'd expect from the interactions used during the test
	assert.Equal(t, getSamplePactContractDto(true), contract)
	assert.Equal(t, []string{"//pact"}, fakeRubyCore.endpointsCalled)
}

func TestInteractionsAddedSuccessfully(t *testing.T) {
	// Set up as if we're creating the contract as the consumer with the Ruby core only returning 200
	fakeRubyCore := &fakeHttpClient{
		t:               t,
		endpointsCalled: make([]string, 0),
		pathToResponse: map[string]*http.Response{
			"//interactions": {
				Body: ioutil.NopCloser(strings.NewReader("")),
			},
		},
	}
	fakeDeps := &controllers.Dependencies{
		HttpClient: fakeRubyCore,
		CliArgs: &domain.CliArgs{
			Helper:      cli.Helper{},
			Verificaion: false,
			PactDir:     "",
			LogDir:      "",
			Port:        0,
			Host:        "",
			RubyCoreUrl: "http://localhost:1234/",
		},
		InteractionLookup: domain.CreateEmptyInteractionLookup(),
	}
	router := SetupRouter(fakeDeps)

	// Run the test
	addStandardProtobufInteraction(t, router, fakeDeps, fakeRubyCore)
	fakeRubyCore.ResetCallsOccurred()
	addStandardJsonInteraction(t, router, fakeDeps, fakeRubyCore)
}

func TestConsumerJsonOrProtoRequestsPassedThrough(t *testing.T) {
	// Set up as if we're creating the contract as the consumer with the Ruby core only returning 200
	fakeRubyCore := &fakeHttpClient{
		t:               t,
		endpointsCalled: make([]string, 0),
		pathToResponse: map[string]*http.Response{
			"//interactions": {
				Body:       ioutil.NopCloser(strings.NewReader("")),
				StatusCode: 200,
			},
			"//users": {
				Body:       ioutil.NopCloser(strings.NewReader("{\"name\": \"Joe Bloggs\", \"email\": \"joe.bloggs@foobarmail.com\"}")),
				StatusCode: 200,
			},
			"//users-json-endpoint": {
				Body:       ioutil.NopCloser(strings.NewReader("{\"name\": \"Joe Bloggs\", \"email\": \"joe.bloggs@foobarmail.com\"}")),
				StatusCode: 200,
			},
		},
	}
	fakeDeps := &controllers.Dependencies{
		HttpClient: fakeRubyCore,
		CliArgs: &domain.CliArgs{
			Helper:      cli.Helper{},
			Verificaion: false,
			PactDir:     "",
			LogDir:      "",
			Port:        0,
			Host:        "",
			RubyCoreUrl: "http://localhost:1234/",
		},
		InteractionLookup: domain.CreateEmptyInteractionLookup(),
	}
	router := SetupRouter(fakeDeps)

	// Set up expected interactions
	addStandardProtobufInteraction(t, router, fakeDeps, fakeRubyCore)
	fakeRubyCore.ResetCallsOccurred()
	addStandardJsonInteraction(t, router, fakeDeps, fakeRubyCore)
	fakeRubyCore.ResetCallsOccurred()

	// Run test and assertions for passing through interactions
	checkRequestForUserProto(t, router, fakeDeps, fakeRubyCore)
	fakeRubyCore.ResetCallsOccurred()
	checkRequestForUserJson(t, router, fakeDeps, fakeRubyCore)
}

func TestMainPactContractCreationSuccess(t *testing.T) {
	// Set up as if we're creating the contract as the consumer with the Ruby core only returning 200
	fakeRubyCore := &fakeHttpClient{
		t:               t,
		endpointsCalled: make([]string, 0),
		pathToResponse: map[string]*http.Response{
			"//interactions": {
				Body:       ioutil.NopCloser(strings.NewReader("")),
				StatusCode: 200,
			},
			"//pact": {
				Body:       ioutil.NopCloser(strings.NewReader(getSamplePactContract(false))),
				StatusCode: 200,
				Request:    &http.Request{},
			},
		},
	}
	fakeDeps := &controllers.Dependencies{
		HttpClient: fakeRubyCore,
		CliArgs: &domain.CliArgs{
			Helper:      cli.Helper{},
			Verificaion: false,
			PactDir:     "",
			LogDir:      "",
			Port:        0,
			Host:        "",
			RubyCoreUrl: "http://localhost:1234/",
		},
		InteractionLookup: domain.CreateEmptyInteractionLookup(),
		FileWriter:        func(filename string, data []byte, perm os.FileMode) error { return nil }, // TODO: Should confirm this is called
	}
	router := SetupRouter(fakeDeps)

	// Set up expected interactions
	addStandardProtobufInteraction(t, router, fakeDeps, fakeRubyCore)
	fakeRubyCore.ResetCallsOccurred()
	addStandardJsonInteraction(t, router, fakeDeps, fakeRubyCore)
	fakeRubyCore.ResetCallsOccurred()

	// No need to actually exercise the interactions: only the Ruby core actually cares about that
	checkPactContractCreationRequest(t, router, fakeDeps, fakeRubyCore)
}

func TestPactCoreErrorsPassedThrough(t *testing.T) {
	// Set up the Ruby core to return an error
	fakeRubyCore := &fakeHttpClient{
		t:               t,
		endpointsCalled: make([]string, 0),
		pathToResponse: map[string]*http.Response{
			"//interactions/verification": {
				Body:       ioutil.NopCloser(strings.NewReader("")),
				StatusCode: 500,
			},
		},
	}
	fakeDeps := &controllers.Dependencies{
		HttpClient: fakeRubyCore,
		CliArgs: &domain.CliArgs{
			Helper:      cli.Helper{},
			Verificaion: false,
			PactDir:     "",
			LogDir:      "",
			Port:        0,
			Host:        "",
			RubyCoreUrl: "http://localhost:1234/",
		},
	}
	router := SetupRouter(fakeDeps)

	// Perform a GET request with that handler.
	response := performRequest(router, "GET", "/interactions/verification", strings.NewReader(""), http.Header{})

	// Assert we would call the corresponding point on the ruby core
	assert.Equal(t, []string{"//interactions/verification"}, fakeRubyCore.endpointsCalled)
	assert.Equal(t, make(http.Header), response.Header()) // No headers are set
	assert.Equal(t, http.StatusInternalServerError, response.Code)
}

func TestMainVerificationSerializationError(t *testing.T) {
	// Check that a sensible error is returned and application state remains sane in the case that the serialization
	// information isn't actually correct when verifying the Pact contract as a Provider.
}

func TestConsumerProtobufSerializationError(t *testing.T) {
	// Check that a sensible error is returned and application state remains sane in the case that the serialization
	// information isn't actually correct when creating the Pact contract as a Consumer.
}

func TestVerificationSuccess(t *testing.T) {
	// Check that we can verify pact contracts as expected
}

func TestProtobufViolatesContractDueToFieldIdChanges(t *testing.T) {
}

func TestConsumerProtobufRequestOnUnknownEndpoint(t *testing.T) {
}
