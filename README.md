# Sonatype Lifecycle Bulk SCM Importer

<!-- Badges Section -->
[![shield_gh-workflow-test]][link_gh-workflow-test]
[![shield_license]][license_file]
<!-- Add other badges or shields as appropriate -->

---

Introduce your project here. A short summary about what its purpose and scope is.

- [What does this tool do?](#what-does-this-tool-do)
  - [Organization Creation](#organization-creation)
  - [Application Creation](#application-creation)
- [Installation](#installation)
- [Usage](#usage)
- [Development](#development)
- [The Fine Print](#the-fine-print)

## What does this tool do?

This tool queries your Source Control Management (SCM) system and creates Organizations and Applications in Sonatype Lifecycle in bulk.

This tool is intended to be run against either an empty Sonatype Lifecycle installation or a specific Organization that represents a given SCM connection ONCE.

This tool is NOT intended to be run multiple times to continuously import/keep Sonatype Lifecycle up to date with newly created Projects or Repositories in your SCM system - to do this (once you've run this tool once successfully), use [Easy SCM Onboarding](https://help.sonatype.com/en/easy-scm-onboarding.html).

Currently supports:
- ✅ Azure DevOps

### Organization Creation

Sub-organizations under either your Root Organization, or an existing Organization of your choosing (see `-org-name` flag) will be created where they do no exist matching your SCM organizations and/or projects. Organizations will not be re-created or duplicated by running this import process. This can lead in some cases to applications from more than one SCM organizations or project being creating in a single Sonatype Organization due to naming restrictions.

### Application Creation

Applications will be create where they cannot be determined to exist for the Repository in your SCM. There are sitations where, due to naming collisions, this cannot be determined and so if you run the import more than once, it is possible that you will have some applications duplicated.

If an Application is determined to already exist, it's SCM configuration will be updated. SCM configuration is always set for newly created Applications.

## Installation

Obtain the binary for your Operating System and Architecture from the [GitHub Releases page](https://github.com/sonatype-nexus-community/nexus-repo-asset-lister/releases).

## Usage

You can see all options by running:

```
./sonatype-lifecycle-bulk-scm-onboarder --help
usage: sonatype-lifecycle-bulk-scm-onboarder [OPTIONS]
  -X    Enable debug logging
  -azure
        Load from Azure DevOps (set PAT in SCM_ADO_PAT Environment Variable else you'll be prompted to enter it)
  -org-name string
        Name of Organization to import structure into (default "Root Organization")
  -password string
        Password used to authenticate to Sonatype Lifecycle (can also be set using the environment variable NXIQ_PASSWORD, else you'll be prompted to enter it)
  -url string
        URL including protocol to your Sonatype Lifecycle (default "http://localhost:8070")
  -username string
        Username used to authenticate to Sonatype Lifecycle (can also be set using the environment variable NXIQ_USERNAME, else you'll be prompted to enter it)
```

The URL of the Sonatype Nexus Repository sever is specified with the `-url` argument and should contain the protcol (e.g. `https://`) and any context path you may have set for the installation.

Credentials can be supplied as command-line, via Environment Variables or failing that you'll be prompted to enter during exectuion.

```
NXRM_USERNAME=username NXRM_PASSWORD=password ./sonatype-lifecycle-bulk-scm-onboarder -url https://my-nexus-repository.tld
```

You can use your User Token instead of actual username and password for Sonatype Lifecycle.

## Development

See [CONTRIBUTING.md](./CONTRIBUTING.md) for details.

## The Fine Print

Remember:

This project is part of the [Sonatype Nexus Community](https://github.com/sonatype-nexus-community) organization, which is not officially supported by Sonatype. Please review the latest pull requests, issues, and commits to understand this project's readiness for contribution and use.

* File suggestions and requests on this repo through GitHub Issues, so that the community can pitch in
* Use or contribute to this project according to your organization's policies and your own risk tolerance
* Don't file Sonatype support tickets related to this project— it won't reach the right people that way

Last but not least of all - have fun!

<!-- Links Section -->
[shield_gh-workflow-test]: https://img.shields.io/github/actions/workflow/status/sonatype-nexus-community/sonatype-lifecycle-bulk-scm-onboarder/build.yml?branch=main&logo=GitHub&logoColor=white "build"
[shield_license]: https://img.shields.io/github/license/sonatype-nexus-community/sonatype-lifecycle-bulk-scm-onboarder?logo=open%20source%20initiative&logoColor=white "license"

[link_gh-workflow-test]: https://github.com/sonatype-nexus-community/sonatype-lifecycle-bulk-scm-onboarder/actions/workflows/build.yml?query=branch%3Amain
[license_file]: https://github.com/sonatype-nexus-community/sonatype-lifecycle-bulk-scm-onboarderblob/main/LICENSE