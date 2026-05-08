package test

import (
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
		LocalId:      "lid",
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
			vernemqtt.TryCreateService(config, &connector, device, topic, payload)
			time.Sleep(100 * time.Millisecond)
			return
		}
		if err != nil {
			t.Error(err)
			return
		}
	}

	for range 5 {
		send(fmt.Sprintf("%v/root/services/plainstr", d.Id), []byte("teststring"))
		send(fmt.Sprintf("%v/root/services/jsonstr", d.Id), []byte(`"teststring"`))
		send(fmt.Sprintf("%v/root/services/number", d.Id), []byte("42"))
		send(fmt.Sprintf("%v/root/services/obj", d.Id), []byte(`{"foo":"bar", "number":42, "sub":{"b":true}}`))
		send(fmt.Sprintf("%v/root/services/plainstr2", d.LocalId), []byte("teststring"))
		send(fmt.Sprintf("%v/root/services/jsonstr2", d.LocalId), []byte(`"teststring"`))
		send(fmt.Sprintf("%v/root/services/number2", d.LocalId), []byte("42"))
		send(fmt.Sprintf("%v/root/services/obj2", d.LocalId), []byte(`{"foo":"bar", "number":42, "sub":{"b":true}}`))

		send(fmt.Sprintf("root/services/plainstr3/%v", d.LocalId), []byte("teststring"))
		send(fmt.Sprintf("root/services/jsonstr3/%v", d.LocalId), []byte(`"teststring"`))
		send(fmt.Sprintf("root/services/number3/%v", d.LocalId), []byte("42"))
		send(fmt.Sprintf("root/services/obj3/%v", d.LocalId), []byte(`{"foo":"bar", "number":42, "sub":{"b":true}}`))

		send(fmt.Sprintf("root/%v/services/plainstr4", d.LocalId), []byte("teststring"))
		send(fmt.Sprintf("root/%v/services/jsonstr4", d.LocalId), []byte(`"teststring"`))
		send(fmt.Sprintf("root/%v/services/number4", d.LocalId), []byte("42"))
		send(fmt.Sprintf("root/%v/services/obj4", d.LocalId), []byte(`{"foo":"bar", "number":42, "sub":{"b":true}}`))

		send(fmt.Sprintf("%v/a/b/c", d.LocalId), []byte("42"))
		send(fmt.Sprintf("%v/a/b", d.LocalId), []byte(`"foo"`))

		send(fmt.Sprintf("%v/x/y", d.LocalId), []byte("13"))
		send(fmt.Sprintf("%v/x/y/z", d.LocalId), []byte(`"bar"`))
	}

	dtAfter, err, _ := ctrl.ReadDeviceType(d.DeviceTypeId, client.InternalAdminToken)
	if err != nil {
		t.Error(err)
		return
	}
	if len(dtAfter.Services) != 21 {
		t.Error("expected 21 services, got", len(dtAfter.Services))
	}
	for _, s := range dtAfter.Services {
		t.Log(s.LocalId)
	}

	check := func(topic string, p models.Type, expectedLocalId string) {
		_, service, err := topicParser.Parse(client.InternalAdminToken, topic)
		if err != nil {
			t.Error(topic, err)
			return
		}
		if service.LocalId != expectedLocalId {
			t.Error(topic, service.LocalId, expectedLocalId)
			return
		}
		if service.Outputs[0].ContentVariable.Type != p {
			t.Error(topic, service.Outputs[0].ContentVariable.Type, p)
			return
		}
	}

	check(fmt.Sprintf("%v/root/services/plainstr", d.Id), models.String, "/root/services/plainstr")
	check(fmt.Sprintf("%v/root/services/jsonstr", d.Id), models.String, "/root/services/jsonstr")
	check(fmt.Sprintf("%v/root/services/number", d.Id), models.Float, "/root/services/number")
	check(fmt.Sprintf("%v/root/services/obj", d.Id), models.Structure, "/root/services/obj")
	check(fmt.Sprintf("%v/root/services/plainstr2", d.LocalId), models.String, "/root/services/plainstr2")
	check(fmt.Sprintf("%v/root/services/jsonstr2", d.LocalId), models.String, "/root/services/jsonstr2")
	check(fmt.Sprintf("%v/root/services/number2", d.LocalId), models.Float, "/root/services/number2")
	check(fmt.Sprintf("%v/root/services/obj2", d.LocalId), models.Structure, "/root/services/obj2")
	check(fmt.Sprintf("root/services/plainstr3/%v", d.LocalId), models.String, "root/services/plainstr3/")
	check(fmt.Sprintf("root/services/jsonstr3/%v", d.LocalId), models.String, "root/services/jsonstr3/")
	check(fmt.Sprintf("root/services/number3/%v", d.LocalId), models.Float, "root/services/number3/")
	check(fmt.Sprintf("root/services/obj3/%v", d.LocalId), models.Structure, "root/services/obj3/")
	check(fmt.Sprintf("root/%v/services/plainstr4", d.LocalId), models.String, "/services/plainstr4")
	check(fmt.Sprintf("root/%v/services/jsonstr4", d.LocalId), models.String, "/services/jsonstr4")
	check(fmt.Sprintf("root/%v/services/number4", d.LocalId), models.Float, "/services/number4")
	check(fmt.Sprintf("root/%v/services/obj4", d.LocalId), models.Structure, "/services/obj4")
	check(fmt.Sprintf("%v/a/b/c", d.LocalId), models.Float, "/a/b/c")
	check(fmt.Sprintf("%v/a/b", d.LocalId), models.String, "/a/b")
	check(fmt.Sprintf("%v/x/y", d.LocalId), models.Float, "/x/y")
	check(fmt.Sprintf("%v/x/y/z", d.LocalId), models.String, "/x/y/z")

}
