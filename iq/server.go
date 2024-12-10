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
	"io"
	"net/http"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	sonatypeiq "github.com/sonatype-nexus-community/nexus-iq-api-client-go"
	"github.com/sonatype-nexus-community/sonatype-lifecycle-bulk-scm-onboarder/scm"
)

type nxiqApplication struct {
	Id             *string
	Name           *string
	OrganizationId *string
	PublicId       *string
	RepositoryUrl  *string
}

// type nxiqOrganization struct {
// 	Id                   *string
// 	Name                 *string
// 	ParentOrganizationId *string
// }

type NxiqServer struct {
	baseUrl               string
	username              string
	password              string
	apiClient             *sonatypeiq.APIClient
	apiContext            *context.Context
	configuration         *sonatypeiq.Configuration
	cacheLoaded           bool
	existingApplications  []nxiqApplication
	existingOrganizations []*sonatypeiq.ApiOrganizationDTO
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

func (s *NxiqServer) InitCache() error {
	if !s.cacheLoaded {
		err := s.cacheExistingOrganizations()
		if err != nil {
			return err
		}

		err = s.cacheExistingApplications()
		if err != nil {
			return err
		}

		s.cacheLoaded = true
	}

	return nil
}

func (s *NxiqServer) cacheExistingApplications() error {
	s.existingApplications = make([]nxiqApplication, 0)

	apiResponse, r, err := s.apiClient.ApplicationsAPI.GetApplications(*s.apiContext).Execute()
	if err != nil {
		log.Error(fmt.Sprintf("Failed to load existing Applications from Sonatype IQ: %s: %v: %v", r.Status, err, r.Body))
		return err
	}
	for _, a := range apiResponse.Applications {
		s.existingApplications = append(s.existingApplications, nxiqApplication{
			Id:             a.Id,
			PublicId:       a.PublicId,
			Name:           a.Name,
			OrganizationId: a.OrganizationId,
		})
	}

	log.Info(fmt.Sprintf("Loaded %d existing Applications from Sonatype Lifecycle", len(s.existingApplications)))
	return nil
}

func (s *NxiqServer) cacheExistingOrganizations() error {
	s.existingOrganizations = make([]*sonatypeiq.ApiOrganizationDTO, 0)

	apiResponse, r, err := s.apiClient.OrganizationsAPI.GetOrganizations(*s.apiContext).Execute()
	if err != nil {
		log.Error(fmt.Sprintf("Failed to load existing Applications from Sonatype IQ: %s: %v: %v", r.Status, err, r.Body))
		return err
	}
	for _, a := range apiResponse.Organizations {
		s.existingOrganizations = append(s.existingOrganizations, &sonatypeiq.ApiOrganizationDTO{
			Id:                   a.Id,
			Name:                 a.Name,
			ParentOrganizationId: a.ParentOrganizationId,
		})
	}

	log.Info(fmt.Sprintf("Loaded %d existing Organizations from Sonatype Lifecycle", len(s.existingOrganizations)))
	return nil
}

func (s *NxiqServer) ApplyOrgContents(orgContent scm.OrgContents, rootOrganization *sonatypeiq.ApiOrganizationDTO, scmConfig *scm.ScmConfiguration) error {
	for _, o := range orgContent.Organizations {
		org, err := s.CreateOrganization(o, *rootOrganization.Id, true, scmConfig)
		if err != nil {
			return err
		}

		err = s.createAppsInOrg(org, o.Applications)
		if err != nil {
			return err
		}

		if len(o.SubOrganizations) > 0 {
			for _, so := range o.SubOrganizations {
				subOrg, err := s.CreateOrganization(so, *org.Id, false, nil)
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
			s.scheduleSourceStageScan(app, a.DefaultBranch)
		}
	}
	return nil
}

func (s *NxiqServer) scheduleSourceStageScan(app *sonatypeiq.ApiApplicationDTO, defaultBranchName *string) {
	sourceStage := "source"
	_, r, err := s.apiClient.PolicyEvaluationAPI.EvaluateSourceControl(*s.apiContext, *app.Id).ApiSourceControlEvaluationRequestDTO(sonatypeiq.ApiSourceControlEvaluationRequestDTO{
		BranchName: defaultBranchName,
		StageId:    &sourceStage,
	}).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `PolicyEvaluationAPI.EvaluateSourceControl``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}

/**
 * Creates an Organization if it does not already exist.
 *
 * If `applyScmConfiguration` is true and the Organization already existed, SCM configuration
 * will be updated. If the Organization was just created, it will be set.
 *
 */
func (s *NxiqServer) CreateOrganization(org scm.Organization, parentOrgId string, applyScmConfiguration bool, scmConfig *scm.ScmConfiguration) (*sonatypeiq.ApiOrganizationDTO, error) {
	existingOrg, err := s.OrganizationExists(org, parentOrgId)
	if err != nil {
		log.Debug(fmt.Sprintf("Failed to determine if Organization %s already exists", org.Name))
		return nil, err
	}

	if existingOrg != nil {
		if applyScmConfiguration {
			err = s.UpdateOrganizationScmConfiguration(existingOrg, scmConfig)
			if err != nil {
				return existingOrg, err
			}
			log.Debug(fmt.Sprintf("Updated %s SCM Configuration for Organization %s - %s", scmConfig.Type, org.SafeName(), *existingOrg.Id))
		}
		return existingOrg, nil
	}

	createdOrg, err := s.createOrganization(org, parentOrgId)
	if err != nil {
		return createdOrg, err
	}
	log.Debug(fmt.Sprintf("Created Organization %s - %v", org.SafeName(), org))
	if applyScmConfiguration {
		err = s.SetOrganizationScmConfiguration(createdOrg, scmConfig)
		if err != nil {
			return createdOrg, err
		}
		log.Debug(fmt.Sprintf("Applied %s SCM Configuration to Organization %s - %s", scmConfig.Type, org.SafeName(), *createdOrg.Id))
	}

	return createdOrg, nil
}

func (s *NxiqServer) OrganizationExists(org scm.Organization, parentOrgId string) (*sonatypeiq.ApiOrganizationDTO, error) {
	s.InitCache()
	for _, existingOrg := range s.existingOrganizations {
		if *existingOrg.Name == org.SafeName() && *existingOrg.ParentOrganizationId == parentOrgId {
			return existingOrg, nil
		}
	}
	return nil, nil
}

func (s *NxiqServer) createOrganization(org scm.Organization, parentOrgId string) (*sonatypeiq.ApiOrganizationDTO, error) {
	orgName := s.getUniqueOrganizationId(org.SafeName())

	var err error
	var httpResponse *http.Response
	var attemptCount = 0
	var createdOrg *sonatypeiq.ApiOrganizationDTO
	for httpResponse == nil || httpResponse.StatusCode != http.StatusOK {
		createdOrg, httpResponse, err = s.apiClient.OrganizationsAPI.AddOrganization(*s.apiContext).ApiOrganizationDTO(sonatypeiq.ApiOrganizationDTO{
			Name:                 &orgName,
			ParentOrganizationId: &parentOrgId,
		}).Execute()

		attemptCount += 1

		if httpResponse.StatusCode == http.StatusBadRequest {
			// We possibly had a colision - check response body
			defer httpResponse.Body.Close()

			b, err := io.ReadAll(httpResponse.Body)
			if err != nil {
				log.Fatalln(err)
			}
			responseBody := string(b)
			log.Debug(fmt.Sprintf("Response Body: %s", responseBody))

			if strings.HasSuffix(responseBody, "used as a name.") {
				// Name had a conflict
				orgName = fmt.Sprintf("%s-%d", s.getUniqueOrganizationId(org.SafeName()), attemptCount)
				log.Debug(fmt.Sprintf("Bumped Organization Name to be %s", orgName))
				continue
			}
		}

		if attemptCount > 2 && err != nil {
			log.Debug(fmt.Sprintf("Error when calling `OrganizationsAPI.AddOrganization` on attempt %d: %v\n", attemptCount, err))
			log.Debug(fmt.Sprintf("Full HTTP response: %v\n", httpResponse))
			return nil, err
		}
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

func (s *NxiqServer) UpdateOrganizationScmConfiguration(org *sonatypeiq.ApiOrganizationDTO, scmConfig *scm.ScmConfiguration) error {
	// Set SCM Configuration for our top level Org(s)
	t := true
	f := false
	_, r, err := s.apiClient.SourceControlAPI.UpdateSourceControl(*s.apiContext, "organization", *org.Id).ApiSourceControlDTO(sonatypeiq.ApiSourceControlDTO{
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
		fmt.Fprintf(os.Stderr, "Error when calling `SourceControlAPI.UpdateSourceControl``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
		return err
	}
	return nil
}

func (s *NxiqServer) CreateApplication(app scm.Application, parentOrgId string) (*sonatypeiq.ApiApplicationDTO, error) {
	createdApp, err := s.createApplication(app, parentOrgId)
	if err != nil {
		return nil, err
	}
	log.Debug(fmt.Sprintf("Created App: %v", createdApp))

	// Set SCM Configuration
	_, r, err := s.apiClient.SourceControlAPI.AddSourceControl(*s.apiContext, "application", *createdApp.Id).ApiSourceControlDTO(sonatypeiq.ApiSourceControlDTO{
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

func (s *NxiqServer) createApplication(app scm.Application, parentOrgId string) (*sonatypeiq.ApiApplicationDTO, error) {
	appId := s.getUniqueSafeApplicationId(app.SafeId())
	appName := app.SafeName()

	var err error
	var httpResponse *http.Response
	var attemptCount = 0
	var createdApp *sonatypeiq.ApiApplicationDTO
	for httpResponse == nil || httpResponse.StatusCode != http.StatusOK {
		createdApp, httpResponse, err = s.apiClient.ApplicationsAPI.AddApplication(*s.apiContext).ApiApplicationDTO(sonatypeiq.ApiApplicationDTO{
			PublicId:       &appId,
			Name:           &appName,
			OrganizationId: &parentOrgId,
		}).Execute()

		attemptCount += 1

		if httpResponse.StatusCode == http.StatusBadRequest {
			// We possibly had a colision - check response body
			defer httpResponse.Body.Close()

			b, err := io.ReadAll(httpResponse.Body)
			if err != nil {
				log.Fatalln(err)
			}
			responseBody := string(b)
			log.Debug(fmt.Sprintf("Response Body: %s", responseBody))

			if strings.HasSuffix(responseBody, "as an ID.") || strings.HasSuffix(responseBody, "as a name.") {
				// ID or Name had a conflict
				appId = fmt.Sprintf("%s-%d", app.SafeId(), attemptCount)
				appName = fmt.Sprintf("%s-%d", app.SafeName(), attemptCount)
				log.Debug(fmt.Sprintf("Bumped Application ID and Name to be %s, %s", appId, appName))
				continue
			}
		}

		if attemptCount > 2 && err != nil {
			log.Debug(fmt.Sprintf("Error when calling `ApplicationsAPI.AddApplication` on attempt %d: %v\n", attemptCount, err))
			log.Debug(fmt.Sprintf("Full HTTP response: %v\n", httpResponse))
			return nil, err
		}
	}

	return createdApp, nil
}

func (s *NxiqServer) getUniqueSafeApplicationId(id string) string {
	for _, existinApp := range s.existingApplications {
		if *existinApp.Id == id {
			return fmt.Sprintf("%s-1", id)
		}
	}
	return id
}

func (s *NxiqServer) getUniqueOrganizationId(id string) string {
	for _, existingOrg := range s.existingOrganizations {
		if *existingOrg.Id == id {
			return fmt.Sprintf("%s-1", id)
		}
	}
	return id
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
