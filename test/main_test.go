package test

import (
	"encoding/json"
	"errors"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib"
	"github.com/SENERGY-Platform/mqtt-platform-connector/test/server"
	platform_connector_lib "github.com/SENERGY-Platform/platform-connector-lib"
	"github.com/SENERGY-Platform/platform-connector-lib/iot"
	"github.com/SENERGY-Platform/platform-connector-lib/model"
	"github.com/SENERGY-Platform/platform-connector-lib/security"
	paho "github.com/eclipse/paho.mqtt.golang"
	"log"
	"testing"
	"time"
)

var mqttClient paho.Client
var config platform_connector_lib.Config
var connector *platform_connector_lib.Connector
var usertoken security.JwtToken = `Bearer eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICIzaUtabW9aUHpsMmRtQnBJdS1vSkY4ZVVUZHh4OUFIckVOcG5CcHM5SjYwIn0.eyJqdGkiOiJhYTk5MzMzYS04YzJiLTQ3OWEtODFjNC1kZTQwNjg2ZDNlODciLCJleHAiOjE1NjA5NDc4OTYsIm5iZiI6MCwiaWF0IjoxNTYwOTQ0Mjk2LCJpc3MiOiJodHRwczovL2F1dGguc2VwbC5pbmZhaS5vcmcvYXV0aC9yZWFsbXMvbWFzdGVyIiwiYXVkIjoiZnJvbnRlbmQiLCJzdWIiOiJkZDY5ZWEwZC1mNTUzLTQzMzYtODBmMy03ZjQ1NjdmODVjN2IiLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJmcm9udGVuZCIsIm5vbmNlIjoiMWEyZmY0MzktMjg1OC00NDI1LTlhOTAtMjc0MDY5OGYyODczIiwiYXV0aF90aW1lIjoxNTYwOTQ0Mjg4LCJzZXNzaW9uX3N0YXRlIjoiMmJmY2M5MzItZWI5OC00YzMwLThiZGYtOWY1MDI1N2ViY2E1IiwiYWNyIjoiMCIsImFsbG93ZWQtb3JpZ2lucyI6WyIqIl0sInJlYWxtX2FjY2VzcyI6eyJyb2xlcyI6WyJjcmVhdGUtcmVhbG0iLCJhZG1pbiIsImRldmVsb3BlciIsInVtYV9hdXRob3JpemF0aW9uIiwidXNlciJdfSwicmVzb3VyY2VfYWNjZXNzIjp7Im1hc3Rlci1yZWFsbSI6eyJyb2xlcyI6WyJ2aWV3LWlkZW50aXR5LXByb3ZpZGVycyIsInZpZXctcmVhbG0iLCJtYW5hZ2UtaWRlbnRpdHktcHJvdmlkZXJzIiwiaW1wZXJzb25hdGlvbiIsImNyZWF0ZS1jbGllbnQiLCJtYW5hZ2UtdXNlcnMiLCJxdWVyeS1yZWFsbXMiLCJ2aWV3LWF1dGhvcml6YXRpb24iLCJxdWVyeS1jbGllbnRzIiwicXVlcnktdXNlcnMiLCJtYW5hZ2UtZXZlbnRzIiwibWFuYWdlLXJlYWxtIiwidmlldy1ldmVudHMiLCJ2aWV3LXVzZXJzIiwidmlldy1jbGllbnRzIiwibWFuYWdlLWF1dGhvcml6YXRpb24iLCJtYW5hZ2UtY2xpZW50cyIsInF1ZXJ5LWdyb3VwcyJdfSwiQmFja2VuZC1yZWFsbSI6eyJyb2xlcyI6WyJ2aWV3LXJlYWxtIiwidmlldy1pZGVudGl0eS1wcm92aWRlcnMiLCJtYW5hZ2UtaWRlbnRpdHktcHJvdmlkZXJzIiwiaW1wZXJzb25hdGlvbiIsImNyZWF0ZS1jbGllbnQiLCJtYW5hZ2UtdXNlcnMiLCJxdWVyeS1yZWFsbXMiLCJ2aWV3LWF1dGhvcml6YXRpb24iLCJxdWVyeS1jbGllbnRzIiwicXVlcnktdXNlcnMiLCJtYW5hZ2UtZXZlbnRzIiwibWFuYWdlLXJlYWxtIiwidmlldy1ldmVudHMiLCJ2aWV3LXVzZXJzIiwidmlldy1jbGllbnRzIiwibWFuYWdlLWF1dGhvcml6YXRpb24iLCJtYW5hZ2UtY2xpZW50cyIsInF1ZXJ5LWdyb3VwcyJdfSwiYWNjb3VudCI6eyJyb2xlcyI6WyJtYW5hZ2UtYWNjb3VudCIsIm1hbmFnZS1hY2NvdW50LWxpbmtzIiwidmlldy1wcm9maWxlIl19fSwicm9sZXMiOlsidW1hX2F1dGhvcml6YXRpb24iLCJhZG1pbiIsImNyZWF0ZS1yZWFsbSIsImRldmVsb3BlciIsInVzZXIiLCJvZmZsaW5lX2FjY2VzcyJdLCJuYW1lIjoiU2VwbCBBZG1pbiIsInByZWZlcnJlZF91c2VybmFtZSI6InNlcGwiLCJnaXZlbl9uYW1lIjoiU2VwbCIsImZhbWlseV9uYW1lIjoiQWRtaW4iLCJlbWFpbCI6InNlcGxAc2VwbC5kZSJ9.fQ03R2IJrn9-zGPAktrl6eUdnWZ4aWF2T4VGS6ACtzeoZ-43y9Wuu_kfRAPIdkEGQJOrydBnLMQO1_-5H_FLQXWXa_dtGOSNF71K3HBxrg7OTRe5Wt3zQmmWc3PspQigK-gTUh6S-jfCkhV3TJD_f7ZjetqizJEe0nCqBfvjr_DzR8AmfWdMtn4c7sp2Rb7mwMVWfzg9piRhjve2fdVMCvEtD60eUIinH7B82Hil6XTFWyUvAY9L_mUQ9xqHxSffSoyi6e5h0ZVj-NWBuJgRAKozpK-VDZAAi_3TuxZLkJZM1s54EL2eojvM1GNj_QudAOLOsUKYKVWWIWcmEsZ3ig`

func TestMqtt(t *testing.T) {
	err := lib.LoadConfig("../config.json")
	if err != nil {
		log.Fatal(err)
	}
	libConf, err := platform_connector_lib.LoadConfig("../config.json")
	if err != nil {
		log.Fatal(err)
	}

	var stop func()
	connector, config, stop, err = server.New(libConf)
	if err != nil {
		log.Fatal(err)
	}
	defer stop()
	time.Sleep(5 * time.Second)
	options := paho.NewClientOptions().
		SetPassword("sepl").
		SetUsername("sepl").
		SetClientID("test").
		SetAutoReconnect(true).
		SetCleanSession(true).
		AddBroker(lib.Config.MqttBroker)
	mqttClient = paho.NewClient(options)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Println("Error on Client.Connect(): ", token.Error())
		t.Error(token.Error())
		return
	}

	t.Run("testEvent", func(t *testing.T) {
		testEvent(t, connector)
	})
}


func createTestCommandMsg(config platform_connector_lib.Config, deviceUri string, serviceUri string, msg map[string]interface{}) (result model.Envelope, err error) {
	token, err := security.New(config.AuthEndpoint, config.AuthClientId, config.AuthClientSecret, config.JwtIssuer, config.JwtPrivateKey, config.JwtExpiration, config.AuthExpirationTimeBuffer, 0, []string{}).Access()
	if err != nil {
		return result, err
	}
	iot := iot.New(config.DeviceManagerUrl, config.DeviceRepoUrl)
	device, err := iot.GetDeviceByLocalId(deviceUri, token)
	if err != nil {
		return result, err
	}
	dt, err := iot.GetDeviceType(device.DeviceTypeId, token)

	found := false
	for _, service := range dt.Services {
		if service.LocalId == serviceUri {
			found = true
			result.ServiceId = service.Id
			result.DeviceId = device.Id
			value := model.ProtocolMsg{
				Service:          service,
				ServiceId:        service.Id,
				ServiceUrl:       serviceUri,
				DeviceUrl:        deviceUri,
				DeviceInstanceId: device.Id,
				OutputName:       "result",
			}

			if msg != nil {
				payload, err := json.Marshal(msg)
				if err != nil {
					return result, err
				}
				value.ProtocolParts = []model.ProtocolPart{{Value: string(payload), Name: "payload"}}
			}
			result.Value = value
		}
	}

	if !found {
		err = errors.New("unable to find device for command creation")
	}

	return
}