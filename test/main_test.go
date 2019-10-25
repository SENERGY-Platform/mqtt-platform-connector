package test

import (
	"context"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib"
	"github.com/SENERGY-Platform/mqtt-platform-connector/test/server"
	"github.com/SENERGY-Platform/platform-connector-lib/model"
	"testing"
	"time"
)

func TestInit(t *testing.T) {
	defaultConfig, err := lib.LoadConfigLocation("../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer time.Sleep(10 * time.Second) //wait for docker cleanup
	defer cancel()

	config, err := server.New(ctx, defaultConfig)
	if err != nil {
		t.Error(err)
		return
	}

	err = lib.Start(ctx, config)
	if err != nil {
		t.Error(err)
		return
	}
}

func TestDeviceCreation(t *testing.T) {
	defaultConfig, err := lib.LoadConfigLocation("../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer time.Sleep(10 * time.Second) //wait for docker cleanup
	defer cancel()

	config, err := server.New(ctx, defaultConfig)
	if err != nil {
		t.Error(err)
		return
	}

	err = lib.Start(ctx, config)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(1 * time.Second)

	deviceLocalId := "testservice1"
	serviceLocalId := "testservice1"
	deviceType := model.DeviceType{}
	protocol := model.Protocol{}

	t.Run("create protocol", func(t *testing.T) {
		protocol = createTestProtocol(t, config)
		time.Sleep(10 * time.Second) //wait for cqrs
	})

	t.Run("create device type", func(t *testing.T) {
		deviceType = createTestDeviceType(t, config, protocol, serviceLocalId)
		time.Sleep(10 * time.Second) //wait for cqrs
	})

	t.Run("send mqtt message", func(t *testing.T) {
		sendMqttEvent(t, config, deviceType.Id+"/"+deviceLocalId+"/"+serviceLocalId, `{"level":42}`)
		time.Sleep(10 * time.Second) //wait for cqrs
	})

	t.Run("check device creation", func(t *testing.T) {
		checkDevice(t, config, deviceLocalId, deviceType.Id)
	})
}

func sendMqttEvent(t *testing.T, config lib.Config, topic string, msg string) {
	mqtt, err := lib.NewMqtt(lib.Config{AuthClientId: "sepl", AuthClientSecret: "sepl", MqttBroker: config.MqttBroker, Qos: config.Qos})
	if err != nil {
		t.Fatal(err)
	}
	defer mqtt.Close()
	err = mqtt.Publish(topic, msg)
	if err != nil {
		t.Fatal(err)
	}
}
