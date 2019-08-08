package lib

import (
	"bytes"
	"errors"
	platform_connector_lib "github.com/SENERGY-Platform/platform-connector-lib"
	"strings"
	"text/template"
)

func CommandHandler(deviceId string, deviceUri string, serviceId string, serviceUri string, requestMsg platform_connector_lib.CommandRequestMsg) (responseMsg platform_connector_lib.CommandResponseMsg, err error) {
	responseMsg = map[string]string{}
	endpoint := ""
	endpoint, err = CreateActuatorTopic(Config.ActuatorTopicPattern, deviceId, deviceUri, serviceId, serviceUri)
	if err != nil {
		return
	}
	err = MqttPublish(endpoint, responseMsg["payload"])
	return
}

func CreateActuatorTopic(templ string, deviceId string, deviceUri string, serviceId string, serviceUri string) (result string, err error) {
	var temp bytes.Buffer
	err = template.Must(template.New("actuatortopic").Parse(templ)).Execute(&temp, map[string]string{
		"DeviceId": deviceId,
		"LocalDeviceId": deviceUri,
		"ServiceId": serviceId,
		"LocalServiceId":serviceUri,
	})
	if err != nil {
		return
	}
	return temp.String(), nil
}

func ParseTopic(pattern string, topic string)(deviceId string, deviceUri string, serviceId string, serviceUri string, err error){
	patternParts := strings.Split(pattern, "/")
	topicParts := strings.Split(topic, "/")
	index := map[string]*string{
		"{{.DeviceId}}": &deviceId,
		"{{.LocalDeviceId}}": &deviceUri,
		"{{.ServiceId}}": &serviceId,
		"{{.LocalServiceId}}": &serviceUri,
	}
	if len(patternParts) != len(topicParts) {
		err = errors.New("topic doesnt match pattern")
		return
	}
	for i, part := range patternParts {
		ptr, ok := index[part]
		if ok {
			*ptr = topicParts[i]
		}else{
			if part != topicParts[i] {
				err = errors.New("topic doesnt match pattern")
			}
		}
	}
	return
}