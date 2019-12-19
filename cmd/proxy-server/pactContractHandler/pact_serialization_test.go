package pactContractHandler

import (
	"encoding/json"
	"github.com/mcon/pact-serialization-proxy/cmd/proxy-server/serialization"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TODO: Add Description and ProviderState to this test
func TestCheckSerializationMatchesPactCore(t *testing.T) {
	expected_data_structure := &serialization.ProviderServiceInteraction{
		Request: serialization.ProviderServiceRequest{
			Method: "POST",
			Path:   "foo/bar",
			Query:  "?number=1",
			Encoding: serialization.SerializationEncoding{
				Type: "protobuf",
				Description: &serialization.ProtobufEncodingDescription{
					MessageName:       "BarRequestMessage",
					FileDescriptorSet: []float64{1, 2, 3},
				},
			},
			Headers: map[string]string{"Accept": "application/octet-stream"},
			Body:    "{\"Key\" : \"Value\"}",
		},
		Response: serialization.ProviderServiceResponse{
			Status: 200,
			Encoding: serialization.SerializationEncoding{
				Type: "protobuf",
				Description: &serialization.ProtobufEncodingDescription{
					MessageName:       "BarResponseMessage",
					FileDescriptorSet: []float64{4, 5, 6},
				},
			},
			Headers: map[string]string{"Accept": "application/octet-stream"},
			Body:    "",
		},
	}
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
        "Body": "{\"Key\" : \"Value\"}"
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
        },
        "Body": ""
    }
}`
	var unmarshalled_interaction = new(serialization.ProviderServiceInteraction)
	e := json.Unmarshal([]byte(json_under_test), unmarshalled_interaction)
	assert.Nil(t, e, "Unmarshaling JSON should succeed")

	assert.Equal(t, expected_data_structure, unmarshalled_interaction, "Expected DTO to serialize into correct form")
}
