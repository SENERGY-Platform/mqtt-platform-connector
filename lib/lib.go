package lib

import (
	"context"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/topic"
	platform_connector_lib "github.com/SENERGY-Platform/platform-connector-lib"
	"github.com/SENERGY-Platform/platform-connector-lib/kafka"
	"github.com/SENERGY-Platform/platform-connector-lib/model"
	paho "github.com/eclipse/paho.mqtt.golang"
	"log"
	"os"
	"time"
)

func Start(ctx context.Context, config Config) error {
	if config.KafkaProducerSlowTimeoutSec != 0 {
		kafka.SlowProducerTimeout = time.Duration(config.KafkaProducerSlowTimeoutSec) * time.Second
	}

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
		SerializationFallback:    config.SerializationFallback,

		Validate:                  config.Validate,
		ValidateAllowMissingField: config.ValidateAllowMissingField,
		ValidateAllowUnknownField: config.ValidateAllowUnknownField,

		SemanticRepositoryUrl:    config.SemanticRepoUrl,
		CharacteristicExpiration: config.SemanticExpiration,

		PartitionsNum:     config.KafkaPartitionNum,
		ReplicationFactor: config.KafkaReplicationFactor,
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
		endpoint, err = topic.New(nil, config.ActuatorTopicPattern).Create(commandRequest.Metadata.Device.Id, commandRequest.Metadata.Service.LocalId)
		if err != nil {
			return
		}
		err = mqtt.Publish(endpoint, commandRequest.Request.Input["payload"])
		return
	}
}
