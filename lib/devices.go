package lib

import (
	platform_connector_lib "github.com/SENERGY-Platform/platform-connector-lib"
	"github.com/SENERGY-Platform/platform-connector-lib/model"
	"github.com/SENERGY-Platform/platform-connector-lib/security"
)

func ensureDeviceExistence(token security.JwtToken, connector *platform_connector_lib.Connector, deviceTypeId string, localDeviceId string) error {
	_, err := connector.IotCache.WithToken(token).EnsureLocalDeviceExistence(model.Device{
		LocalId:      localDeviceId,
		Name:         localDeviceId,
		DeviceTypeId: deviceTypeId,
	})
	return err
}
