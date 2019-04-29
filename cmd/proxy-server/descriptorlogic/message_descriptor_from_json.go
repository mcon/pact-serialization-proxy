package descriptorlogic

import (
	"errors"
	"fmt"

	"github.com/Jeffail/gabs"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/desc"
)

func GetMessageDescriptorFromBody(interaction *gabs.Container) (messageDescriptor *desc.MessageDescriptor, err error) {
	fmt.Println(interaction)
	fdcContainer := interaction.Path("response.encoding.description.fileDescriptorSet")
	fmt.Println(fdcContainer)
	fdcBytes := make([]byte, 0, 100000)

	fmt.Println(fdcContainer.String())
	children, _ := fdcContainer.Children()
	for _, child := range children {
		floatRepr := child.Data().(float64)
		fdcBytes = append(fdcBytes, byte(floatRepr))
	}

	fdc := &descriptor.FileDescriptorSet{}
	err = proto.Unmarshal(fdcBytes, fdc)
	if err != nil {
		return nil, err
	}

	var d *desc.FileDescriptor
	d, err = desc.CreateFileDescriptorFromSet(fdc)
	if err != nil {
		return nil, err
	}

	messages := d.GetMessageTypes()
	for _, msg := range messages {
		fmt.Println(msg.GetName(), interaction.Path("response.encoding.description.messageName").Data().(string))
		if msg.GetName() == interaction.Path("response.encoding.description.messageName").Data().(string) {
			return msg, nil
		}
	}
	return nil, errors.New("Expected route was not found in interactions: " + interaction.Path("request.path").String())
	//dynamic.Message
}
