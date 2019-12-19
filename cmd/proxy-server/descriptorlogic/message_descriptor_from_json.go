package descriptorlogic

import (
	"errors"
	"fmt"
	"github.com/mcon/pact-serialization-proxy/cmd/proxy-server/serialization"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/desc"
)

func GetMessageDescriptorFromBody(encoding *serialization.SerializationEncoding, path string) (messageDescriptor *desc.MessageDescriptor, err error) {
	fmt.Println(encoding)

	fileDescriptorSetBytes := make([]byte, 0, 100000)
	for _, child := range encoding.Description.FileDescriptorSet {
		fileDescriptorSetBytes = append(fileDescriptorSetBytes, byte(child))
	}

	fileDescriptorSet := &descriptor.FileDescriptorSet{}

	err = proto.Unmarshal(fileDescriptorSetBytes, fileDescriptorSet)
	if err != nil {
		return nil, err
	}

	var fileDescriptor *desc.FileDescriptor
	fileDescriptor, err = desc.CreateFileDescriptorFromSet(fileDescriptorSet)
	if err != nil {
		return nil, err
	}

	messages := fileDescriptor.GetMessageTypes()
	for _, msg := range messages {
		fmt.Println(msg.GetName(), encoding.Description.MessageName)
		if msg.GetName() == encoding.Description.MessageName {
			return msg, nil
		}
	}
	return nil, errors.New("Expected route was not found in interactions: " + path)
}
