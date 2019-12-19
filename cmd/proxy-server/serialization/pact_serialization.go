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

type ProviderServiceRequest struct {
	Method   string
	Path     string
	Query    string
	Encoding SerializationEncoding
	Headers  map[string]string
	Body     string
}

type ProviderServiceResponse struct {
	Status   int32
	Encoding SerializationEncoding
	Headers  map[string]string
	Body     string
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
