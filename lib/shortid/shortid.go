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

package shortid

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
)

func EnsureLongDeviceId(shortDeviceId string) (longDeviceId string, err error) {
	if shortDeviceId == "" {
		return "", nil //is empty -> nothing to decode
	}
	if strings.Contains(shortDeviceId, DEVICE_PREFIX) {
		return shortDeviceId, nil //is already long version
	}

	hexByte, err := base64.RawURLEncoding.DecodeString(shortDeviceId)
	if err != nil {
		return shortDeviceId, err
	}
	if len(hexByte) < 10 {
		return "", errors.New("expected uuid with at least 10 byte")
	}
	uuidStr := bytesToUUidString(hexByte)
	longDeviceId = DEVICE_PREFIX + uuidStr
	return longDeviceId, nil
}

func ShortId(longId string) (shortId string, err error) {
	if longId == "" {
		return "", nil
	}
	parts := strings.Split(longId, ":")
	uuidStr := parts[len(parts)-1]
	uuidHexStr := strings.ReplaceAll(uuidStr, "-", "")
	uuid, err := hex.DecodeString(uuidHexStr)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(uuid), nil
}

func bytesToUUidString(u []byte) string {
	buf := make([]byte, 36)
	hex.Encode(buf[0:8], u[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], u[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], u[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], u[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:], u[10:])
	return string(buf)
}
