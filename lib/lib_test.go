package lib

import (
	"encoding/json"
	"fmt"
)

func ExampleCreateActuatorTopic() {
	fmt.Println(CreateActuatorTopic("cmd/{{.DeviceId}}/{{.ServiceId}}", "dtid", "foo", "", "bar", ""))
	fmt.Println(CreateActuatorTopic("cmd/{{.DeviceId}}/{{.ServiceId}}", "dtid", "", "foo", "", "bar"))
	fmt.Println(CreateActuatorTopic("cmd/{{.LocalDeviceId}}/{{.LocalServiceId}}", "dtid", "", "foo", "", "bar"))

	fmt.Println(CreateActuatorTopic("cmd/{{.DeviceTypeId}}/{{.DeviceId}}/{{.ServiceId}}", "dtid", "foo", "", "bar", ""))
	fmt.Println(CreateActuatorTopic("cmd/{{.DeviceTypeId}}/{{.DeviceId}}/{{.ServiceId}}", "dtid", "", "foo", "", "bar"))
	fmt.Println(CreateActuatorTopic("cmd/{{.DeviceTypeId}}/{{.LocalDeviceId}}/{{.LocalServiceId}}", "dtid", "", "foo", "", "bar"))

	//output:
	//cmd/foo/bar <nil>
	//cmd// <nil>
	//cmd/foo/bar <nil>
	//cmd/dtid/foo/bar <nil>
	//cmd/dtid// <nil>
	//cmd/dtid/foo/bar <nil>
}

func ExampleParseTopic() {
	fmt.Println(toListString(ParseTopic("cmd/{{.DeviceId}}/{{.ServiceId}}", "cmd/a9B7ddfMShqI26yT9hqnsw/bar")))
	fmt.Println(toListString(ParseTopic("cmd/{{.DeviceId}}/{{.ServiceId}}", "cmd/urn:infai:ses:device:6bd07b75-d7cc-4a1a-88db-ac93f61aa7b3/bar")))
	fmt.Println(toListString(ParseTopic("cmd/{{.LocalDeviceId}}/{{.LocalServiceId}}", "cmd/foo/bar")))
	fmt.Println(toListString(ParseTopic("fail/{{.LocalDeviceId}}/{{.LocalServiceId}}", "cmd/foo/bar")))
	fmt.Println(toListString(ParseTopic("cmd/{{.LocalDeviceId}}/{{.LocalServiceId}}", "cmd/foo/bar/fail")))
	fmt.Println(toListString(ParseTopic("cmd/{{.LocalDeviceId}}/{{.LocalServiceId}}", "cmd/fail")))

	fmt.Println(toListString(ParseTopic("cmd/{{.DeviceTypeId}}/{{.DeviceId}}/{{.ServiceId}}", "cmd/foo/urn:infai:ses:device:6bd07b75-d7cc-4a1a-88db-ac93f61aa7b3")))
	fmt.Println(toListString(ParseTopic("cmd/{{.DeviceTypeId}}/{{.LocalDeviceId}}/{{.LocalServiceId}}", "cmd/foo/bar")))
	fmt.Println(toListString(ParseTopic("fail/{{.DeviceTypeId}}/{{.LocalDeviceId}}/{{.LocalServiceId}}", "cmd/foo/bar")))
	fmt.Println(toListString(ParseTopic("cmd/{{.DeviceTypeId}}/{{.LocalDeviceId}}/{{.LocalServiceId}}", "cmd/foo/bar/fail")))
	fmt.Println(toListString(ParseTopic("cmd/{{.DeviceTypeId}}/{{.LocalDeviceId}}/{{.LocalServiceId}}", "cmd/fail")))

	fmt.Println(toListString(ParseTopic("cmd/{{.DeviceTypeId}}/{{.DeviceId}}/{{.ServiceId}}", "cmd/dtid/urn:infai:ses:device:6bd07b75-d7cc-4a1a-88db-ac93f61aa7b3/bar")))
	fmt.Println(toListString(ParseTopic("cmd/{{.DeviceTypeId}}/{{.LocalDeviceId}}/{{.LocalServiceId}}", "cmd/dtid/foo/bar")))
	fmt.Println(toListString(ParseTopic("fail/{{.DeviceTypeId}}/{{.LocalDeviceId}}/{{.LocalServiceId}}", "cmd/dtid/foo/bar")))
	fmt.Println(toListString(ParseTopic("cmd/{{.DeviceTypeId}}/{{.LocalDeviceId}}/{{.LocalServiceId}}", "cmd/dtid/foo/bar/fail")))
	fmt.Println(toListString(ParseTopic("cmd/{{.DeviceTypeId}}/{{.LocalDeviceId}}/{{.LocalServiceId}}", "cmd/dtid/fail")))

	//output:
	//["","urn:infai:ses:device:6bd07b75-d7cc-4a1a-88db-ac93f61aa7b3","","bar","",null]
	//["","urn:infai:ses:device:6bd07b75-d7cc-4a1a-88db-ac93f61aa7b3","","bar","",null]
	//["","","foo","","bar",null]
	//["","","foo","","bar",{}]
	//["","","","","",{}]
	//["","","","","",{}]
	//["","","","","",{}]
	//["","","","","",{}]
	//["","","","","",{}]
	//["foo","","bar","","fail",null]
	//["","","","","",{}]
	//["dtid","urn:infai:ses:device:6bd07b75-d7cc-4a1a-88db-ac93f61aa7b3","","bar","",null]
	//["dtid","","foo","","bar",null]
	//["dtid","","foo","","bar",{}]
	//["","","","","",{}]
	//["","","","","",{}]
}

func toListString(element ...interface{}) string {
	b, _ := json.Marshal(append([]interface{}{}, element...))
	return string(b)
}
