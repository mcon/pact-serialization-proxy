package serialization

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

var expectedDataStructure = &ProviderServiceInteraction{
	Request: ProviderServiceRequest{
		Method: "POST",
		Path:   "foo/bar",
		Query:  "?number=1",
		Encoding: SerializationEncoding{
			Type: "protobuf",
			Description: &ProtobufEncodingDescription{
				MessageName:       "BarRequestMessage",
				FileDescriptorSet: []float64{1, 2, 3},
			},
		},
		Headers: map[string]string{"Accept": "application/octet-stream"},
		Body:    CreatePactRequestBody("{\"Key\":\"Value\"}"),
	},
	Response: ProviderServiceResponse{
		Status: 200,
		Encoding: SerializationEncoding{
			Type: "protobuf",
			Description: &ProtobufEncodingDescription{
				MessageName:       "BarResponseMessage",
				FileDescriptorSet: []float64{4, 5, 6},
			},
		},
		Headers: map[string]string{"Accept": "application/octet-stream"},
		Body:    CreatePactRequestBody(""),
	},
}

// TODO: Add Description and ProviderState to this test
func TestCheckSerializationMatchesPactCore(t *testing.T) {
	json_under_test :=
		`{
    "Request": {
        "Method": "POST",
        "Path": "foo/bar",
        "Query": "?number=1",
        "Encoding": {
            "Type": "protobuf",
            "Description": {
                "MessageName": "BarRequestMessage",
                "FileDescriptorSet": [
                    1,
                    2,
                    3
                ]
            }
        },
        "Headers": {
            "Accept": "application/octet-stream"
        },
        "Body": {"Key":"Value"}
    },
    "Response": {
        "Status": 200,
        "Encoding": {
            "Type": "protobuf",
            "Description": {
                "MessageName": "BarResponseMessage",
                "FileDescriptorSet": [
                    4,
                    5,
                    6
                ]
            }
        },
        "Headers": {
            "Accept": "application/octet-stream"
        }
    }
}`
	var unmarshalledInteraction = new(ProviderServiceInteraction)
	e := json.Unmarshal([]byte(json_under_test), unmarshalledInteraction)
	assert.NoError(t, e, "Unmarshaling JSON should succeed")

	assert.Equal(t, expectedDataStructure, unmarshalledInteraction, "Expected DTO to serialize into correct form")
}

func TestSerializationRoundTrips(t *testing.T) {
	marshaled, err := json.Marshal(expectedDataStructure)
	assert.NoError(t, err, "Marshaling JSON should succeed")

	unmarshalledInteraction := new(ProviderServiceInteraction)
	err = json.Unmarshal(marshaled, unmarshalledInteraction)
	assert.NoError(t, err, "Unmarshaling JSON should succeed")

	assert.Equal(t, expectedDataStructure, unmarshalledInteraction, "Expected DTO to round-trip")
}
