package vernemqtt

import (
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/SENERGY-Platform/models/go/models"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/configuration"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/shortid"
	"github.com/SENERGY-Platform/permissions-v2/pkg/client"
	platform_connector_lib "github.com/SENERGY-Platform/platform-connector-lib"
	"github.com/SENERGY-Platform/platform-connector-lib/model"
	"github.com/SENERGY-Platform/platform-connector-lib/security"
)

const GenerateServiceAttr = "senergy/mqtt-generate-services"

func TryCreateService(config configuration.Config, connector *platform_connector_lib.Connector, device model.Device, topic string, payload []byte) {
	token := security.JwtToken(client.InternalAdminToken)
	dt, err := connector.IotCache.GetDeviceType(token, device.DeviceTypeId)
	if err != nil {
		config.GetLogger().Error("unable to get device type", "error", err, "device-type-id", device.DeviceTypeId)
		return
	}
	if !slices.ContainsFunc(dt.Attributes, func(a models.Attribute) bool {
		return a.Key == GenerateServiceAttr && strings.ToLower(strings.TrimSpace(a.Value)) == "true"
	}) {
		return //device-type does not wish to let services be generated --> ignore
	}

	short, err := shortid.ShortId(device.Id)
	if err != nil {
		config.GetLogger().Error("unable to shorten device id", "error", err, "device-id", device.Id)
		return
	}

	var parts []string
	if strings.Contains(topic, device.Id) {
		parts = strings.Split(topic, device.Id)
	} else if strings.Contains(topic, short) {
		parts = strings.Split(topic, short)
	} else if strings.Contains(topic, device.LocalId) {
		parts = strings.Split(topic, device.LocalId)
	}

	parts = slices.DeleteFunc(parts, func(a string) bool {
		if a == "" {
			return true
		}
		for _, service := range dt.Services {
			if service.LocalId == a {
				return true
			}
		}
		return false
	})
	if len(parts) == 0 {
		config.GetLogger().Warn("unable to find valid/unused topic part to generate new service", "topic", topic, "device-id", device.Id)
		return
	}
	serviceLocalId := slices.MaxFunc(parts, func(a, b string) int {
		return len(a) - len(b)
	})

	service := models.Service{
		LocalId:     serviceLocalId,
		Name:        serviceLocalId,
		Interaction: models.EVENT,
		Outputs: []models.Content{{
			Serialization: models.JSON,
		}},
	}

	service.Outputs[0].ContentVariable, err = jsonValueToContentVariable("payload", payload)
	if errors.Is(err, NotJsonErr) {
		service.Outputs = []models.Content{{
			ContentVariable: models.ContentVariable{
				Name: "payload",
				Type: models.String,
			},
			Serialization: models.PlainText,
		}}
		err = nil
	}
	if err != nil {
		config.GetLogger().Error("unable to generate content variable", "error", err, "device-id", device.Id, "topic", topic, "payload", string(payload))
		return
	}

	for _, s := range dt.Services {
		service.ProtocolId = s.ProtocolId
		for _, o := range s.Outputs {
			service.Outputs[0].ProtocolSegmentId = o.ProtocolSegmentId
			break
		}
		if service.ProtocolId != "" && service.Outputs[0].ProtocolSegmentId != "" {
			break
		}
	}
	if service.ProtocolId == "" || service.Outputs[0].ProtocolSegmentId == "" {
		config.GetLogger().Warn("unable to find protocol-id and protocol-segment-id for new service", "device-type-id", device.DeviceTypeId, "topic", topic)
		return
	}

	dt.Services = append(dt.Services, service)

	_, err = connector.IotCache.UpdateDeviceType(token, dt)
	if err != nil {
		config.GetLogger().Error("unable to update device type with generated service", "error", err, "device-type-id", device.DeviceTypeId)
		return
	}

}

var NotJsonErr = errors.New("not json")

func jsonValueToContentVariable(name string, data []byte) (result models.ContentVariable, err error) {
	var obj interface{}
	err = json.Unmarshal(data, &obj)
	if err != nil {
		return result, errors.Join(err, NotJsonErr)
	}
	return valueToContentVariable(name, obj), nil
}

func valueToContentVariable(name string, data interface{}) (result models.ContentVariable) {
	result = models.ContentVariable{
		Name:                name,
		IsVoid:              false,
		SubContentVariables: nil,
		OmitEmpty:           true,
	}
	switch v := data.(type) {
	case string:
		result.Type = models.String
	case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		result.Type = models.Float
	case bool:
		result.Type = models.Boolean
	case []interface{}:
		result.Type = models.List
		sub := []models.ContentVariable{}
		for i, e := range v {
			sub = append(sub, valueToContentVariable(strconv.Itoa(i), e))
		}
		result.SubContentVariables = sub
	case map[string]interface{}:
		result.Type = models.Structure
		sub := []models.ContentVariable{}
		for k, e := range v {
			sub = append(sub, valueToContentVariable(k, e))
		}
		result.SubContentVariables = sub
	default:
		result.Type = models.Type(reflect.TypeOf(v).String())
	}
	return result
}
