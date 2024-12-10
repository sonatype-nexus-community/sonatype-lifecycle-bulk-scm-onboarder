/**
 * Copyright (c) 2019-present Sonatype, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package scm

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScmOrganisationSafeName(t *testing.T) {

	cases := []struct {
		input    string
		expected string
	}{
		{
			input:    "Name",
			expected: "Name",
		},
		{
			input:    "Name<",
			expected: "Name-",
		},
		{
			input:    "Name[something]",
			expected: "Name-something-",
		},
		{
			input:    "Name  WithDoubleSpace",
			expected: "Name--WithDoubleSpace",
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("TestScmOrganisationSafeName-%d-%s", i, tc.input), func(t *testing.T) {
			o := Organization{Name: tc.input}
			assert.Equal(t, tc.expected, o.SafeName())
		})
	}
}
