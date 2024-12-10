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

func TestSafeBranchNameNil(t *testing.T) {
	var nil string
	assert.Equal(t, false, safeBranchName(nil))
}

func TestSafeBranchName(t *testing.T) {

	cases := []struct {
		input     string
		permitted bool
	}{
		{
			input:     "main",
			permitted: true,
		},
		{
			input:     "master",
			permitted: true,
		},
		{
			input:     "with(bracket",
			permitted: false,
		},
		{
			input:     "with&mpersand",
			permitted: false,
		},
		{
			input:     "ma?in",
			permitted: false,
		},
		{
			input:     "pi|pe",
			permitted: false,
		},
		{
			input:     "give;injection",
			permitted: false,
		},
		{
			input:     "give~injection",
			permitted: false,
		},
		{
			input:     "give%injection",
			permitted: false,
		},
		{
			input:     "give'injection",
			permitted: false,
		},
		{
			input:     ".start-period",
			permitted: false,
		},
		{
			input:     "end-period.",
			permitted: false,
		},
		{
			input:     "end-slash/",
			permitted: false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("TestSafeBranchName-%d-%s", i, tc.input), func(t *testing.T) {
			assert.Equal(t, tc.permitted, safeBranchName(tc.input))
		})
	}
}

func TestSafeRepositoryUrl(t *testing.T) {

	cases := []struct {
		input     string
		permitted bool
	}{
		{
			input:     "https://REDACTED@dev.azure.com/REDACTED/Scan-Test-1/_git/main",
			permitted: true,
		}, {
			input:     "https://REDACTED@dev.azure.com/REDACTED/Scan-Test-1/_git/Craz%28%29y%20Repo",
			permitted: false,
		}, {
			input:     "https://dev.azure.com/PHorton0655/Scan-Test-1/_git/Craz%28%29y%20Repo",
			permitted: false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("TestSafeRepositoryUrl-%d-%s", i, tc.input), func(t *testing.T) {
			assert.Equal(t, tc.permitted, safeRepositoryUrl(tc.input))
		})
	}
}

func TestScmSafeName(t *testing.T) {

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
			expected: "Name-WithDoubleSpace",
		},
		{
			input:    "Name  WithManySpaces",
			expected: "Name-WithManySpaces",
		},
		{
			input:    "  Name  WithLeadingSpaces",
			expected: "Name-WithLeadingSpaces",
		},
		{
			input:    "Name  WithTrailingSpaces  ",
			expected: "Name-WithTrailingSpaces",
		},
		{
			input:    "Name	WithTab",
			expected: "Name-WithTab",
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("TestScmOrganisationSafeName-%d-%s", i, tc.input), func(t *testing.T) {
			assert.Equal(t, tc.expected, safeName(tc.input))
		})
	}
}
