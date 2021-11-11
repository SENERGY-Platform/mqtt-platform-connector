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

package connectionlog

type VoidType struct{}

var Void = &VoidType{}

func (this *VoidType) Subscribe(client string, topic string, deviceId string) {
	return
}

func (this *VoidType) Unsubscribe(client string, topic string, deviceId string) {
	return
}

func (this *VoidType) Disconnect(client string) {
	return
}

func (this *VoidType) Connect(client string) {
	return
}

func (this *VoidType) SetCleanSession(id string, session bool) {
	return
}
