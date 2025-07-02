package test

import (
	"context"
	"encoding/json"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/configuration"
	"github.com/SENERGY-Platform/mqtt-platform-connector/test/client"
	"github.com/SENERGY-Platform/mqtt-platform-connector/test/server"
	"github.com/SENERGY-Platform/platform-connector-lib/kafka"
	"github.com/SENERGY-Platform/platform-connector-lib/model"
	"github.com/google/uuid"
	"log"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestEventWithoutProvisioningPw4(t *testing.T) {
	testEventWithoutProvisioning(t, "password", client.MQTT4)
}

func TestEventWithoutProvisioningPw5(t *testing.T) {
	testEventWithoutProvisioning(t, "password", client.MQTT5)
}

func TestEventWithoutProvisioningCert4(t *testing.T) {
	t.Skip("expired certificate") //TODO: fix
	testEventWithoutProvisioning(t, "certificate", client.MQTT4)
}

func TestEventWithoutProvisioningCert5(t *testing.T) {
	t.Skip("expired certificate") //TODO: fix
	testEventWithoutProvisioning(t, "certificate", client.MQTT5)
}

func testEventWithoutProvisioning(t *testing.T, authMethod string, mqttVersion client.MqttVersion) {
	defaultConfig, err := configuration.Load("../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	defaultConfig.InitTopics = true
	defaultConfig.MqttAuthMethod = authMethod
	if mqttVersion == client.MQTT5 {
		defaultConfig.MqttVersion = "5"
	}

	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, clientBroker, err := server.New(ctx, wg, defaultConfig)
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
	deviceId := "urn:infai:ses:device:b2d95adb-1b40-4b0f-bb98-0fabe18d935e"
	serviceLocalId := "testservice1"
	serviceId := "urn:infai:ses:service:efed3e07-e738-445f-8a4f-847b87688506"
	deviceType := model.DeviceType{}
	protocol := model.Protocol{}
	device := model.Device{}
	msg := `{"level":42}`

	t.Run("create protocol", func(t *testing.T) {
		protocol = createTestProtocol(t, config)
		time.Sleep(10 * time.Second) //wait for cqrs
	})

	t.Run("create device type", func(t *testing.T) {
		deviceType = createTestDeviceType(t, config, protocol, serviceLocalId, serviceId)
		time.Sleep(10 * time.Second) //wait for cqrs
	})

	t.Run("create device", func(t *testing.T) {
		device = createTestDevice(t, config, deviceType, deviceLocalId, deviceId)
		time.Sleep(10 * time.Second) //wait for cqrs
	})

	t.Run("send mqtt message", func(t *testing.T) {
		sendMqttEvent(t, clientBroker, "senergy/"+device.Id+"/"+serviceLocalId, msg, authMethod, mqttVersion)
		time.Sleep(10 * time.Second) //wait for cqrs
	})

	t.Run("check kafka event", func(t *testing.T) {
		trySensorFromDevice(t, config, ctx, deviceType, device, serviceLocalId, msg)
	})
}

func TestEventPlainTextPw4(t *testing.T) {
	testEventPlainText(t, "password", client.MQTT4)
}

func TestEventPlainTextPw5(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	testEventPlainText(t, "password", client.MQTT5)
}

func TestEventPlainTextCert4(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	testEventPlainText(t, "certificate", client.MQTT4)
}

func TestEventPlainTextCert5(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	testEventPlainText(t, "certificate", client.MQTT5)
}

func testEventPlainText(t *testing.T, authMethod string, mqttVersion client.MqttVersion) {
	defaultConfig, err := configuration.Load("../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	defaultConfig.InitTopics = true
	defaultConfig.PublishToPostgres = true
	defaultConfig.MqttAuthMethod = authMethod
	if mqttVersion == client.MQTT5 {
		defaultConfig.MqttVersion = "5"
	}

	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, clientBroker, err := server.New(ctx, wg, defaultConfig)
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
	deviceId := "urn:infai:ses:device:00dbdd68-7a57-41fc-a959-1f04892b5c5e"
	serviceLocalId := "testservice1"
	serviceId := "urn:infai:ses:service:d2ada448-9e3f-408a-ab5c-b3040ab99470"
	deviceType := model.DeviceType{}
	protocol := model.Protocol{}
	device := model.Device{}
	msg := `100 watt`

	t.Run("create protocol", func(t *testing.T) {
		protocol = createTestProtocol(t, config)
		time.Sleep(10 * time.Second) //wait for cqrs
	})

	t.Run("create device type", func(t *testing.T) {
		deviceType = createTestDeviceTypeWithTextPayload(t, config, protocol, serviceLocalId, serviceId)
		time.Sleep(10 * time.Second) //wait for cqrs
	})

	t.Run("create device", func(t *testing.T) {
		device = createTestDevice(t, config, deviceType, deviceLocalId, deviceId)
		time.Sleep(10 * time.Second) //wait for cqrs
	})

	t.Run("send mqtt message", func(t *testing.T) {
		sendMqttEvent(t, clientBroker, "senergy/"+device.Id+"/"+serviceLocalId, msg, authMethod, mqttVersion)
		time.Sleep(10 * time.Second) //wait for cqrs
	})

	t.Run("check kafka event", func(t *testing.T) {
		trySensorFromDevice(t, config, ctx, deviceType, device, serviceLocalId, "\""+msg+"\"")
	})
}

func sendMqttEvent(t *testing.T, broker string, topic string, msg string, authMethod string, mqttVersion client.MqttVersion) {
	mqtt, err := client.New(broker, "sepl", "sepl", uuid.NewString(), authMethod, mqttVersion, true, true)
	if err != nil {
		t.Error(err)
		return
	}
	defer mqtt.Stop()
	err = mqtt.PublishRaw(topic, []byte(msg), 2)
	if err != nil {
		t.Error(err)
		return
	}
}

func trySensorFromDevice(t *testing.T, config configuration.Config, ctx context.Context, deviceType model.DeviceType, device model.Device, serviceLocalId string, msg string) {
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
	err := kafka.NewConsumer(ctx, kafka.ConsumerConfig{
		KafkaUrl: config.KafkaUrl,
		GroupId:  "testing_" + uuid.NewString(),
		Topic:    model.ServiceIdToTopic(service.Id),
		MinBytes: 1000,
		MaxBytes: 1000000,
		MaxWait:  100 * time.Millisecond,
	}, func(topic string, msg []byte, _ time.Time) error {
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
	}, func(err error) {
		t.Error(err)
	})

	time.Sleep(20 * time.Second)

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
	err = json.Unmarshal([]byte("{\"payload\":"+msg+"}"), &expected)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(event.Value, expected) {
		t.Fatal(event.Value, "\n\n!=\n\n", expected)
	}
	t.Log(event.Value, "\n\n!=\n\n", expected)
}
