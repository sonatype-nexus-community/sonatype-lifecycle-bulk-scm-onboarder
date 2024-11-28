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

	log "github.com/sirupsen/logrus"

	sonatypeiq "github.com/sonatype-nexus-community/nexus-iq-api-client-go"
	"github.com/sonatype-nexus-community/sonatype-lifecycle-bulk-scm-onboarder/scm"
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

func (s *NxiqServer) ApplyOrgContents(orgContent scm.OrgContents, rootOrganization *sonatypeiq.ApiOrganizationDTO, scmConfig *scm.ScmConfiguration) error {
	for _, o := range orgContent.Organizations {
		org, err := s.CreateOrganization(o, *rootOrganization.Id)
		if err != nil {
			return err
		}
		log.Debug(fmt.Sprintf("Created Organization %s - %s", o.SafeName(), *org.Id))
		err = s.SetOrganizationScmConfiguration(org, scmConfig)
		if err != nil {
			return err
		}
		log.Debug(fmt.Sprintf("Applied %s SCM Configuration to Organization %s - %s", scmConfig.Type, o.SafeName(), *org.Id))

		err = s.createAppsInOrg(org, o.Applications)
		if err != nil {
			return err
		}

		if len(o.Applications) > 0 {
			for _, a := range o.Applications {
				app, err := s.CreateApplication(a, *org.Id)
				if err != nil {
					return err
				}
				log.Debug(fmt.Sprintf("Created Application %s - %s", a.SafeName(), *app.Id))
			}
		}

		if len(o.SubOrganizations) > 0 {
			for _, so := range o.SubOrganizations {
				subOrg, err := s.CreateOrganization(so, *org.Id)
				if err != nil {
					return err
				}
				log.Debug(fmt.Sprintf("Created Organization %s - %s", so.SafeName(), *subOrg.Id))

				err = s.createAppsInOrg(subOrg, so.Applications)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (s *NxiqServer) createAppsInOrg(org *sonatypeiq.ApiOrganizationDTO, apps []scm.Application) error {
	if len(apps) > 0 {
		for _, a := range apps {
			app, err := s.CreateApplication(a, *org.Id)
			if err != nil {
				return err
			}
			log.Debug(fmt.Sprintf("Created Application %s - %s", a.SafeName(), *app.Id))
		}
	}
	return nil
}

func (s *NxiqServer) CreateOrganization(org scm.Organization, parentOrgId string) (*sonatypeiq.ApiOrganizationDTO, error) {
	orgName := org.SafeName()
	createdOrg, r, err := s.apiClient.OrganizationsAPI.AddOrganization(*s.apiContext).ApiOrganizationDTO(sonatypeiq.ApiOrganizationDTO{
		Name:                 &orgName,
		ParentOrganizationId: &parentOrgId,
	}).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `OrganizationsAPI.AddOrganization``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
		return nil, err
	}
	return createdOrg, nil
}

func (s *NxiqServer) SetOrganizationScmConfiguration(org *sonatypeiq.ApiOrganizationDTO, scmConfig *scm.ScmConfiguration) error {
	// Set SCM Configuration for our top level Org(s)
	t := true
	f := false
	_, r, err := s.apiClient.SourceControlAPI.AddSourceControl(*s.apiContext, "organization", *org.Id).ApiSourceControlDTO(sonatypeiq.ApiSourceControlDTO{
		Username:                        &scmConfig.Username,
		Token:                           &scmConfig.Password,
		Provider:                        &scmConfig.Type,
		RemediationPullRequestsEnabled:  &f,
		PullRequestCommentingEnabled:    &f,
		SourceControlEvaluationsEnabled: &t,
		SshEnabled:                      &f,
		CommitStatusEnabled:             &f,
	}).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `SourceControlAPI.AddSourceControl``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
		return err
	}
	return nil
}

func (s *NxiqServer) CreateApplication(app scm.Application, parentOrgId string) (*sonatypeiq.ApiApplicationDTO, error) {
	appId := app.SafeId()
	appName := app.SafeName()
	createdApp, r, err := s.apiClient.ApplicationsAPI.AddApplication(*s.apiContext).ApiApplicationDTO(sonatypeiq.ApiApplicationDTO{
		PublicId:       &appId,
		Name:           &appName,
		OrganizationId: &parentOrgId,
	}).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `ApplicationsAPI.AddApplication``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
		return nil, err
	}

	// Set SCM Configuration
	_, r, err = s.apiClient.SourceControlAPI.AddSourceControl(*s.apiContext, "application", *createdApp.Id).ApiSourceControlDTO(sonatypeiq.ApiSourceControlDTO{
		RepositoryUrl:                   &app.RepositoryUrl,
		BaseBranch:                      app.DefaultBranch,
		EnablePullRequests:              nil,
		RemediationPullRequestsEnabled:  nil,
		PullRequestCommentingEnabled:    nil,
		SourceControlEvaluationsEnabled: nil,
		SshEnabled:                      nil,
	}).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `SourceControlAPI.AddSourceControl``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
		return nil, err
	}

	return createdApp, nil
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
