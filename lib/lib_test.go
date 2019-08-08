package lib

import (
	"encoding/json"
	"fmt"
)

func ExampleCreateActuatorTopic() {
	fmt.Println(CreateActuatorTopic("cmd/{{.DeviceId}}/{{.ServiceId}}", "foo", "", "bar", ""))
	fmt.Println(CreateActuatorTopic("cmd/{{.DeviceId}}/{{.ServiceId}}", "", "foo", "", "bar"))
	fmt.Println(CreateActuatorTopic("cmd/{{.LocalDeviceId}}/{{.LocalServiceId}}", "", "foo", "", "bar"))

	//output:
	//cmd/foo/bar <nil>
	//cmd// <nil>
	//cmd/foo/bar <nil>
}

func ExampleParseTopic() {
	fmt.Println(toListString(ParseTopic("cmd/{{.DeviceId}}/{{.ServiceId}}", "cmd/foo/bar")))
	fmt.Println(toListString(ParseTopic("cmd/{{.LocalDeviceId}}/{{.LocalServiceId}}", "cmd/foo/bar")))
	fmt.Println(toListString(ParseTopic("fail/{{.LocalDeviceId}}/{{.LocalServiceId}}", "cmd/foo/bar")))
	fmt.Println(toListString(ParseTopic("cmd/{{.LocalDeviceId}}/{{.LocalServiceId}}", "cmd/foo/bar/fail")))
	fmt.Println(toListString(ParseTopic("cmd/{{.LocalDeviceId}}/{{.LocalServiceId}}", "cmd/fail")))

	//output:
	//["foo","","bar","",null]
	//["","foo","","bar",null]
	//["","foo","","bar",{}]
	//["","","","",{}]
	//["","","","",{}]
}

func toListString(element ...interface{})string{
	b, _ := json.Marshal(append([]interface{}{}, element...))
	return string(b)
}