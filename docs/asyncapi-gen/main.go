/*
 * Copyright 2025 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"flag"
	"fmt"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/configuration"
	platform_connector_lib "github.com/SENERGY-Platform/platform-connector-lib"
	"github.com/SENERGY-Platform/platform-connector-lib/connectionlog"
	"github.com/SENERGY-Platform/platform-connector-lib/model"
	"log"
	"os"

	"github.com/swaggest/go-asyncapi/reflector/asyncapi-2.4.0"
	"github.com/swaggest/go-asyncapi/spec-2.4.0"
)

//go:generate go run main.go

type Envelope struct {
	DeviceId  string                 `json:"device_id,omitempty"`
	ServiceId string                 `json:"service_id,omitempty"`
	Value     map[string]interface{} `json:"value"`
}

func main() {
	configLocation := flag.String("config", "../../config.json", "configuration file")
	flag.Parse()

	conf, err := configuration.Load(*configLocation)
	if err != nil {
		log.Fatal("ERROR: unable to load config", err)
	}

	asyncAPI := spec.AsyncAPI{}
	asyncAPI.Info.Title = "Mqtt-Platform-Connector"
	asyncAPI.Info.Description = "topics or parts of topics in '[]' are placeholders"

	asyncAPI.AddServer("kafka", spec.Server{
		URL:      conf.KafkaUrl,
		Protocol: "kafka",
	})

	asyncAPI.AddServer("mqtt", spec.Server{
		URL:      conf.MqttBroker,
		Protocol: "mqtt",
	})

	reflector := asyncapi.Reflector{}
	reflector.Schema = &asyncAPI

	mustNotFail := func(err error) {
		if err != nil {
			panic(err.Error())
		}
	}

	//"topic is a service.Id with replaced '#' and ':' by '_'"
	mustNotFail(reflector.AddChannel(asyncapi.ChannelInfo{
		Name: "[service-topic]",
		BaseChannelItem: &spec.ChannelItem{
			Servers:     []string{"kafka"},
			Description: "[service-topic] is a service.Id with replaced '#' and ':' by '_'",
		},
		Subscribe: &asyncapi.MessageSample{
			MessageEntity: spec.MessageEntity{
				Name:  "Envelope",
				Title: "Envelope",
			},
			MessageSample: new(Envelope),
		},
	}))

	mustNotFail(reflector.AddChannel(asyncapi.ChannelInfo{
		Name: conf.KafkaResponseTopic,
		BaseChannelItem: &spec.ChannelItem{
			Description: "may be configured by config.KafkaResponseTopic",
			Servers:     []string{"kafka"},
		},
		Subscribe: &asyncapi.MessageSample{
			MessageEntity: spec.MessageEntity{
				Name:  "ProtocolMsg",
				Title: "ProtocolMsg",
			},
			MessageSample: new(model.ProtocolMsg),
		},
	}))

	mustNotFail(reflector.AddChannel(asyncapi.ChannelInfo{
		Name: conf.Protocol,
		BaseChannelItem: &spec.ChannelItem{
			Servers:     []string{"kafka"},
			Description: "may be configured by config.Protocol",
		},
		Publish: &asyncapi.MessageSample{
			MessageEntity: spec.MessageEntity{
				Name:  "ProtocolMsg",
				Title: "ProtocolMsg",
			},
			MessageSample: new(model.ProtocolMsg),
		},
	}))

	mustNotFail(reflector.AddChannel(asyncapi.ChannelInfo{
		Name: conf.DeviceTypeTopic,
		BaseChannelItem: &spec.ChannelItem{
			Servers: []string{"kafka"},
		},
		Publish: &asyncapi.MessageSample{
			MessageEntity: spec.MessageEntity{
				Name:  "DeviceTypeCommand",
				Title: "DeviceTypeCommand",
			},
			MessageSample: new(platform_connector_lib.DeviceTypeCommand),
		},
	}))

	mustNotFail(reflector.AddChannel(asyncapi.ChannelInfo{
		Name: conf.DeviceLogTopic,
		BaseChannelItem: &spec.ChannelItem{
			Servers: []string{"kafka"},
		},
		Subscribe: &asyncapi.MessageSample{
			MessageEntity: spec.MessageEntity{
				Name:  "DeviceLog",
				Title: "DeviceLog",
			},
			MessageSample: new(connectionlog.DeviceLog),
		},
	}))

	mustNotFail(reflector.AddChannel(asyncapi.ChannelInfo{
		Name: "[local-service-id]",
		BaseChannelItem: &spec.ChannelItem{
			Servers:     []string{"mqtt"},
			Description: "[local-service-id] is a service-local-id interpreted as a go template where {{.DeviceId}} and {{.ShortDeviceId}} are valid placeholder. if the local-service-id does not contain a placeholder, config.actuator_topic_pattern is used as the template. config.actuator_topic_pattern may contain {{.LocalServiceId}} as placeholder (e.g.: 'something/{{.LocalDeviceId}}/{{.LocalServiceId}}')",
		},
		Subscribe: &asyncapi.MessageSample{
			MessageEntity: spec.MessageEntity{
				Name:        "DeviceCommand",
				Title:       "DeviceCommand",
				Description: "as described by the DeviceType.Service",
			},
		},
	}))

	mustNotFail(reflector.AddChannel(asyncapi.ChannelInfo{
		Name: "[any-topic-with-valid-device-id]",
		BaseChannelItem: &spec.ChannelItem{
			Servers:     []string{"mqtt"},
			Description: "any topic-level must contain a valid device-id or short-device-id. device-services are matched by searching the topic for the local-service-id.",
		},
		Publish: &asyncapi.MessageSample{
			MessageEntity: spec.MessageEntity{
				Name:        "DeviceEvent",
				Title:       "DeviceEvent",
				Description: "as described by the DeviceType.Service",
			},
		},
	}))

	buff, err := reflector.Schema.MarshalJSON()
	mustNotFail(err)

	fmt.Println(string(buff))
	mustNotFail(os.WriteFile("asyncapi.json", buff, 0o600))
}
