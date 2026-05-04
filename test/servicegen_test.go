package test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SENERGY-Platform/device-repository/lib/api"
	"github.com/SENERGY-Platform/device-repository/lib/client"
	devicerepoconfig "github.com/SENERGY-Platform/device-repository/lib/configuration"
	"github.com/SENERGY-Platform/models/go/models"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/configuration"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/topic"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/webhooks/vernemqtt"
	platform_connector_lib "github.com/SENERGY-Platform/platform-connector-lib"
	"github.com/SENERGY-Platform/platform-connector-lib/iot"
)

func TestServiceGen(t *testing.T) {

	ctrl, _, err := client.NewTestClient()
	if err != nil {
		t.Error(err)
		return
	}
	handler := api.GetRouter(devicerepoconfig.Config{}, ctrl)
	server := httptest.NewServer(handler)
	defer server.Close()

	config := configuration.Config{
		DeviceManagerUrl:     server.URL,
		DeviceRepoUrl:        server.URL,
		PermissionsV2Url:     "",
		LogLevel:             "info",
		DeviceExpiration:     60,
		DeviceTypeExpiration: 60,
	}

	iotcache, err := iot.NewCache(
		iot.New(
			config.DeviceManagerUrl,
			config.DeviceRepoUrl,
			config.PermissionsV2Url,
			config.GetLogger(),
		),
		config.DeviceExpiration,
		config.DeviceTypeExpiration,
		config.DeviceTypeExpiration,
		3,
		time.Minute,
	)
	if err != nil {
		t.Error(err)
		return
	}

	connector := platform_connector_lib.Connector{IotCache: iotcache}

	protocol, err, _ := ctrl.SetProtocol(client.InternalAdminToken, models.Protocol{
		Name:    "p",
		Handler: "p",
		ProtocolSegments: []models.ProtocolSegment{{
			Name: "p",
		}},
	})
	if err != nil {
		t.Error(err)
	}

	dc, err, _ := ctrl.SetDeviceClass(client.InternalAdminToken, models.DeviceClass{
		Name: "dc",
	})
	if err != nil {
		t.Error(err)
		return
	}

	dt, err, _ := ctrl.SetDeviceType(client.InternalAdminToken, models.DeviceType{
		Name:          "dt",
		Description:   "dt",
		DeviceClassId: dc.Id,
		Attributes: []models.Attribute{{
			Key:   vernemqtt.GenerateServiceAttr,
			Value: "true",
		}},
		Services: []models.Service{{
			LocalId:     "void/poweron",
			Name:        "s1",
			Description: "s1",
			Interaction: models.EVENT,
			ProtocolId:  protocol.Id,
			Outputs: []models.Content{{
				Serialization:     models.JSON,
				ProtocolSegmentId: protocol.ProtocolSegments[0].Id,
				ContentVariable: models.ContentVariable{
					Name: "pl",
					Type: models.String,
				},
			}},
		}},
	}, client.DeviceTypeUpdateOptions{})
	if err != nil {
		t.Error(err)
		return
	}

	d, err, _ := ctrl.CreateDevice(client.InternalAdminToken, models.Device{
		Name:         "d1",
		LocalId:      "lid/d1",
		DeviceTypeId: dt.Id,
	})
	if err != nil {
		t.Error(err)
		return
	}

	topicParser := topic.New(iotcache, "")

	ErrNoServiceMatchFound := topic.ErrNoServiceMatchFound

	send := func(topic string, payload []byte) {
		device, _, err := topicParser.Parse(client.InternalAdminToken, topic)
		if errors.Is(err, ErrNoServiceMatchFound) {
			err = nil
		}
		if err != nil {
			t.Error(err)
			return
		}
		vernemqtt.TryCreateService(config, &connector, device, topic, payload)
		time.Sleep(100 * time.Millisecond)
	}

	send(fmt.Sprintf("%v/root/services/plainstr", d.Id), []byte("teststring"))
	send(fmt.Sprintf("%v/root/services/jsonstr", d.Id), []byte(`"teststring"`))
	send(fmt.Sprintf("%v/root/services/number", d.Id), []byte("42"))
	send(fmt.Sprintf("%v/root/services/obj", d.Id), []byte(`{"foo":"bar", "number":42, "sub":{"b":true}}`))
	send(fmt.Sprintf("%v/root/services/plainstr2", d.LocalId), []byte("teststring"))
	send(fmt.Sprintf("%v/root/services/jsonstr2", d.LocalId), []byte(`"teststring"`))
	send(fmt.Sprintf("%v/root/services/number2", d.LocalId), []byte("42"))
	send(fmt.Sprintf("%v/root/services/obj2", d.LocalId), []byte(`{"foo":"bar", "number":42, "sub":{"b":true}}`))

	dtAfter, err, _ := ctrl.ReadDeviceType(d.DeviceTypeId, client.InternalAdminToken)
	if err != nil {
		t.Error(err)
		return
	}
	if len(dtAfter.Services) != 9 {
		t.Error("expected 9 services")
	}
	t.Logf("%#v", dtAfter.Services)
	dtJson, _ := json.MarshalIndent(dtAfter, "", "  ")
	t.Log(string(dtJson))
}
