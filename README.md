[![codecov](https://codecov.io/gh/epam/edp-admin-console-operator/branch/master/graph/badge.svg?token=5EDGDQXLLA)](https://codecov.io/gh/epam/edp-admin-console-operator)

|![](https://upload.wikimedia.org/wikipedia/commons/thumb/1/17/Warning.svg/156px-Warning.svg.png) | This operator is deprecated starting from EDP v2.13
|---|---|

# Admin Console Operator

| :heavy_exclamation_mark: Please refer to [EDP documentation](https://epam.github.io/edp-install/) to get the notion of the main concepts and guidelines. |
| --- |

Get acquainted with the Admin Console Operator and the installation process as well as the local development, and architecture scheme.

## Overview

Admin Console operator is an EDP operator that is responsible for installing and configuring EDP Admin Console. Operator installation can be applied on two container orchestration platforms: OpenShift and Kubernetes.

_**NOTE:** Operator is platform-independent, that is why there is a unified instruction for deploying._

## Prerequisites

1. Linux machine or Windows Subsystem for Linux instance with [Helm 3](https://helm.sh/docs/intro/install/) installed;
2. Cluster admin access to the cluster;
3. EDP project/namespace is deployed by following the [Install EDP](https://epam.github.io/edp-install/operator-guide/install-edp/) instruction.

## Installation
In order to install the Admin Console operator, follow the steps below:

1. To add the Helm EPAMEDP Charts for local client, run "helm repo add":
     ```bash
     helm repo add epamedp https://epam.github.io/edp-helm-charts/stable
     ```
2. Choose available Helm chart version:
     ```bash
     helm search repo epamedp/admin-console-operator -l
     NAME                               CHART VERSION   APP VERSION     DESCRIPTION
     epamedp/admin-console-operator     2.13.0          2.13.0          A Helm chart for EDP Admin Console Operator
     epamedp/admin-console-operator     2.12.0          2.12.0          A Helm chart for EDP Admin Console Operator
     ```

    _**NOTE:** It is highly recommended to use the latest released version._

3. Full chart parameters available in [deploy-templates/README.md](deploy-templates/README.md).

4. Install operator in the <edp-project> namespace with the helm command; find below the installation command example:
    ```bash
    helm install admin-console-operator epamedp/admin-console-operator --version <chart_version> --namespace <edp-project> --set name=admin-console-operator --set global.edpName=<edp-project> --set global.platform=<platform_type>
    ```
5. Check the <edp-project> namespace that should contain operator deployment with your operator in a running status.

## Local Development

In order to develop the operator, first set up a local environment. For details, please refer to the [Developer Guide](https://epam.github.io/edp-install/developer-guide/local-development/) page.

Development versions are also available, please refer to the [snapshot helm chart repository](https://epam.github.io/edp-helm-charts/snapshot/) page.

### Related Articles

- [Architecture Scheme of Admin Console Operator](documentation/arch.md)
- [Install EDP](https://epam.github.io/edp-install/operator-guide/install-edp/)
- [Set Up Kubernetes](https://epam.github.io/edp-install/operator-guide/kubernetes-cluster-settings/)
- [Set Up OpenShift](https://epam.github.io/edp-install/operator-guide/openshift-cluster-settings/)
