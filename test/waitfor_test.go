/*
 * Copyright 2026 InfAI (CC SES)
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

package test

import (
	"testing"
	"time"
)

// waitFor polls check until it returns nil and reports the last error if that
// does not happen within timeout. It replaces "sleep for a guessed duration,
// then assert once": the assertion is unchanged, but a message that arrives
// later than the guess no longer fails the test, and a message that arrives
// earlier no longer costs the full sleep.
//
// settle covers assertions that also assert the absence of messages. The
// condition has to still hold after settle has elapsed, so it cannot be
// satisfied by a snapshot taken right before an unexpected message arrives.
// Pass 0 where only the presence of a message is asserted.
//
// The return value reports whether the condition was met, so callers can skip
// follow-up assertions that would only add noise.
func waitFor(t *testing.T, timeout time.Duration, settle time.Duration, check func() error) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		err := check()
		if err == nil && settle > 0 {
			time.Sleep(settle)
			err = check()
		}
		if err == nil {
			return true
		}
		if time.Now().After(deadline) {
			t.Errorf("condition not met within %v: %v", timeout, err)
			return false
		}
		time.Sleep(250 * time.Millisecond)
	}
}
