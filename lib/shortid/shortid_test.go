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

import "testing"

func TestShortening(t *testing.T) {
	t.Run(testShortening("urn:infai:ses:device:6bd07b75-d7cc-4a1a-88db-ac93f61aa7b3", "a9B7ddfMShqI26yT9hqnsw"))
	t.Run(testShorteningExpectError("6bd07b75-d7cc-4a1a-88db-ac93f61aa7b3"))
	t.Run(testShortening("", ""))
}

func TestEnsureLongDeviceId(t *testing.T) {
	t.Run(testEnsureLongDeviceId("a9B7ddfMShqI26yT9hqnsw", "urn:infai:ses:device:6bd07b75-d7cc-4a1a-88db-ac93f61aa7b3"))
	t.Run(testEnsureLongDeviceId("urn:infai:ses:device:6bd07b75-d7cc-4a1a-88db-ac93f61aa7b3", "urn:infai:ses:device:6bd07b75-d7cc-4a1a-88db-ac93f61aa7b3"))
	t.Run(testEnsureLongDeviceId("", ""))
}

func testEnsureLongDeviceId(short string, expectedLong string) (string, func(t *testing.T)) {
	return short, func(t *testing.T) {
		long, err := EnsureLongDeviceId(short)
		if err != nil {
			t.Error(err)
			return
		}
		if long != expectedLong {
			t.Error(long, len(long), expectedLong)
			return
		}
	}
}

func testShortening(long string, expectedShort string) (string, func(t *testing.T)) {
	return long, func(t *testing.T) {
		short, err := ShortId(long)
		if err != nil {
			t.Error(err)
			return
		}
		if expectedShort != short {
			t.Error(short, len(short), expectedShort)
			return
		}
	}
}

func testShorteningExpectError(long string) (string, func(t *testing.T)) {
	return long, func(t *testing.T) {
		_, err := ShortId(long)
		if err == nil {
			t.Error(err)
			return
		}
	}
}
