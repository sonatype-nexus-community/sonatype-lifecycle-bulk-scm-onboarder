module github.com/sonatype-nexus-community/sonatype-lifecycle-bulk-scm-onboarder

go 1.22.0

toolchain go1.22.8

require github.com/sirupsen/logrus v1.9.3

require github.com/google/uuid v1.1.1 // indirect

require (
	github.com/microsoft/azure-devops-go-api/azuredevops/v7 v7.1.0
	github.com/sonatype-nexus-community/nexus-iq-api-client-go v0.184.3
	golang.org/x/sys v0.27.0 // indirect
)
