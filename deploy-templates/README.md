# admin-console-operator

![Version: 2.13.0](https://img.shields.io/badge/Version-2.13.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 2.13.0](https://img.shields.io/badge/AppVersion-2.13.0-informational?style=flat-square)

A Helm chart for EDP Admin Console Operator

**Homepage:** <https://epam.github.io/edp-install/>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| epmd-edp | <SupportEPMD-EDP@epam.com> | <https://solutionshub.epam.com/solution/epam-delivery-platform> |
| sergk |  | <https://github.com/SergK> |

## Source Code

* <https://github.com/epam/edp-admin-console>

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| adminConsole.affinity | object | `{}` |  |
| adminConsole.annotations | object | `{}` |  |
| adminConsole.authKeycloakEnabled | bool | `true` | Authentication Keycloak enabled/disabled |
| adminConsole.basePath | string | `""` | Base path for Admin Console URL |
| adminConsole.envs[0].name | string | `"INTEGRATION_STRATEGIES"` |  |
| adminConsole.envs[0].value | string | `"Create,Clone,Import"` |  |
| adminConsole.envs[1].name | string | `"BUILD_TOOLS"` |  |
| adminConsole.envs[1].value | string | `"maven"` |  |
| adminConsole.envs[2].name | string | `"DEPLOYMENT_SCRIPT"` |  |
| adminConsole.envs[2].value | string | `"helm-chart"` |  |
| adminConsole.envs[3].name | string | `"VERSIONING_TYPES"` |  |
| adminConsole.envs[3].value | string | `"default,edp"` |  |
| adminConsole.envs[4].name | string | `"CI_TOOLS"` |  |
| adminConsole.envs[4].value | string | `"Jenkins,GitLab CI"` |  |
| adminConsole.envs[5].name | string | `"PERF_DATA_SOURCES"` |  |
| adminConsole.envs[5].value | string | `"Sonar,Jenkins,GitLab"` |  |
| adminConsole.image | string | `"epamedp/edp-admin-console"` | EDP image. The released image can be found on [Dockerhub](https://hub.docker.com/r/epamedp/edp-admin-console) |
| adminConsole.imagePullPolicy | string | `"IfNotPresent"` |  |
| adminConsole.imagePullSecrets | string | `nil` | Secrets to pull from private Docker registry |
| adminConsole.imageStreamUrlMask | string | `"/console/project/{namespace}/browse/images/{stream}"` |  |
| adminConsole.ingress.annotations | object | `{}` |  |
| adminConsole.ingress.pathType | string | `"Prefix"` |  |
| adminConsole.ingress.tls | list | `[]` |  |
| adminConsole.nodeSelector | object | `{}` |  |
| adminConsole.projectUrlMask | string | `"/console/project/{namespace}/overview"` |  |
| adminConsole.resources.limits.memory | string | `"256Mi"` |  |
| adminConsole.resources.requests.cpu | string | `"50m"` |  |
| adminConsole.resources.requests.memory | string | `"64Mi"` |  |
| adminConsole.tolerations | list | `[]` |  |
| adminConsole.version | string | `"2.14.0"` | EDP image. The released image can be found on [Dockerhub](https://hub.docker.com/r/epamedp/edp-admin-console/tags) |
| affinity | object | `{}` |  |
| annotations | object | `{}` |  |
| global.dnsWildCard | string | `nil` | a cluster DNS wildcard name |
| global.edpName | string | `""` | namespace or a project name (in case of OpenShift) |
| global.openshift.deploymentType | string | `"deployments"` | Which type of kind will be deployed to Openshift (values: deployments/deploymentConfigs) |
| global.platform | string | `"openshift"` | platform type that can be "kubernetes" or "openshift |
| global.version | string | `"3.0.0"` | EDP version |
| global.webConsole.url | string | `nil` | URL to OpenShift/Kubernetes Web console |
| image.repository | string | `"epamedp/admin-console-operator"` | EDP reconciler Docker image name. The released image can be found on [Dockerhub](https://hub.docker.com/r/epamedp/admin-console-operator) |
| image.tag | string | `nil` | EDP reconciler Docker image tag. The released image can be found on [Dockerhub](https://hub.docker.com/r/epamedp/admin-console-operator/tags) |
| imagePullPolicy | string | `"IfNotPresent"` |  |
| name | string | `"admin-console-operator"` | component name |
| nodeSelector | object | `{}` |  |
| resources.limits.memory | string | `"192Mi"` |  |
| resources.requests.cpu | string | `"50m"` |  |
| resources.requests.memory | string | `"64Mi"` |  |
| tolerations | list | `[]` |  |

