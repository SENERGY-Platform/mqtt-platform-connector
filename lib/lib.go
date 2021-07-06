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

func Start(basectx context.Context, config Config) (err error) {
	ctx, cancel := context.WithCancel(basectx)
	defer func() {
		if err != nil {
			cancel()
		}
	}()
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
		KafkaUrl:                 config.KafkaUrl,
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
		Debug:                    config.Debug,
		SerializationFallback:    config.SerializationFallback,

		Validate:                  config.Validate,
		ValidateAllowMissingField: config.ValidateAllowMissingField,
		ValidateAllowUnknownField: config.ValidateAllowUnknownField,

		SemanticRepositoryUrl:    config.SemanticRepoUrl,
		CharacteristicExpiration: config.SemanticExpiration,

		PartitionsNum:     config.KafkaPartitionNum,
		ReplicationFactor: config.KafkaReplicationFactor,

		PublishToPostgres: config.PublishToPostgres,
		PostgresHost:      config.PostgresHost,
		PostgresPort:      config.PostgresPort,
		PostgresUser:      config.PostgresUser,
		PostgresPw:        config.PostgresPw,
		PostgresDb:        config.PostgresDb,
	}

	connector := platform_connector_lib.New(libConf)

	if config.Debug {
		connector.SetKafkaLogger(log.New(os.Stdout, "[CONNECTOR-KAFKA] ", 0))
		connector.IotCache.Debug = true
	}

	err = connector.InitProducer(ctx, []platform_connector_lib.Qos{platform_connector_lib.Async, platform_connector_lib.Sync, platform_connector_lib.SyncIdempotent})
	if err != nil {
		return err
	}

	AuthWebhooks(ctx, config, connector)

	time.Sleep(1 * time.Second) //ensure http server startup before continue

	mqtt, err := NewMqtt(config)
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		mqtt.Close()
	}()

	err = connector.SetAsyncCommandHandler(CreateCommandHandler(config, mqtt)).StartConsumer(ctx)
	if err != nil {
		return err
	}

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
