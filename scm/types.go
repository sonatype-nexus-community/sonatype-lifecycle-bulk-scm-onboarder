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
	"regexp"
	"strings"
)

const (
	BANNED_CHARS_ID   = ";$!&|()[]<> _#"
	BANNED_CHARS_NAME = ";$!&|()[]<>"
	SCM_TYPE_AZURE    = "azure"
)

var MULTIPLE_SPACES = regexp.MustCompile(`\s(\s+)`)

type ScmConfiguration struct {
	Type     string
	Username string
	Password string
}

type Application struct {
	Name          string
	DefaultBranch *string
	RepositoryUrl string
}

func (a *Application) PrintTree(depth int) {
	println(fmt.Sprintf("%sAPP: %s (to be created as %s)", strings.Repeat(" -- ", depth), a.Name, a.SafeName()))
}

func (a *Application) SafeId() string {
	return strings.ToLower(strings.Map(func(r rune) rune {
		if strings.Contains(BANNED_CHARS_ID, string(r)) {
			return '-'
		} else {
			return r
		}
	}, a.Name))
}

func (a *Application) SafeName() string {
	return safeName(a.Name)
}

type Organization struct {
	Name             string
	ScmProvider      string
	Applications     []Application
	SubOrganizations []Organization
}

func (o *Organization) PrintTree(depth int) {
	println(fmt.Sprintf("%sORG: %s (to be created as %s)", strings.Repeat(" -- ", depth), o.Name, o.SafeName()))
	for _, a := range o.Applications {
		a.PrintTree((depth + 1))
	}
	for _, so := range o.SubOrganizations {
		so.PrintTree((depth + 1))
	}
}

func (o *Organization) SafeName() string {
	return safeName(o.Name)
}

type OrgContents struct {
	Organizations []Organization
}

func (oc *OrgContents) PrintTree() {
	depth := 0
	for _, o := range oc.Organizations {
		o.PrintTree(depth)
	}
}

func safeName(in string) string {
	return MULTIPLE_SPACES.ReplaceAllString(
		strings.Map(func(r rune) rune {
			if strings.Contains(BANNED_CHARS_NAME, string(r)) {
				return '-'
			} else {
				return r
			}
		}, strings.ReplaceAll(strings.TrimSpace(in), "\t", "-")),
		"-",
	)
}
