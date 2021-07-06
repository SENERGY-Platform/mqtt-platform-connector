package test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib"
	"github.com/SENERGY-Platform/mqtt-platform-connector/test/server"
	"github.com/SENERGY-Platform/platform-connector-lib/kafka"
	"github.com/SENERGY-Platform/platform-connector-lib/model"
	"github.com/SENERGY-Platform/platform-connector-lib/psql"
	uuid "github.com/satori/go.uuid"
	"log"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestEventWithoutProvisioning(t *testing.T) {
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
		sendMqttEvent(t, config, "senergy/"+device.Id+"/"+serviceLocalId, msg)
		time.Sleep(10 * time.Second) //wait for cqrs
	})

	t.Run("check kafka event", func(t *testing.T) {
		trySensorFromDevice(t, config, ctx, deviceType, device, serviceLocalId, msg)
	})
}

func TestEventPlainText(t *testing.T) {
	defaultConfig, err := lib.LoadConfigLocation("../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	defaultConfig.PublishToPostgres = true

	ctx, cancel := context.WithCancel(context.Background())
	defer time.Sleep(10 * time.Second) //wait for docker cleanup
	defer cancel()

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
		sendMqttEvent(t, config, "senergy/"+device.Id+"/"+serviceLocalId, msg)
		time.Sleep(10 * time.Second) //wait for cqrs
	})

	t.Run("check kafka event", func(t *testing.T) {
		trySensorFromDevice(t, config, ctx, deviceType, device, serviceLocalId, "\""+msg+"\"")
	})

	t.Run("check written to postgres", func(t *testing.T) {
		psqlconn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", config.PostgresHost,
			config.PostgresPort, config.PostgresUser, config.PostgresPw, config.PostgresDb)

		// open database
		db, err := sql.Open("postgres", psqlconn)
		if err != nil {
			t.Fatal("could not establish db")
		}
		err = db.Ping()
		if err != nil {
			t.Fatal("could not connect to db")
		}
		shortServiceId1, err := psql.ShortenId(serviceId)
		if err != nil {
			t.Fatal(err)
		}
		shortDeviceId, err := psql.ShortenId(deviceId)
		if err != nil {
			t.Fatal(err)
		}
		query := "SELECT * FROM \"device:" + shortDeviceId + "_service:" + shortServiceId1 + "\";"
		resp, err := db.Query(query)
		if err != nil {
			t.Fatal(err)
		}
		if !resp.Next() {
			t.Fatal("Event not written to Postgres!")
		}
		var dt time.Time
		var payload string
		err = resp.Scan(&dt, &payload)
		if err != nil {
			t.Fatal(err)
		}
		if payload != "100 watt" {
			t.Fatal("Invalid values written to postgres")
		}
		if resp.Next() {
			t.Fatal("Too many events written to Postgres!")
		}
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

func trySensorFromDevice(t *testing.T, config lib.Config, ctx context.Context, deviceType model.DeviceType, device model.Device, serviceLocalId string, msg string) {
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
	err := kafka.NewConsumer(ctx, config.KafkaUrl, "testing_"+uuid.NewV4().String(), model.ServiceIdToTopic(service.Id), func(topic string, msg []byte, time time.Time) error {
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
