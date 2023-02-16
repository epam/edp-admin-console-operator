<a name="unreleased"></a>
## [Unreleased]

### Routine

- Update current development version [EPMDEDP-10610](https://jiraeu.epam.com/browse/EPMDEDP-10610)

### Documentation

- Update chart and application version in Readme file [EPMDEDP-11221](https://jiraeu.epam.com/browse/EPMDEDP-11221)


<a name="v2.13.0"></a>
## [v2.13.0] - 2022-12-06
### Features

- Added a stub linter [EPMDEDP-10536](https://jiraeu.epam.com/browse/EPMDEDP-10536)

### Code Refactoring

- Remove db resources from deployment templates [EPMDEDP-10750](https://jiraeu.epam.com/browse/EPMDEDP-10750)
- Set visibility to false for EDPComponent [EPMDEDP-10771](https://jiraeu.epam.com/browse/EPMDEDP-10771)
- Revert: Do not create EDPComponent [EPMDEDP-10771](https://jiraeu.epam.com/browse/EPMDEDP-10771)
- Do not create EDPComponent [EPMDEDP-10771](https://jiraeu.epam.com/browse/EPMDEDP-10771)

### Routine

- Update current development version [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)
- Update current development version [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)
- Bump version to 2.14.0 [EPMDEDP-10610](https://jiraeu.epam.com/browse/EPMDEDP-10610)
- Add deprecation warning [EPMDEDP-10764](https://jiraeu.epam.com/browse/EPMDEDP-10764)


<a name="v2.12.0"></a>
## [v2.12.0] - 2022-08-26
### Features

- Switch to use V1 apis of EDP components [EPMDEDP-10079](https://jiraeu.epam.com/browse/EPMDEDP-10079)
- Download required tools for Makefile targets [EPMDEDP-10105](https://jiraeu.epam.com/browse/EPMDEDP-10105)
- Set default client scopes for keycloak admin-console-client [EPMDEDP-8323](https://jiraeu.epam.com/browse/EPMDEDP-8323)
- Switch CRD to v1 version [EPMDEDP-9213](https://jiraeu.epam.com/browse/EPMDEDP-9213)

### Bug Fixes

- Fix e2e tests [EPMDEDP-10079](https://jiraeu.epam.com/browse/EPMDEDP-10079)

### Code Refactoring

- Deprecate unused Spec components for admin-console v1 [EPMDEDP-10148](https://jiraeu.epam.com/browse/EPMDEDP-10148)
- Storageclass is no longer used by Admin Console [EPMDEDP-10228](https://jiraeu.epam.com/browse/EPMDEDP-10228)
- Use repository and tag for image reference in chart [EPMDEDP-10389](https://jiraeu.epam.com/browse/EPMDEDP-10389)

### Routine

- Upgrade go version to 1.18 [EPMDEDP-10110](https://jiraeu.epam.com/browse/EPMDEDP-10110)
- Fix Jira Ticket pattern for changelog generator [EPMDEDP-10159](https://jiraeu.epam.com/browse/EPMDEDP-10159)
- Bump version to 2.12.0 [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)
- Update alpine base image to 3.16.2 version [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)
- Update alpine base image version [EPMDEDP-10280](https://jiraeu.epam.com/browse/EPMDEDP-10280)
- Change 'go get' to 'go install' for git-chglog [EPMDEDP-10337](https://jiraeu.epam.com/browse/EPMDEDP-10337)
- Use deployments as default deploymentType for OpenShift [EPMDEDP-10344](https://jiraeu.epam.com/browse/EPMDEDP-10344)
- Remove VERSION file [EPMDEDP-10387](https://jiraeu.epam.com/browse/EPMDEDP-10387)
- Add gcflags for go build artifact [EPMDEDP-10411](https://jiraeu.epam.com/browse/EPMDEDP-10411)
- Update current development version [EPMDEDP-8832](https://jiraeu.epam.com/browse/EPMDEDP-8832)
- Update chart annotation [EPMDEDP-9515](https://jiraeu.epam.com/browse/EPMDEDP-9515)

### Documentation

- Align README.md [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)
- Align README.md [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)
- Fix indents in README.md [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)
- Align README.md [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)


<a name="v2.11.0"></a>
## [v2.11.0] - 2022-05-25
### Features

- Update Makefile changelog target [EPMDEDP-8218](https://jiraeu.epam.com/browse/EPMDEDP-8218)
- Add get cbis permission for admin-console service account [EPMDEDP-8262](https://jiraeu.epam.com/browse/EPMDEDP-8262)
- Add ingress tls certificate option when using ingress controller [EPMDEDP-8377](https://jiraeu.epam.com/browse/EPMDEDP-8377)
- Add get and list edpcomponents permission for edp-resources-admin role [EPMDEDP-8382](https://jiraeu.epam.com/browse/EPMDEDP-8382)
- Generate CRDs and helm docs automatically [EPMDEDP-8385](https://jiraeu.epam.com/browse/EPMDEDP-8385)
- Add permission for edp-resources-admin role [EPMDEDP-8798](https://jiraeu.epam.com/browse/EPMDEDP-8798)
- Add CIS list permission for edp-resources-admin role [EPMDEDP-8812](https://jiraeu.epam.com/browse/EPMDEDP-8812)

### Bug Fixes

- Change ca-certificates in dockerfile [EPMDEDP-8238](https://jiraeu.epam.com/browse/EPMDEDP-8238)
- Fix changelog generation in GH Release Action [EPMDEDP-8468](https://jiraeu.epam.com/browse/EPMDEDP-8468)
- Correct image version [EPMDEDP-8471](https://jiraeu.epam.com/browse/EPMDEDP-8471)
- Switch e2e stage to new helm chart repository [EPMDEDP-8800](https://jiraeu.epam.com/browse/EPMDEDP-8800)

### Code Refactoring

- Always start admin-console with SSO disabled [EPMDEDP-7105](https://jiraeu.epam.com/browse/EPMDEDP-7105)
- Refactor basePath definition [EPMDEDP-7105](https://jiraeu.epam.com/browse/EPMDEDP-7105)
- Ensure we are aligned with SSO enabled flag [EPMDEDP-7105](https://jiraeu.epam.com/browse/EPMDEDP-7105)
- Introduce wait-for-db for AC deployment [EPMDEDP-8005](https://jiraeu.epam.com/browse/EPMDEDP-8005)
- Define namespace for Service Account in Role Binding [EPMDEDP-8084](https://jiraeu.epam.com/browse/EPMDEDP-8084)

### Routine

- Update Ingress resources to the newest API version [EPMDEDP-7476](https://jiraeu.epam.com/browse/EPMDEDP-7476)
- Update release CI pipelines [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Populate chart with Artifacthub annotations [EPMDEDP-8049](https://jiraeu.epam.com/browse/EPMDEDP-8049)
- Update changelog [EPMDEDP-8227](https://jiraeu.epam.com/browse/EPMDEDP-8227)
- Update artifacthub.io images metadata [EPMDEDP-8386](https://jiraeu.epam.com/browse/EPMDEDP-8386)
- Fix artifacthub.io crdsExamples metadata [EPMDEDP-8386](https://jiraeu.epam.com/browse/EPMDEDP-8386)
- Update artifacthub.io chart metadata [EPMDEDP-8386](https://jiraeu.epam.com/browse/EPMDEDP-8386)
- Bump version to 2.11.0 [EPMDEDP-8832](https://jiraeu.epam.com/browse/EPMDEDP-8832)
- Update base docker image to alpine 3.15.4 [EPMDEDP-8853](https://jiraeu.epam.com/browse/EPMDEDP-8853)
- Update changelog [EPMDEDP-9185](https://jiraeu.epam.com/browse/EPMDEDP-9185)


<a name="v2.10.0"></a>
## [v2.10.0] - 2021-12-07
### Features

- Provide operator's build information [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)

### Bug Fixes

- Provide Admin Console deploy through deployments on OKD cluster [EPMDEDP-7178](https://jiraeu.epam.com/browse/EPMDEDP-7178)
- Changelog links [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)

### Code Refactoring

- Replace cluster-wide role/rolebinding to namespaced, remove unused roles [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
- Address golangci-lint issues [EPMDEDP-7945](https://jiraeu.epam.com/browse/EPMDEDP-7945)

### Formatting

- Remove unnecessary spaces [EPMDEDP-7943](https://jiraeu.epam.com/browse/EPMDEDP-7943)

### Testing

- Exclude cmd from sonar scan [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)

### Routine

- Bump version to 2.10.0 [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Use custom sorting order for changelog [EPMDEDP-7874](https://jiraeu.epam.com/browse/EPMDEDP-7874)
- Add changelog generator [EPMDEDP-7874](https://jiraeu.epam.com/browse/EPMDEDP-7874)
- Add codecov report [EPMDEDP-7885](https://jiraeu.epam.com/browse/EPMDEDP-7885)
- Update docker image [EPMDEDP-7895](https://jiraeu.epam.com/browse/EPMDEDP-7895)
- Update keycloak to the latest stable [EPMDEDP-7930](https://jiraeu.epam.com/browse/EPMDEDP-7930)
- Use custom go build step for operator [EPMDEDP-7932](https://jiraeu.epam.com/browse/EPMDEDP-7932)
- Update go to version 1.17 [EPMDEDP-7932](https://jiraeu.epam.com/browse/EPMDEDP-7932)

### Documentation

- Update the links on GitHub [EPMDEDP-7781](https://jiraeu.epam.com/browse/EPMDEDP-7781)


<a name="v2.9.0"></a>
## [v2.9.0] - 2021-12-03

<a name="v2.8.2"></a>
## [v2.8.2] - 2021-12-03

<a name="v2.8.1"></a>
## [v2.8.1] - 2021-12-03

<a name="v2.8.0"></a>
## [v2.8.0] - 2021-12-03

<a name="v2.7.2"></a>
## [v2.7.2] - 2021-12-03

<a name="v2.7.1"></a>
## [v2.7.1] - 2021-12-03

<a name="v2.7.0"></a>
## [v2.7.0] - 2021-12-03

[Unreleased]: https://github.com/epam/edp-admin-console-operator/compare/v2.13.0...HEAD
[v2.13.0]: https://github.com/epam/edp-admin-console-operator/compare/v2.12.0...v2.13.0
[v2.12.0]: https://github.com/epam/edp-admin-console-operator/compare/v2.11.0...v2.12.0
[v2.11.0]: https://github.com/epam/edp-admin-console-operator/compare/v2.10.0...v2.11.0
[v2.10.0]: https://github.com/epam/edp-admin-console-operator/compare/v2.9.0...v2.10.0
[v2.9.0]: https://github.com/epam/edp-admin-console-operator/compare/v2.8.2...v2.9.0
[v2.8.2]: https://github.com/epam/edp-admin-console-operator/compare/v2.8.1...v2.8.2
[v2.8.1]: https://github.com/epam/edp-admin-console-operator/compare/v2.8.0...v2.8.1
[v2.8.0]: https://github.com/epam/edp-admin-console-operator/compare/v2.7.2...v2.8.0
[v2.7.2]: https://github.com/epam/edp-admin-console-operator/compare/v2.7.1...v2.7.2
[v2.7.1]: https://github.com/epam/edp-admin-console-operator/compare/v2.7.0...v2.7.1
[v2.7.0]: https://github.com/epam/edp-admin-console-operator/compare/v2.3.0-78...v2.7.0
