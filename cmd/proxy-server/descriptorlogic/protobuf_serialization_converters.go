package descriptorlogic

import (
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
)

func JsonBytesToProtobufBytes(jsonBytes []byte, messageDescriptor *desc.MessageDescriptor) (protobuf_byte []byte, err error) {
	protoMessage := dynamic.NewMessage(messageDescriptor)
	// TODO: Add in debug logging of the decoded JSON
	// decodedJson, _ := gabs.ParseJSON(jsonBytes)

	protoMessage.UnmarshalJSON(jsonBytes)

	return protoMessage.Marshal()
}
