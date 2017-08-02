// Copyright 2014 Orchestrate, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dvr

import (
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/liquidgecka/testlib"
)

func TestMatcher(t *testing.T) {
	T := testlib.NewT(t)
	defer T.Finish()

	left := &RequestResponse{}
	right := &RequestResponse{
		Request: &http.Request{
			Method: "GET",
			URL: &url.URL{
				Scheme:   "http",
				Opaque:   "opaque",
				User:     url.UserPassword("user", "password"),
				Host:     "host",
				Path:     "path",
				RawQuery: "raw",
				Fragment: "fragment",
			},
			Header: http.Header(map[string][]string{
				"header1": []string{"value1", "value2"},
			}),
			Trailer: http.Header(map[string][]string{
				"header2": []string{"value3", "value4"},
			}),
		},
		RequestBody: []byte("body"),
	}

	// Test 1: nil values (returns false
	T.Equal(matcher(nil, nil), false)
	T.Equal(matcher(left, right), false)

	// Test 2: nil url.
	left.Request = &http.Request{}
	T.Equal(matcher(left, right), false)

	// Test 3: Different Schemes
	left.Request = &http.Request{
		URL: &url.URL{
			Scheme: "NOT_GET",
		},
	}
	T.Equal(matcher(left, right), false)
	left.Request.URL.Scheme = right.Request.URL.Scheme

	// Test 4: Different Opaque values.
	left.Request.URL.Opaque = "NOT_OPAQUE"
	T.Equal(matcher(left, right), false)
	left.Request.URL.Opaque = right.Request.URL.Opaque

	// Test 5: Different Host values.
	left.Request.URL.Host = "NOT_HOST"
	T.Equal(matcher(left, right), false)
	left.Request.URL.Host = right.Request.URL.Host

	// Test 6: Different Path values.
	left.Request.URL.Path = "NOT_PATH"
	T.Equal(matcher(left, right), false)
	left.Request.URL.Path = right.Request.URL.Path

	// Test 7: Different RawQuery values.
	left.Request.URL.RawQuery = "NOT_RAW_QUERY"
	T.Equal(matcher(left, right), false)
	left.Request.URL.RawQuery = right.Request.URL.RawQuery

	// Test 8: Different Fragment values.
	left.Request.URL.Fragment = "NOT_FRAGMENT"
	T.Equal(matcher(left, right), false)
	left.Request.URL.Fragment = right.Request.URL.Fragment

	// Test 9: Left URL.User == nil
	T.Equal(matcher(left, right), false)

	// Test 10: Right URL.User = nil
	left.Request.URL.User = right.Request.URL.User
	right.Request.URL.User = nil
	T.Equal(matcher(left, right), false)

	// Test 11: URL.User.String() is different.
	right.Request.URL.User = url.UserPassword("not_user", "not_password")
	T.Equal(matcher(left, right), false)
	right.Request.URL.User = left.Request.URL.User

	// Test 12: RequestBody values differ.
	left.RequestBody = []byte("NOT_THE_SAME")
	T.Equal(matcher(left, right), false)
	left.RequestBody = right.RequestBody

	// Test 13: Headers are different.
	left.Request.Header = http.Header(map[string][]string{
		"header1": []string{"value1", "value2_XXX"},
	})
	T.Equal(matcher(left, right), false)
	left.Request.Header = right.Request.Header

	// Test 14: Trailers are different.
	left.Request.Trailer = http.Header(map[string][]string{
		"header2": []string{"value1", "value2_XXX"},
	})
	T.Equal(matcher(left, right), false)
	left.Request.Trailer = right.Request.Trailer

	// Test 15: Successful match.
	T.Equal(matcher(left, right), true)

	// Test 16: Second try fails.
	T.Equal(matcher(left, right), false)
}

func TestBadVersion(t *testing.T) {
	T := testlib.NewT(t)
	defer T.Finish()

	fd := T.TempFile()
	isSetup = sync.Once{}
	record = false
	replay = true
	passThrough = false
	DefaultReplay = false
	replayTimestamp = time.Time{}
	fileName = fd.Name()

	// Write all zeros to the temp file which will be version 0.
	_, err := fd.Write([]byte{0, 0, 0, 0})
	T.ExpectSuccess(err)
	T.ExpectSuccess(fd.Close())

	// Now check to see if the setup function fails for the right reasons.
	defer func() {
		err := recover()
		if err == nil {
			T.Fatalf("Expected panic not returned.")
		} else if err2, ok := err.(error); !ok {
			T.Fatalf("Unknown panic return: %#v\n", err)
		} else {
			T.ExpectErrorMessage(err2, "Unknown version: 0")
		}
	}()
	rt := roundTripper{}
	rt.replaySetup()
}
