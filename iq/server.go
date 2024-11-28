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

package iq

import (
	"context"
	"fmt"
	"os"
	"strings"

	sonatypeiq "github.com/sonatype-nexus-community/nexus-iq-api-client-go"
)

type NxiqServer struct {
	baseUrl       string
	username      string
	password      string
	apiClient     *sonatypeiq.APIClient
	apiContext    *context.Context
	configuration *sonatypeiq.Configuration
}

func NewNxiqServer(url string, username string, password string) *NxiqServer {
	url = strings.TrimRight(url, "/")
	server := &NxiqServer{
		baseUrl:       url,
		username:      username,
		password:      password,
		configuration: sonatypeiq.NewConfiguration(),
	}

	server.configuration.Servers = sonatypeiq.ServerConfigurations{
		{
			URL:         url,
			Description: "Configured Sonatype Lifecycle",
		},
	}
	server.apiClient = sonatypeiq.NewAPIClient(server.configuration)

	c := context.WithValue(
		context.Background(),
		sonatypeiq.ContextBasicAuth,
		sonatypeiq.BasicAuth{
			UserName: username,
			Password: password,
		},
	)
	server.apiContext = &c
	return server
}

func (s *NxiqServer) ValidateOrganizationByName(organizationName string) (*sonatypeiq.ApiOrganizationDTO, error) {
	request := s.apiClient.OrganizationsAPI.GetOrganizations(*s.apiContext)
	request = request.OrganizationName([]string{organizationName})
	orgList, r, err := request.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `OrganizationsAPI.GetOrganizations``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
		return nil, err
	}

	if len(orgList.Organizations) == 1 {
		org := &orgList.Organizations[0]
		return org, nil
	}

	return nil, fmt.Errorf("%d Organizations returned for Name '%s'", len(orgList.Organizations), organizationName)
}
