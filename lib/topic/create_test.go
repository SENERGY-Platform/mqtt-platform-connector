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

import "testing"

const shortDeviceIdExample = "a9B7ddfMShqI26yT9hqnsw"
const longDeviceIdExample = "urn:infai:ses:device:6bd07b75-d7cc-4a1a-88db-ac93f61aa7b3"

func TestCreate(t *testing.T) {
	topic := New(nil, "{{.DeviceId}}/cmnd/{{.LocalServiceId}}")

	t.Run(testTopicCreate(topic, "{{.DeviceId}}/temperature", longDeviceIdExample+"/temperature"))
	t.Run(testTopicCreate(topic, "cmd/{{.DeviceId}}/temperature", "cmd/"+longDeviceIdExample+"/temperature"))
	t.Run(testTopicCreate(topic, "cmd/temperature/{{.DeviceId}}", "cmd/temperature/"+longDeviceIdExample))

	t.Run(testTopicCreate(topic, "{{.ShortDeviceId}}/temperature", shortDeviceIdExample+"/temperature"))
	t.Run(testTopicCreate(topic, "cmd/{{.ShortDeviceId}}/temperature", "cmd/"+shortDeviceIdExample+"/temperature"))
	t.Run(testTopicCreate(topic, "cmd/temperature/{{.ShortDeviceId}}", "cmd/temperature/"+shortDeviceIdExample))

	t.Run(testTopicCreate(topic, "temperature", longDeviceIdExample+"/cmnd/temperature"))
	t.Run(testTopicCreate(topic, "temperature/celsius", longDeviceIdExample+"/cmnd/temperature/celsius"))
}

func TestCreateDefaultShort(t *testing.T) {
	topic := New(nil, "{{.ShortDeviceId}}/cmnd/{{.LocalServiceId}}")

	t.Run(testTopicCreate(topic, "{{.DeviceId}}/temperature", longDeviceIdExample+"/temperature"))
	t.Run(testTopicCreate(topic, "cmd/{{.DeviceId}}/temperature", "cmd/"+longDeviceIdExample+"/temperature"))
	t.Run(testTopicCreate(topic, "cmd/temperature/{{.DeviceId}}", "cmd/temperature/"+longDeviceIdExample))

	t.Run(testTopicCreate(topic, "{{.ShortDeviceId}}/temperature", shortDeviceIdExample+"/temperature"))
	t.Run(testTopicCreate(topic, "cmd/{{.ShortDeviceId}}/temperature", "cmd/"+shortDeviceIdExample+"/temperature"))
	t.Run(testTopicCreate(topic, "cmd/temperature/{{.ShortDeviceId}}", "cmd/temperature/"+shortDeviceIdExample))

	t.Run(testTopicCreate(topic, "temperature", shortDeviceIdExample+"/cmnd/temperature"))
	t.Run(testTopicCreate(topic, "temperature/celsius", shortDeviceIdExample+"/cmnd/temperature/celsius"))
}

func testTopicCreate(topic *Topic, localServiceId string, expectedTopic string) (string, func(t *testing.T)) {
	return localServiceId, func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Error("recover", r)
			}
		}()
		actual, err := topic.Create(longDeviceIdExample, localServiceId)
		if err != nil {
			t.Error(err)
			return
		}
		if actual != expectedTopic {
			t.Error(actual, expectedTopic)
			return
		}
	}
}
