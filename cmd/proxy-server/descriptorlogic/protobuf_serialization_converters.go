package descriptorlogic

import (
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
)

func JsonBytesToProtobufBytes(jsonBytes []byte, messageDescriptor *desc.MessageDescriptor) ([]byte, error) {
	protoMessage := dynamic.NewMessage(messageDescriptor)
	// TODO: Add in debug logging of the decoded JSON
	// decodedJson, _ := gabs.ParseJSON(jsonBytes)

	err := protoMessage.UnmarshalJSON(jsonBytes)
	if err != nil {
		return nil, err
	}

	return protoMessage.Marshal()
}
