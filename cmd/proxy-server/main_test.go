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
	"net/url"
	"strings"
	"testing"

	"io/ioutil"

	"github.com/mcon/pact-serialization-proxy/cmd/proxy-server/controllers"
	"github.com/stretchr/testify/assert"
)

type fakeHttpClient struct {
	t *testing.T
	// For now, assume that a single URL will only be used with one HTTP method type
	cannedResponse map[url.URL]*http.Response

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
	client.cannedResponse[*req.URL].Request = req

	return client.cannedResponse[*req.URL], nil
}

func (client *fakeHttpClient) Reset() {
	client.cannedResponse = make(map[url.URL]*http.Response)
	client.err = nil
	client.endpointsCalled = make([]string, 0)
}

func performRequest(r http.Handler, method, path string, contents io.Reader) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, contents)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TODO: Should test default headers and passing headers through separately.
// TODO: Should test adding PUT, POST requests as well as GET

// Just check that the /interactions/verification endpoint that the Pact framework relies on is called on the ruby Core
func TestMainVerificationSuccessPassedThrough(t *testing.T) {
	verificationUrl, err := url.ParseRequestURI("http://localhost:1234/interactions/verification")
	if err != nil {
		panic(err)
	}
	fakeRubyCore := &fakeHttpClient{
		t:               t,
		endpointsCalled: make([]string, 0),
		cannedResponse: map[url.URL]*http.Response{
			*verificationUrl: {
				Body: ioutil.NopCloser(strings.NewReader("")),
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
	response := performRequest(router, "GET", "/interactions/verification", strings.NewReader(""))

	// Assert we would call the corresponding point on the ruby core
	assert.Equal(t, []string{"/interactions/verification"}, fakeRubyCore.endpointsCalled)
	assert.Equal(t, make(http.Header), response.Header()) // No headers are set
	assert.Equal(t, http.StatusOK, response.Code)
}

// Just a sample proto encoding
func getFileDescriptorSetForUserType(t *testing.T) *descriptor.FileDescriptorSet {
	base64FileDescriptor := "Cg5jb250cmFjdC5wcm90bxIIY29udHJhY3QiMQoGUGVyc29uEgwKBG5hbWUYASABKAkSCgoCaWQYAiABKAUSDQoFZW1haWwYAyABKAliBnByb3RvMw=="

	fileDescriptor := descriptor.FileDescriptorProto{}
	bytes, err := base64.StdEncoding.DecodeString(base64FileDescriptor)
	if err != nil {
		assert.Fail(t, "Unable to decode base64 string to []byte")
	}
	err = proto.Unmarshal(bytes, &fileDescriptor)
	if err != nil {
		assert.Fail(t, "Unable to unmarshal FileDescriptor")
	}

	// The Pact client library will send over the FileDescriptorSet (dependencies of Proto message definitions may span many files in general)
	return &descriptor.FileDescriptorSet{
		File: []*descriptor.FileDescriptorProto{&fileDescriptor},
	}
}

func getMessageDescriptorForUserType(t *testing.T, set *descriptor.FileDescriptorSet) *desc.MessageDescriptor {
	fileDescriptor, err := desc.CreateFileDescriptorFromSet(set)
	if err != nil {
		assert.Fail(t, "Unable to create FileDescriptor from FileDescriptorSet")
	}

	messages := fileDescriptor.GetMessageTypes()
	for _, msg := range messages {
		if msg.GetName() == "Person" {
			return msg
		}
	}
	return nil
}

func getSampleEncodedDataForUser(t *testing.T) string {
	fds := getFileDescriptorSetForUserType(t)
	messageDescriptor := getMessageDescriptorForUserType(t, fds)

	message := dynamic.NewMessage(messageDescriptor)
	message.SetFieldByName("Name", "Joe Bloggs")
	message.SetFieldByName("Email", "joe.bloggs@foobarmail.com")

	marshalledBytes, err := proto.Marshal(message)
	if err != nil {
		assert.Fail(t, "Unable to marshall dynamic message")
	}

	return string(marshalledBytes)
}

func getfloat64RepresenationOfProtoMessage(t *testing.T, desc proto.Message) []float64 {
	encoded, err := proto.Marshal(desc)
	if err != nil {
		assert.Fail(t, "Unable to serialize proto message %T", desc)
	}

	// Probably should work whether a better way exists to get the encoding from JSON into the code
	fileDescriptorSetFloats := make([]float64, 0, 100000)
	for _, child := range encoded {
		fileDescriptorSetFloats = append(fileDescriptorSetFloats, float64(child))
	}
	return fileDescriptorSetFloats
}

func addStandardProtobufInteraction(
	t *testing.T, router *gin.Engine, fakeDeps *controllers.Dependencies, fakeRubyCore *fakeHttpClient) domain.UniqueInteractionIdentifier {
	// No need to test serialization of types, as this is tested in pact_serialization_test.go
	httpMethod := "GET"
	httpPath := "/users"
	httpQuery := "?type=verified"

	fdsForUser := getFileDescriptorSetForUserType(t)
	encodingForInteractionRegistration := getfloat64RepresenationOfProtoMessage(t, fdsForUser)
	interactionFromClient := serialization.ProviderServiceInteraction{
		Description:   "Successfully get a set of users",
		ProviderState: "Success state",
		Request: serialization.ProviderServiceRequest{
			Method:   httpMethod,
			Path:     httpPath,
			Query:    httpQuery,
			Encoding: serialization.SerializationEncoding{}, // No request body for GET
			Headers:  map[string]string{"Content-type": "application/octet-stream", "Arbitrary-header": "some-value"},
			Body:     "",
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
			Body:    "{\"Name\": \"Joe Bloggs\", \"Email\": \"joe.bloggs@foobarmail.com\"}",
		},
	}

	marshalledInteraction, err := json.Marshal(interactionFromClient)
	postRequestBodyReader := bytes.NewReader(marshalledInteraction)
	if err != nil {
		assert.Fail(t, "Unable to serialize message %T", interactionFromClient)
	}
	response := performRequest(router, "POST", "/interactions", postRequestBodyReader)

	// Assert we would call the corresponding point on the ruby core
	assert.Equal(t, []string{"/interactions"}, fakeRubyCore.endpointsCalled)
	// Content-type should be JSON now, as that's all the core can read
	assert.Equal(t, http.Header{"Content-type": {"application/json"}, "Arbitrary-header": {"some-value"}}, response.Header())
	assert.Equal(t, http.StatusOK, response.Code)

	// Assert that we have an entry in our internal interactions map as expected
	interactionLookupKey := domain.CreateUniqueInteractionIdentifier(httpMethod, httpPath, httpQuery)
	_, atteptedLookupSuccess := fakeDeps.InteractionLookup.Get(interactionLookupKey)
	assert.True(t, atteptedLookupSuccess, "Unable to look up expected interaction in global map")

	return interactionLookupKey
}

func addStandardJsonInteraction(
	t *testing.T, router *gin.Engine, fakeDeps *controllers.Dependencies, fakeRubyCore *fakeHttpClient) domain.UniqueInteractionIdentifier {
	httpMethod := "GET"
	httpPath := "/users-json-endpoint"
	httpQuery := "?type=verified"

	// No need to test serialization of types, as this is tested in pact_serialization_test.go
	interactionFromClient := serialization.ProviderServiceInteraction{
		Description:   "Successfully get a set of users",
		ProviderState: "Success state",
		Request: serialization.ProviderServiceRequest{
			Method:   httpMethod,
			Path:     httpPath,
			Query:    httpQuery,
			Encoding: serialization.SerializationEncoding{}, // No request body for GET
			Headers:  map[string]string{"Content-type": "application/json", "Arbitrary-header": "some-value"},
			Body:     "",
		},
		Response: serialization.ProviderServiceResponse{
			Status: 200,
			Encoding: serialization.SerializationEncoding{
				Type:        "protobuf",
				Description: nil,
			},
			Headers: nil,
			Body:    getSampleEncodedDataForUser(t),
		},
	}

	marshalledInteraction, err := json.Marshal(interactionFromClient)
	postRequestBodyReader := bytes.NewReader(marshalledInteraction)
	if err != nil {
		assert.Fail(t, "Unable to serialize message %T", interactionFromClient)
	}
	response := performRequest(router, "POST", "/interactions", postRequestBodyReader)

	// Assert we would call the corresponding point on the ruby core
	assert.Equal(t, []string{"/interactions"}, fakeRubyCore.endpointsCalled)
	// Content-type should be JSON now, as that's all the core can read
	assert.Equal(t, http.Header{"Content-type": {"application/json"}, "Arbitrary-header": {"some-value"}}, response.Header())
	assert.Equal(t, http.StatusOK, response.Code)

	// Assert that we have an entry in our internal interactions map as expected
	interactionLookupKey := domain.CreateUniqueInteractionIdentifier(httpMethod, httpPath, httpQuery)
	_, atteptedLookupSuccess := fakeDeps.InteractionLookup.Get(interactionLookupKey)
	assert.True(t, atteptedLookupSuccess, "Unable to look up expected interaction in global map")

	return interactionLookupKey
}

func checkRequestForUserProto(
	t *testing.T, router *gin.Engine, fakeDeps *controllers.Dependencies, fakeRubyCore *fakeHttpClient) {
	getSampleEncodedDataForUser
	panic("")
}

func checkRequestForUserJson(
	t *testing.T, router *gin.Engine, fakeDeps *controllers.Dependencies, fakeRubyCore *fakeHttpClient) {
	panic("")
}

func TestInteractionsAddedSuccessfully(t *testing.T) {
	// Set up as if we're creating the contract as the consumer with the Ruby core only returning 200
	interactionsUrl, err := url.ParseRequestURI("http://localhost:1234/interactions")
	if err != nil {
		panic(err)
	}
	fakeRubyCore := &fakeHttpClient{
		t:               t,
		endpointsCalled: make([]string, 0),
		cannedResponse: map[url.URL]*http.Response{
			*interactionsUrl: {
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
	addStandardJsonInteraction(t, router, fakeDeps, fakeRubyCore)
}

func TestConsumerJsonOrProtoRequestsPassedThrough(t *testing.T) {
	// Set up as if we're creating the contract as the consumer with the Ruby core only returning 200
	usersUrl, err := url.ParseRequestURI("http://localhost:1234/users")
	usersJsonUrl, err := url.ParseRequestURI("http://localhost:1234/users-json-endpoint")
	interactionsUrl, err := url.ParseRequestURI("http://localhost:1234/interactions")
	if err != nil {
		panic(err)
	}
	fakeRubyCore := &fakeHttpClient{
		t:               t,
		endpointsCalled: make([]string, 0),
		cannedResponse: map[url.URL]*http.Response{
			*interactionsUrl: {
				Body: ioutil.NopCloser(strings.NewReader("")),
			},
			*usersUrl: {
				Body: ioutil.NopCloser(strings.NewReader("{\"Name\": \"Joe Bloggs\", \"Email\": \"joe.bloggs@foobarmail.com\"}")),
			},
			*usersJsonUrl: {
				Body: ioutil.NopCloser(strings.NewReader("{\"Name\": \"Joe Bloggs\", \"Email\": \"joe.bloggs@foobarmail.com\"}")),
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
	addStandardJsonInteraction(t, router, fakeDeps, fakeRubyCore)

	// Run test and assertions for passing through interactions
	checkRequestForUserProto(t, router, fakeDeps, fakeRubyCore)
	checkRequestForUserJson(t, router, fakeDeps, fakeRubyCore)
}

func getSamplePactContract() string {
	panic("")
}

func checkPactContractCreationRequest(
	t *testing.T, router *gin.Engine, fakeDeps *controllers.Dependencies, fakeRubyCore *fakeHttpClient) {
	panic("")
}

func TestMainPactContractCreationSuccess(t *testing.T) {
	// Set up as if we're creating the contract as the consumer with the Ruby core only returning 200
	pactCreationUrl, err := url.ParseRequestURI("http://localhost:1234/pact")
	if err != nil {
		panic(err)
	}
	fakeRubyCore := &fakeHttpClient{
		t:               t,
		endpointsCalled: make([]string, 0),
		cannedResponse: map[url.URL]*http.Response{
			*pactCreationUrl: {
				Body:       ioutil.NopCloser(strings.NewReader(getSamplePactContract())),
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
	addStandardJsonInteraction(t, router, fakeDeps, fakeRubyCore)

	// No need to actually exercise the interactions: only the Ruby core actually cares about that
}

func TestPactCoreErrorsPassedThrough(t *testing.T) {
	// Set up the Ruby core to return an error
	verificationUrl, err := url.ParseRequestURI("http://localhost:1234/interactions/verification")
	if err != nil {
		panic(err)
	}
	fakeRubyCore := &fakeHttpClient{
		t:               t,
		endpointsCalled: make([]string, 0),
		cannedResponse: map[url.URL]*http.Response{
			*verificationUrl: {
				Body:       ioutil.NopCloser(strings.NewReader("")),
				StatusCode: 500,
			},
		},
	}
	fakeDeps := &controllers.Dependencies{
		HttpClient: fakeRubyCore,
	}
	router := SetupRouter(fakeDeps)

	// Perform a GET request with that handler.
	response := performRequest(router, "GET", "/interactions/verification", strings.NewReader(""))

	// Assert we would call the corresponding point on the ruby core
	assert.Equal(t, []string{"/interactions/verification"}, fakeRubyCore.endpointsCalled)
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

func TestConsumerProtobufRequestOnUnknownEndpoint(t *testing.T) {
}
