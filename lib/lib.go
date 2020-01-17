package lib

import (
	"bytes"
	"context"
	"errors"
	platform_connector_lib "github.com/SENERGY-Platform/platform-connector-lib"
	"github.com/SENERGY-Platform/platform-connector-lib/model"
	paho "github.com/eclipse/paho.mqtt.golang"
	"log"
	"os"
	"strings"
	"text/template"
	"time"
)

func Start(ctx context.Context, config Config) error {
	switch config.MqttLogLevel {
	case "critical":
		paho.CRITICAL = log.New(os.Stderr, "[paho] ", log.LstdFlags)
	case "error":
		paho.CRITICAL = log.New(os.Stderr, "[paho] ", log.LstdFlags)
		paho.ERROR = log.New(os.Stderr, "[paho] ", log.LstdFlags)
	case "warn":
		paho.CRITICAL = log.New(os.Stderr, "[paho] ", log.LstdFlags)
		paho.ERROR = log.New(os.Stderr, "[paho] ", log.LstdFlags)
		paho.WARN = log.New(os.Stderr, "[paho] ", log.LstdFlags)
	case "debug":
		paho.CRITICAL = log.New(os.Stderr, "[paho] ", log.LstdFlags)
		paho.ERROR = log.New(os.Stderr, "[paho] ", log.LstdFlags)
		paho.WARN = log.New(os.Stderr, "[paho] ", log.LstdFlags)
		paho.DEBUG = log.New(os.Stdout, "[paho] ", log.LstdFlags)
	}

	libConf := platform_connector_lib.Config{
		ZookeeperUrl:             config.ZookeeperUrl,
		KafkaResponseTopic:       config.KafkaResponseTopic,
		KafkaGroupName:           config.KafkaGroupName,
		FatalKafkaError:          config.FatalKafkaError,
		Protocol:                 config.Protocol,
		DeviceManagerUrl:         config.DeviceManagerUrl,
		DeviceRepoUrl:            config.DeviceRepoUrl,
		AuthClientId:             config.AuthClientId,
		AuthClientSecret:         config.AuthClientSecret,
		AuthExpirationTimeBuffer: config.AuthExpirationTimeBuffer,
		AuthEndpoint:             config.AuthEndpoint,
		JwtPrivateKey:            config.JwtPrivateKey,
		JwtExpiration:            config.JwtExpiration,
		JwtIssuer:                config.JwtIssuer,
		DeviceExpiration:         config.DeviceExpiration,
		DeviceTypeExpiration:     config.DeviceTypeExpiration,
		TokenCacheExpiration:     config.TokenCacheExpiration,
		IotCacheUrl:              config.IotCacheUrl,
		TokenCacheUrl:            config.TokenCacheUrl,
		SyncKafka:                config.SyncKafka,
		SyncKafkaIdempotent:      config.SyncKafkaIdempotent,
		Debug:                    config.Debug,
	}

	connector := platform_connector_lib.New(libConf)

	if config.Debug {
		connector.SetKafkaLogger(log.New(os.Stdout, "[CONNECTOR-KAFKA] ", 0))
		connector.IotCache.Debug = true
	}

	AuthWebhooks(ctx, config, connector)

	time.Sleep(1 * time.Second) //ensure http server startup before continue

	mqtt, err := NewMqtt(config)
	if err != nil {
		return err
	}

	err = connector.SetAsyncCommandHandler(CreateCommandHandler(config, mqtt)).Start()

	if err != nil {
		connector.Stop()
		mqtt.Close()
		return err
	}

	go func() {
		<-ctx.Done()
		mqtt.Close()
		connector.Stop()
	}()

	return nil
}

func CreateCommandHandler(config Config, mqtt *MqttClient) platform_connector_lib.AsyncCommandHandler {
	return func(commandRequest model.ProtocolMsg, requestMsg platform_connector_lib.CommandRequestMsg, t time.Time) (err error) {
		endpoint := ""
		endpoint, err = CreateActuatorTopic(config.ActuatorTopicPattern, commandRequest.Metadata.Device.DeviceTypeId, commandRequest.Metadata.Device.Id, commandRequest.Metadata.Device.LocalId, commandRequest.Metadata.Service.Id, commandRequest.Metadata.Service.LocalId)
		if err != nil {
			return
		}
		err = mqtt.Publish(endpoint, commandRequest.Request.Input["payload"])
		return
	}
}

func CreateActuatorTopic(templ string, deviceTypeId string, deviceId string, deviceUri string, serviceId string, serviceUri string) (result string, err error) {
	var temp bytes.Buffer
	err = template.Must(template.New("actuatortopic").Parse(templ)).Execute(&temp, map[string]string{
		"DeviceTypeId":   deviceTypeId,
		"DeviceId":       deviceId,
		"LocalDeviceId":  deviceUri,
		"ServiceId":      serviceId,
		"LocalServiceId": serviceUri,
	})
	if err != nil {
		return
	}
	return temp.String(), nil
}

func ParseTopic(pattern string, topic string) (deviceTypeId string, deviceId string, deviceUri string, serviceId string, serviceUri string, err error) {
	patternParts := strings.Split(pattern, "/")
	topicParts := strings.Split(topic, "/")
	var ignore string
	index := map[string]*string{
		"{{.Ignore}}":         &ignore,
		"{{.DeviceTypeId}}":   &deviceTypeId,
		"{{.DeviceId}}":       &deviceId,
		"{{.LocalDeviceId}}":  &deviceUri,
		"{{.ServiceId}}":      &serviceId,
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
		} else {
			if part != topicParts[i] {
				err = errors.New("topic doesnt match pattern")
			}
		}
	}
	return
}
