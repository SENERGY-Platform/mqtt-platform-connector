package test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/configuration"
	"github.com/SENERGY-Platform/mqtt-platform-connector/test/helper"
	"github.com/SENERGY-Platform/platform-connector-lib/model"
	"github.com/SENERGY-Platform/platform-connector-lib/security"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"
)

func createTestProtocol(t *testing.T, config configuration.Config) model.Protocol {
	protocol := model.Protocol{}
	err := helper.AdminJwt.PostJSON(config.DeviceManagerUrl+"/protocols", model.Protocol{
		Name:             config.Protocol,
		Handler:          config.Protocol,
		ProtocolSegments: []model.ProtocolSegment{{Name: "payload"}},
	}, &protocol)
	if err != nil {
		t.Error(err)
		return model.Protocol{}
	}
	return protocol
}

func createTestDeviceType(t *testing.T, config configuration.Config, protocol model.Protocol, serviceLocalId string, serviceId string) (result model.DeviceType) {
	if reflect.DeepEqual(protocol, model.Protocol{}) {
		t.Error("invalid protocol")
		return model.DeviceType{}
	}
	err := helper.AdminJwt.PostJSON(config.DeviceManagerUrl+"/device-types", model.DeviceType{
		Name: "testDeviceType",
		Services: []model.Service{
			{
				Id:          serviceId,
				Name:        serviceLocalId,
				LocalId:     serviceLocalId,
				Description: serviceLocalId,
				ProtocolId:  protocol.Id,
				Outputs: []model.Content{
					{
						ProtocolSegmentId: protocol.ProtocolSegments[0].Id,
						Serialization:     "json",
						ContentVariable: model.ContentVariable{
							Name: "payload",
							Type: model.Structure,
							SubContentVariables: []model.ContentVariable{
								{
									Name: "level",
									Type: model.Integer,
								},
							},
						},
					},
				},
			},
			{
				Id:          serviceId + "_2",
				Name:        serviceLocalId + "_2",
				LocalId:     serviceLocalId + "_2",
				Description: serviceLocalId + "_2",
				ProtocolId:  protocol.Id,
				Outputs: []model.Content{
					{
						ProtocolSegmentId: protocol.ProtocolSegments[0].Id,
						Serialization:     "json",
						ContentVariable: model.ContentVariable{
							Name: "payload",
							Type: model.Structure,
							SubContentVariables: []model.ContentVariable{
								{
									Name: "level",
									Type: model.Integer,
								},
							},
						},
					},
				},
			},
		},
	}, &result)
	if err != nil {
		t.Fatal(err)
	}
	if result.Id == "" {
		t.Fatal("unexpected result", result)
	}
	return
}

func createTestDeviceTypeWithTextPayload(t *testing.T, config configuration.Config, protocol model.Protocol, serviceLocalId string, serviceId string) (result model.DeviceType) {
	if reflect.DeepEqual(protocol, model.Protocol{}) {
		t.Error("invalid protocol")
		return model.DeviceType{}
	}
	err := helper.AdminJwt.PostJSON(config.DeviceManagerUrl+"/device-types", model.DeviceType{
		Name: "testDeviceType",
		Services: []model.Service{
			{
				Id:          serviceId,
				Name:        serviceLocalId,
				LocalId:     serviceLocalId,
				Description: serviceLocalId,
				ProtocolId:  protocol.Id,
				Outputs: []model.Content{
					{
						ProtocolSegmentId: protocol.ProtocolSegments[0].Id,
						Serialization:     "json",
						ContentVariable: model.ContentVariable{
							Name: "payload",
							Type: model.String,
						},
					},
				},
			},
		},
	}, &result)
	if err != nil {
		t.Fatal(err)
	}
	if result.Id == "" {
		t.Fatal("unexpected result", result)
	}
	return
}

func createTestDevice(t *testing.T, config configuration.Config, dt model.DeviceType, deviceLocalId string, deviceId string) (result model.Device) {
	if reflect.DeepEqual(dt, model.DeviceType{}) {
		t.Error("invalid device-type")
		return
	}
	err := helper.AdminJwt.PostJSON(config.DeviceManagerUrl+"/devices", model.Device{
		Id:           deviceId,
		LocalId:      deviceLocalId,
		Name:         "test-device",
		DeviceTypeId: dt.Id,
	}, &result)
	if err != nil {
		t.Fatal(err)
	}
	if result.Id == "" {
		t.Fatal("unexpected result", result)
	}
	return
}

func createTestDeviceWithUserToken(t *testing.T, token string, config configuration.Config, dt model.DeviceType, deviceLocalId string, deviceId string) (result model.Device) {
	if reflect.DeepEqual(dt, model.DeviceType{}) {
		t.Error("invalid device-type")
		return
	}
	err := security.JwtToken(token).PostJSON(config.DeviceManagerUrl+"/devices", model.Device{
		Id:           deviceId,
		LocalId:      deviceLocalId,
		Name:         "test-device",
		DeviceTypeId: dt.Id,
	}, &result)
	if err != nil {
		t.Fatal(err)
	}
	if result.Id == "" {
		t.Fatal("unexpected result", result)
	}
	return
}

func checkDevice(t *testing.T, config configuration.Config, deviceLocalId string, deviceTypeId string) (result model.Device) {
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	req, err := http.NewRequest("GET", config.DeviceManagerUrl+"/local-devices/"+url.PathEscape(deviceLocalId), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	req.WithContext(ctx)
	req.Header.Set("Authorization", string(helper.AdminJwt))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		responseBody, _ := ioutil.ReadAll(resp.Body)
		err = errors.New(resp.Status + ": " + string(responseBody))
		t.Fatal(err)
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		t.Fatal(err)
	}
	if result.LocalId != deviceLocalId {
		t.Fatal("unexpected result", result)
	}
	if result.DeviceTypeId != deviceTypeId {
		t.Fatal("unexpected result", result)
	}
	return
}
