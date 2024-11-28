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

package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/sonatype-nexus-community/sonatype-lifecycle-bulk-scm-onboarder/iq"
	"github.com/sonatype-nexus-community/sonatype-lifecycle-bulk-scm-onboarder/scm"
	"github.com/sonatype-nexus-community/sonatype-lifecycle-bulk-scm-onboarder/util"
)

const (
	ENV_ADO_PAT       = "SCM_ADO_PAT"
	ENV_NXIQ_USERNAME = "NXIQ_USERNAME"
	ENV_NXIQ_PASSWORD = "NXIQ_PASSWORD"
)

var (
	azureScm              bool   = false
	debugLogging          bool   = false
	currentRuntime        string = runtime.GOOS
	commit                       = "unknown"
	nxiqOrgNameToImportTo string
	nxiqUrl               string
	nxiqUsername          string
	nxiqPassword          string
	version               = "dev"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: sonatype-lifecycle-bulk-scm-onboarder [OPTIONS]\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func init() {
	flag.BoolVar(&azureScm, "azure", false, fmt.Sprintf("Load from Azure DevOps (set PAT in %s Environment Variable)", ENV_ADO_PAT))
	flag.StringVar(&nxiqUrl, "url", "http://localhost:8070", "URL including protocol to your Sonatype Lifecycle")
	flag.StringVar(&nxiqUsername, "username", "", fmt.Sprintf("Username used to authenticate to Sonatype Lifecycle (can also be set using the environment variable %s)", ENV_NXIQ_USERNAME))
	flag.StringVar(&nxiqPassword, "password", "", fmt.Sprintf("Password used to authenticate to Sonatype Lifecycle (can also be set using the environment variable %s)", ENV_NXIQ_PASSWORD))
	flag.StringVar(&nxiqOrgNameToImportTo, "org-name", "Root Organization", fmt.Sprintf("Name of Organization to import structure into"))
	flag.BoolVar(&debugLogging, "X", false, "Enable debug logging")
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&util.LogFormatter{Module: "SLI"})

	flag.Usage = usage
	flag.Parse()

	// Disable Debug Logging if not requested
	if !debugLogging {
		log.SetLevel(log.InfoLevel)
	}

	// Load Credentials
	err := loadCredentials()
	if err != nil {
		os.Exit(1)
	}

	if strings.TrimSpace(nxiqUrl) == "" {
		println("URL to Sonatype Lifecycle must be supplied")
		os.Exit(1)
	}

	// Output Banner
	println(strings.Repeat("⬢⬡", 42))
	println("")
	println("	███████╗ ██████╗ ███╗   ██╗ █████╗ ████████╗██╗   ██╗██████╗ ███████╗  ")
	println(" 	██╔════╝██╔═══██╗████╗  ██║██╔══██╗╚══██╔══╝╚██╗ ██╔╝██╔══██╗██╔════╝  ")
	println("	███████╗██║   ██║██╔██╗ ██║███████║   ██║    ╚████╔╝ ██████╔╝█████╗    ")
	println(" 	╚════██║██║   ██║██║╚██╗██║██╔══██║   ██║     ╚██╔╝  ██╔═══╝ ██╔══╝    ")
	println(" 	███████║╚██████╔╝██║ ╚████║██║  ██║   ██║      ██║   ██║     ███████╗  ")
	println(" 	╚══════╝ ╚═════╝ ╚═╝  ╚═══╝╚═╝  ╚═╝   ╚═╝      ╚═╝   ╚═╝     ╚══════╝  ")
	println("")
	println(fmt.Sprintf("	Running on:		%s/%s", currentRuntime, runtime.GOARCH))
	println(fmt.Sprintf("	Version: 		%s (%s)", version, commit))
	println("")
	println(strings.Repeat("⬢⬡", 42))
	println("")

	// Connect to IQ
	nxiqServer := iq.NewNxiqServer(nxiqUrl, nxiqUsername, nxiqPassword)
	iqTargetOrganization, err := nxiqServer.ValidateOrganizationByName(nxiqOrgNameToImportTo)
	if err != nil {
		println(fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}
	if iqTargetOrganization == nil {
		println(fmt.Sprintf("Could not find requested Organization %s", nxiqOrgNameToImportTo))
		os.Exit(1)
	}

	println(fmt.Sprintf("Target Organization in Sonatype: %s (%s)", *iqTargetOrganization.Name, *iqTargetOrganization.Id))
	println("")

	var orgContents *scm.OrgContents
	var scmConfig *scm.ScmConfiguration

	// If Azure, query Azure DevOps
	if azureScm {
		println("Loading from Azure DevOps...")
		println("")
		orgContents, scmConfig, err = loadFromAzureDevOps()
		if err != nil {
			panic(err)
		}
	}

	orgContents.PrintTree()

	println("")
	continueToCreateInIq := askForConfirmation("Continue to create Organizations and Applications in Sonatype Lifecycle?")
	if continueToCreateInIq {
		println("Creating Organizations and Applications in Sonatype Lifecycle. Please wait...")
		nxiqServer.ApplyOrgContents(*orgContents, iqTargetOrganization, scmConfig)
		println("Done 😉")
	}
}

func askForConfirmation(s string) bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s [y/n]: ", s)
		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		response = strings.ToLower(strings.TrimSpace(response))
		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		}
	}
}

func loadFromAzureDevOps() (*scm.OrgContents, *scm.ScmConfiguration, error) {
	envPat := os.Getenv(ENV_ADO_PAT)
	if strings.TrimSpace(envPat) == "" {
		return nil, nil, fmt.Errorf("Missing Azure PAT in environment varaible")
	}

	scmConnection := scm.NewAzureDevOpsScmIntegration(envPat, nil)
	orgContents, err := scmConnection.GetMappedAsOrgContents()
	if err != nil {
		return nil, nil, err
	}

	return orgContents, scmConnection.GetScmConfig(), nil
}

func loadCredentials() error {
	if strings.TrimSpace(nxiqUsername) == "" {
		log.Debug("Username not supplied as argument - checking environment variable")
		envUsername := os.Getenv(ENV_NXIQ_USERNAME)
		if strings.TrimSpace(envUsername) == "" {
			return fmt.Errorf("No username has been supplied either via argument or environment variable. Cannot continue.")
		} else {
			nxiqUsername = envUsername
		}
	}

	if strings.TrimSpace(nxiqPassword) == "" {
		log.Debug("Password not supplied as argument - checking environment variable")
		envPassword := os.Getenv(ENV_NXIQ_PASSWORD)
		if strings.TrimSpace(envPassword) == "" {
			log.Error("No password has been supplied either via argument or environment variable. Cannot continue.")
			return fmt.Errorf("No password has been supplied either via argument or environment variable. Cannot continue.")
		} else {
			nxiqPassword = envPassword
		}
	}

	return nil
}
