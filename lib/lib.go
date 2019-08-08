package lib

import (
	"bytes"
	"errors"
	platform_connector_lib "github.com/SENERGY-Platform/platform-connector-lib"
	"github.com/SENERGY-Platform/platform-connector-lib/model"
	"strings"
	"text/template"
	"time"
)

func CommandHandler(commandRequest model.ProtocolMsg, requestMsg platform_connector_lib.CommandRequestMsg, t time.Time) (err error) {
	endpoint := ""
	endpoint, err = CreateActuatorTopic(Config.ActuatorTopicPattern, commandRequest.DeviceInstanceId, commandRequest.DeviceUrl, commandRequest.ServiceId, commandRequest.ServiceUrl)
	if err != nil {
		return
	}
	err = MqttPublish(endpoint, getProtocolPartMap(commandRequest.ProtocolParts)["payload"])
	return
}

func getProtocolPartMap(protocolParts []model.ProtocolPart) (result map[string]string) {
	result = map[string]string{}
	for _, pp := range protocolParts {
		result[pp.Name] = pp.Value
	}
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