/*
 * Copyright 2020 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package topic

import (
	"bytes"
	"github.com/SENERGY-Platform/mqtt-platform-connector/lib/shortid"
	"text/template"
)

func (this *Topic) Create(deviceId string, localServiceId string) (topic string, err error) {
	shortDeviceId, err := shortid.ShortId(deviceId)
	if err != nil {
		return topic, err
	}
	topic, err = this.createTopicFromLocalServiceId(deviceId, shortDeviceId, localServiceId)
	if err != nil {
		return topic, err
	}
	if topic == localServiceId {
		topic, err = this.createTopicFromDefaultPattern(deviceId, shortDeviceId, localServiceId)
		if err != nil {
			return topic, err
		}
	}
	return topic, nil
}

func (this *Topic) createTopicFromLocalServiceId(deviceId string, shortDeviceId, localServiceId string) (topic string, err error) {
	var temp bytes.Buffer
	err = template.Must(template.New("").Parse(localServiceId)).Execute(&temp, map[string]string{
		"DeviceId":      deviceId,
		"ShortDeviceId": shortDeviceId,
	})
	if err != nil {
		return
	}
	return temp.String(), nil
}

func (this *Topic) createTopicFromDefaultPattern(deviceId string, shortDeviceId string, localServiceId string) (result string, err error) {
	var temp bytes.Buffer
	err = template.Must(template.New("").Parse(this.defaultActuatorPattern)).Execute(&temp, map[string]string{
		"DeviceId":       deviceId,
		"ShortDeviceId":  shortDeviceId,
		"LocalServiceId": localServiceId,
	})
	if err != nil {
		return
	}
	return temp.String(), nil
}
