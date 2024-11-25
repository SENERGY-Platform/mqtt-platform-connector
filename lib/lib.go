package lib

import (
	"context"
	"errors"
	"fmt"
	"github.com/IBM/sarama"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/configuration"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/connectionlog"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/topic"
	platform_connector_lib "github.com/SENERGY-Platform/platform-connector-lib"
	"github.com/SENERGY-Platform/platform-connector-lib/kafka"
	"github.com/SENERGY-Platform/platform-connector-lib/model"
	"github.com/SENERGY-Platform/platform-connector-lib/statistics"
	paho "github.com/eclipse/paho.mqtt.golang"
	"log"
	"os"
	"strings"
	"time"
)

func Start(basectx context.Context, config configuration.Config) (err error) {
	ctx, cancel := context.WithCancel(basectx)
	defer func() {
		if err != nil {
			cancel()
		}
	}()

	asyncFlushFrequency, err := time.ParseDuration(config.AsyncFlushFrequency)
	if err != nil {
		return err
	}

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

		CharacteristicExpiration: config.DeviceExpiration,

		PartitionsNum:     config.KafkaPartitionNum,
		ReplicationFactor: config.KafkaReplicationFactor,

		PublishToPostgres: config.PublishToPostgres,
		PostgresHost:      config.PostgresHost,
		PostgresPort:      config.PostgresPort,
		PostgresUser:      config.PostgresUser,
		PostgresPw:        config.PostgresPw,
		PostgresDb:        config.PostgresDb,

		HttpCommandConsumerPort: config.HttpCommandConsumerPort,

		SyncCompression:     getKafkaCompression(config.SyncCompression),
		AsyncCompression:    getKafkaCompression(config.AsyncCompression),
		AsyncFlushFrequency: asyncFlushFrequency,
		AsyncFlushMessages:  int(config.AsyncFlushMessages),
		AsyncPgThreadMax:    int(config.AsyncPgThreadMax),

		KafkaConsumerMinBytes: int(config.KafkaConsumerMinBytes),
		KafkaConsumerMaxBytes: int(config.KafkaConsumerMaxBytes),
		KafkaConsumerMaxWait:  config.KafkaConsumerMaxWait,

		IotCacheTimeout:                      config.IotCacheTimeout,
		IotCacheMaxIdleConns:                 int(config.IotCacheMaxIdleConns),
		KafkaTopicConfigs:                    config.KafkaTopicConfigs,
		DeviceTypeTopic:                      config.DeviceTypeTopic,
		PermQueryUrl:                         config.PermQueryUrl,
		NotificationUrl:                      config.NotificationUrl,
		NotificationsIgnoreDuplicatesWithinS: config.NotificationsIgnoreDuplicatesWithinS,
		NotificationUserOverwrite:            config.NotificationUserOverwrite,
		DeveloperNotificationUrl:             config.DeveloperNotificationUrl,
		MutedUserNotificationTitles:          config.MutedUserNotificationTitles,
	}

	connector, err := platform_connector_lib.New(libConf)
	if err != nil {
		return err
	}

	if config.Debug {
		connector.SetKafkaLogger(log.New(os.Stdout, "[CONNECTOR-KAFKA] ", 0))
		connector.IotCache.Debug = true
	}

	err = connector.InitProducer(ctx, []platform_connector_lib.Qos{platform_connector_lib.Async, platform_connector_lib.Sync, platform_connector_lib.SyncIdempotent})
	if err != nil {
		return err
	}

	var logging connectionlog.ConnectionLog = connectionlog.Void
	if config.SubscriptionDbConStr != "" && config.SubscriptionDbConStr != "-" {
		producer, err := connector.GetProducer(platform_connector_lib.Sync)
		if err != nil {
			return err
		}
		logging, err = connectionlog.New(producer, config.SubscriptionDbConStr, config.DeviceLogTopic)
		if err != nil {
			return err
		}
	}

	AuthWebhooks(ctx, config, connector, logging)

	time.Sleep(1 * time.Second) //ensure http server startup before continue

	mqtt, err := MqttStart(ctx, config)
	if err != nil {
		return err
	}

	statistics.Init() //ensure start of prometheus metrics endpoint

	if config.CommandWorkerCount > 1 {
		err = connector.SetAsyncCommandHandler(CreateQueuedCommandHandler(ctx, config, mqtt)).StartConsumer(ctx)
	} else {
		err = connector.SetAsyncCommandHandler(CreateCommandHandler(config, mqtt)).StartConsumer(ctx)
	}

	return err
}

type commandQueueValue struct {
	commandRequest model.ProtocolMsg
	requestMsg     platform_connector_lib.CommandRequestMsg
	t              time.Time
}

func CreateQueuedCommandHandler(ctx context.Context, config configuration.Config, mqtt Mqtt) platform_connector_lib.AsyncCommandHandler {
	queue := make(chan commandQueueValue, config.CommandWorkerCount)
	handler := CreateCommandHandler(config, mqtt)
	for i := int64(0); i < config.CommandWorkerCount; i++ {
		go func() {
			for msg := range queue {
				err := handler(msg.commandRequest, msg.requestMsg, msg.t)
				if err != nil {
					log.Println("ERROR: ", err)
				}
			}
		}()
	}
	go func() {
		<-ctx.Done()
		close(queue)
	}()
	return func(commandRequest model.ProtocolMsg, requestMsg platform_connector_lib.CommandRequestMsg, t time.Time) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = errors.New(fmt.Sprint(r))
			}
		}()
		queue <- commandQueueValue{
			commandRequest: commandRequest,
			requestMsg:     requestMsg,
			t:              t,
		}
		return err
	}
}

func CreateCommandHandler(config configuration.Config, mqtt Mqtt) platform_connector_lib.AsyncCommandHandler {
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

func getKafkaCompression(compression string) sarama.CompressionCodec {
	switch strings.ToLower(compression) {
	case "":
		return sarama.CompressionNone
	case "-":
		return sarama.CompressionNone
	case "none":
		return sarama.CompressionNone
	case "gzip":
		return sarama.CompressionGZIP
	case "snappy":
		return sarama.CompressionSnappy
	}
	log.Println("WARNING: unknown compression", compression, "fallback to none")
	return sarama.CompressionNone
}
