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
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/accounts"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/profile"
)

const (
	DEFAULT_ADO_BASE_URL = "https://app.vssps.visualstudio.com"
)

var (
	profileIdMe = "me"
)

type AzureDevOpsScmIntegration struct {
	BaseUrl       string
	pat           string
	connection    *azuredevops.Connection
	clientContext *context.Context
	profileId     *uuid.UUID
}

func NewAzureDevOpsScmIntegration(pat string, baseUrl *string) *AzureDevOpsScmIntegration {
	scm := &AzureDevOpsScmIntegration{
		pat: pat,
	}
	if baseUrl == nil {
		scm.BaseUrl = DEFAULT_ADO_BASE_URL
	} else {
		scm.BaseUrl = *baseUrl
	}

	scm.connection = azuredevops.NewPatConnection(scm.BaseUrl, scm.pat)
	ctx := context.Background()
	scm.clientContext = &ctx

	return scm
}

func (scm *AzureDevOpsScmIntegration) GetMappedAsOrgContents() (*OrgContents, error) {
	orgContents := OrgContents{}

	azureOrgs, err := scm.getOrganisations()
	if err != nil {
		return nil, err
	}

	for _, azureOrg := range *azureOrgs {
		subOrgs, err := scm.getSubOrganizationsForAzureAccount(&azureOrg)
		if err != nil {
			return nil, err
		}

		org := Organization{
			Name:             *azureOrg.AccountName,
			ScmProvider:      SCM_TYPE_AZURE,
			SubOrganizations: *subOrgs,
		}

		orgContents.Organizations = append(orgContents.Organizations, org)
	}

	return &orgContents, nil
}

func (scm *AzureDevOpsScmIntegration) getSubOrganizationsForAzureAccount(account *accounts.Account) (*[]Organization, error) {
	projects, err := scm.getProjectsForAccount(account)
	if err != nil {
		return nil, err
	}

	orgs := make([]Organization, 0)
	for _, o := range *projects {
		apps, err := scm.getApplicationsForProject(account, &o)
		if err != nil {
			return nil, err
		}

		org := Organization{
			Name:        *o.Name,
			ScmProvider: SCM_TYPE_AZURE,
			Applicatons: *apps,
		}
		orgs = append(orgs, org)
	}

	return &orgs, nil
}

func (scm *AzureDevOpsScmIntegration) getApplicationsForProject(account *accounts.Account, project *core.TeamProjectReference) (*[]Application, error) {
	repos, err := scm.getRepositoriesForProjectForAccount(*account.AccountUri, project.Id)
	if err != nil {
		return nil, err
	}

	apps := make([]Application, 0)
	for _, repo := range *repos {
		apps = append(apps, Application{
			Name:          *repo.Name,
			DefaultBranch: repo.DefaultBranch,
			RepositoryUrl: *repo.RemoteUrl,
		})
	}

	return &apps, nil
}

func (scm *AzureDevOpsScmIntegration) getOrganisations() (*[]accounts.Account, error) {
	log.Debug("Azure DevOps - Loading Organisations (from Accounts)")

	_, err := scm.getProfile()
	if err != nil {
		return nil, err
	}

	accounts, err := scm.getAccounts()
	if err != nil {
		return nil, err
	}

	return accounts, nil
}

func (scm *AzureDevOpsScmIntegration) getProfile() (*profile.Profile, error) {
	pClient, err := profile.NewClient(*scm.clientContext, scm.connection)
	if err != nil {
		return nil, err
	}

	profile, err := pClient.GetProfile(*scm.clientContext, profile.GetProfileArgs{
		Id: &profileIdMe,
	})
	if err != nil {
		return nil, err
	}

	log.Debug(fmt.Sprintf("Successfully connected to Azure DevOps (profiled ID %s)", profile.Id))
	scm.profileId = profile.Id
	return profile, nil
}

func (scm *AzureDevOpsScmIntegration) getAccounts() (*[]accounts.Account, error) {
	aClient, err := accounts.NewClient(*scm.clientContext, scm.connection)
	if err != nil {
		return nil, err
	}

	accounts, err := aClient.GetAccounts(*scm.clientContext, accounts.GetAccountsArgs{
		MemberId: scm.profileId,
	})
	if err != nil {
		return nil, err
	}

	return accounts, nil
}

func (scm *AzureDevOpsScmIntegration) getProjectsForAccount(account *accounts.Account) (*[]core.TeamProjectReference, error) {
	accountConnection := azuredevops.NewPatConnection(*account.AccountUri, scm.pat)
	coreClient, err := core.NewClient(*scm.clientContext, accountConnection)
	if err != nil {
		return nil, err
	}

	responseValue, err := coreClient.GetProjects(*scm.clientContext, core.GetProjectsArgs{})
	if err != nil {
		return nil, err
	}

	// index := 0
	allProjects := make([]core.TeamProjectReference, 0)
	for responseValue != nil {
		allProjects = append(allProjects, responseValue.Value...)
		log.Debug(fmt.Sprintf("Found %d Projects in Account %s", len(allProjects), *account.AccountName))

		// // Log the page of team project names
		// for _, teamProjectReference := range (*responseValue).Value {
		// 	log.Debug(fmt.Sprintf("Name[%0000d] = %s", index, *teamProjectReference.Name))
		// 	repos, err := scm.getRepositoriesForProjectForAccount(*account.AccountUri, teamProjectReference.Id)
		// 	if err != nil {
		// 		log.Error(err)
		// 	}

		// 	if repos != nil {
		// 		for j, repo := range *repos {
		// 			log.Debug(fmt.Sprintf("  	REPO[%0000d] = %s", j, *repo.Name))
		// 		}
		// 	}

		// 	index++
		// }

		// if continuationToken has a value, then there is at least one more page of projects to get
		if responseValue.ContinuationToken != "" {
			continuationToken, err := strconv.Atoi(responseValue.ContinuationToken)
			if err != nil {
				return nil, err
			}

			// Get next page of team projects
			projectArgs := core.GetProjectsArgs{
				ContinuationToken: &continuationToken,
			}
			responseValue, err = coreClient.GetProjects(*scm.clientContext, projectArgs)
			if err != nil {
				return nil, err
			}
		} else {
			responseValue = nil
		}
	}

	return &allProjects, nil
}

func (scm *AzureDevOpsScmIntegration) getRepositoriesForProjectForAccount(accountUri string, projectId *uuid.UUID) (*[]git.GitRepository, error) {
	log.Debug(fmt.Sprintf("Getting Repositories for Project %v", projectId))
	accountConnection := azuredevops.NewPatConnection(accountUri, scm.pat)
	gClient, err := git.NewClient(*scm.clientContext, accountConnection)
	if err != nil {
		return nil, err
	}

	pid := projectId.String()

	repositories, err := gClient.GetRepositories(*scm.clientContext, git.GetRepositoriesArgs{
		Project: &pid,
	})
	if err != nil {
		return nil, err
	}

	return repositories, nil
}

func (scm *AzureDevOpsScmIntegration) ValidateConnection() (bool, error) {
	return false, nil
}
