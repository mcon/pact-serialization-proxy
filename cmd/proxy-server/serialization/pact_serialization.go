package serialization

type ProtobufEncodingDescription struct {
	MessageName       string
	FileDescriptorSet []float64
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

type ProviderServiceRequest struct {
	Method   string
	Path     string
	Query    string
	Encoding SerializationEncoding
	Headers  map[string]string
	Body     *PactRequestBody `json:"body,omitempty"`
}

type ProviderServiceResponse struct {
	Status   int
	Encoding SerializationEncoding
	Headers  map[string]string
	Body     *PactRequestBody `json:"body,omitempty"`
}

type ProviderServiceInteraction struct {
	Description   string
	ProviderState string
	Request       ProviderServiceRequest
	Response      ProviderServiceResponse
}

type PactContractMetadata struct {
	PactSpecificationVersion string
}

type PactContract struct {
	Consumer     string
	Provider     string
	Interactions []ProviderServiceInteraction
	Metadata     PactContractMetadata
}
