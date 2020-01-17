package test

import (
	"context"
	"encoding/json"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib"
	"github.com/SENERGY-Platform/mqtt-platform-connector/test/server"
	"github.com/SENERGY-Platform/platform-connector-lib/kafka"
	"github.com/SENERGY-Platform/platform-connector-lib/model"
	uuid "github.com/satori/go.uuid"
	"log"
	"reflect"
	"sync"
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

func TestEventWithoutProvisioning(t *testing.T) {
	defaultConfig, err := lib.LoadConfigLocation("../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer time.Sleep(10 * time.Second) //wait for docker cleanup
	defer cancel()

	defaultConfig.SensorTopicPattern = "{{.Ignore}}/{{.LocalDeviceId}}/{{.LocalServiceId}}"

	config, err := server.New(ctx, defaultConfig)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(2 * time.Second)

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
	device := model.Device{}
	msg := `{"level":42}`

	t.Run("create protocol", func(t *testing.T) {
		protocol = createTestProtocol(t, config)
		time.Sleep(10 * time.Second) //wait for cqrs
	})

	t.Run("create device type", func(t *testing.T) {
		deviceType = createTestDeviceType(t, config, protocol, serviceLocalId)
		time.Sleep(10 * time.Second) //wait for cqrs
	})

	t.Run("create device", func(t *testing.T) {
		device = createTestDevice(t, config, deviceType, deviceLocalId)
		time.Sleep(10 * time.Second) //wait for cqrs
	})

	t.Run("send mqtt message", func(t *testing.T) {
		sendMqttEvent(t, config, "senergy/"+deviceLocalId+"/"+serviceLocalId, msg)
		time.Sleep(10 * time.Second) //wait for cqrs
	})

	t.Run("check kafka event", func(t *testing.T) {
		trySensorFromDevice(t, config, deviceType, device, serviceLocalId, msg)
	})
}

func TestEvent(t *testing.T) {
	defaultConfig, err := lib.LoadConfigLocation("../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer time.Sleep(10 * time.Second) //wait for docker cleanup
	defer cancel()

	defaultConfig.SensorTopicPattern = "{{.DeviceTypeId}}/{{.LocalDeviceId}}/{{.LocalServiceId}}"

	config, err := server.New(ctx, defaultConfig)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(2 * time.Second)

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
	device := model.Device{}
	msg := `{"level":42}`

	t.Run("create protocol", func(t *testing.T) {
		protocol = createTestProtocol(t, config)
		time.Sleep(10 * time.Second) //wait for cqrs
	})

	t.Run("create device type", func(t *testing.T) {
		deviceType = createTestDeviceType(t, config, protocol, serviceLocalId)
		time.Sleep(10 * time.Second) //wait for cqrs
	})

	t.Run("send mqtt message", func(t *testing.T) {
		sendMqttEvent(t, config, deviceType.Id+"/"+deviceLocalId+"/"+serviceLocalId, msg)
		time.Sleep(10 * time.Second) //wait for cqrs
	})

	t.Run("check device creation", func(t *testing.T) {
		device = checkDevice(t, config, deviceLocalId, deviceType.Id)
	})

	t.Run("check kafka event", func(t *testing.T) {
		trySensorFromDevice(t, config, deviceType, device, serviceLocalId, msg)
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

func trySensorFromDevice(t *testing.T, config lib.Config, deviceType model.DeviceType, device model.Device, serviceLocalId string, msg string) {
	service := model.Service{}
	for _, s := range deviceType.Services {
		if s.LocalId == serviceLocalId {
			service = s
			break
		}
	}
	mux := sync.Mutex{}
	events := []model.Envelope{}
	log.Println("DEBUG CONSUME:", model.ServiceIdToTopic(service.Id))
	consumer, err := kafka.NewConsumer(config.ZookeeperUrl, "testing_"+uuid.NewV4().String(), model.ServiceIdToTopic(service.Id), func(topic string, msg []byte, time time.Time) error {
		mux.Lock()
		defer mux.Unlock()
		resp := model.Envelope{}
		err := json.Unmarshal(msg, &resp)
		if err != nil {
			t.Fatal(err)
			return err
		}
		events = append(events, resp)
		return nil
	}, func(err error, consumer *kafka.Consumer) {
		t.Fatal(err)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer consumer.Stop()

	time.Sleep(10 * time.Second)

	mux.Lock()
	defer mux.Unlock()
	if len(events) == 0 {
		t.Fatal("unexpected event count", events)
	}
	event := events[0]
	if event.DeviceId != device.Id {
		t.Fatal("unexpected envelope", event)
	}
	if event.ServiceId != service.Id {
		t.Fatal("unexpected envelope", event)
	}

	var expected interface{}
	err = json.Unmarshal([]byte("{\"metrics\":"+msg+"}"), &expected)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(event.Value, expected) {
		t.Fatal(event.Value, "\n\n!=\n\n", expected)
	}
}
