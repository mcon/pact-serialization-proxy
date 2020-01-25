package serialization

import "encoding/json"

type ProtobufEncodingDescription struct {
	MessageName       string    `json:"messageName"`
	FileDescriptorSet []float64 `json:"fileDescriptorSet"`
}

// In general, the contents of `Description` might be different based on the `Type` of the encoding:
// for now only protobuf is supported, so don't worry about that.
type SerializationEncoding struct {
	Type        string
	Description *ProtobufEncodingDescription
}

type PactRequestBody struct {
	data string
}

func CreatePactRequestBody(data string) *PactRequestBody {
	if data == "" {
		return nil
	}
	return &PactRequestBody{data: data}
}

func (body *PactRequestBody) MarshalJSON() ([]byte, error) {
	return []byte(body.data), nil
}

func (body *PactRequestBody) UnmarshalJSON(data []byte) error {
	body.data = string(data)
	return nil
}

type RegexMatcherDescription struct {
	JsonClass string `json:"json_class"`
	o         int32  // Pact-core specific fields not relevant to the serialization proxy
	s         string
}

type RexexMatcher struct {
	ExamplePath string                  `json:"generate"`
	Matcher     RegexMatcherDescription `json:"matcher"`
}

type RegexedString struct {
	JsonClass string        `json:"json_class,omitempty"`
	Data      *RexexMatcher `json:"data,omitempty"`
}

// TODO: The 'WithRegex' case can only apply in the body of posting to '/interactions', sadly matching rules in
// the pact contract follow a different form
type PossiblyRegexedString struct {
	NoRegex   string
	WithRegex *RegexedString
}

func (x *PossiblyRegexedString) GetString() string {
	if x.WithRegex == nil {
		return x.NoRegex
	}
	return x.WithRegex.Data.ExamplePath
}

func (x *PossiblyRegexedString) MarshalJSON() ([]byte, error) {
	if x.WithRegex != nil {
		return json.Marshal(x.WithRegex)
	}
	return []byte("\"" + x.NoRegex + "\""), nil
}

func (x *PossiblyRegexedString) UnmarshalJSON(data []byte) error {
	err := json.Unmarshal(data, &x.WithRegex)

	// Assume that if we don't have a regex, then there's just a raw string
	if err != nil {
		x.NoRegex = string(data[1 : len(data)-1]) // Remove quotes around path
		x.WithRegex = nil
	}
	return nil
}

type ProviderServiceRequest struct {
	Method        string                 `json:"method"`
	Path          *PossiblyRegexedString `json:"path"`
	Query         *PossiblyRegexedString `json:"query,omitempty"`
	Encoding      *SerializationEncoding `json:"encoding,omitempty"`
	Headers       interface{}            `json:"headers,omitempty"`
	Body          *PactRequestBody       `json:"body,omitempty"`
	MatchingRules interface{}            `json:"matchingRules,omitempty"` // Only applies to pact contract
}

type ProviderServiceResponse struct {
	Status        int                    `json:"status"`
	Encoding      *SerializationEncoding `json:"encoding,omitempty"`
	Headers       interface{}            `json:"headers,omitempty"`
	Body          *PactRequestBody       `json:"body,omitempty"`
	MatchingRules interface{}            `json:"matchingRules,omitempty"` // Only applies to pact contract
}

type ProviderServiceInteraction struct {
	Description   string                  `json:"description"`
	ProviderState string                  `json:"providerState"`
	Request       ProviderServiceRequest  `json:"request"`
	Response      ProviderServiceResponse `json:"response"`
}

type PactSpecificationDescription struct {
	Version string `json:"version"`
}

type PactContractMetadata struct {
	PactSpecification PactSpecificationDescription `json:"pactSpecification"`
}

type ConsumerOrProvider struct {
	Name string `json:"name"`
}

type PactContract struct {
	Consumer     ConsumerOrProvider           `json:"consumer"`
	Provider     ConsumerOrProvider           `json:"provider"`
	Interactions []ProviderServiceInteraction `json:"interactions"`
	Metadata     PactContractMetadata         `json:"metadata"`
}
