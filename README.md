# Admin Console Operator

Get acquainted with the Admin Console Operator and the installation process as well as the local development, 
and architecture scheme.

## Overview

Admin Console operator is an EDP operator that is responsible for installing and configuring EDP Admin Console. 
Operator installation can be applied on two container orchestration platforms: OpenShift and Kubernetes.

_**NOTE:** Operator is platform-independent, that is why there is a unified instruction for deploying._

## Prerequisites
1. Linux machine or Windows Subsystem for Linux instance with [Helm 3](https://helm.sh/docs/intro/install/) installed;
2. Cluster admin access to the cluster;
3. EDP project/namespace is deployed by following one of the instructions: [edp-install-openshift](https://github.com/epmd-edp/edp-install/blob/master/documentation/openshift_install_edp.md#edp-project) or [edp-install-kubernetes](https://github.com/epmd-edp/edp-install/blob/master/documentation/kubernetes_install_edp.md#edp-namespace).

## Installation
In order to install the Admin Console operator, follow the steps below:

1. To add the Helm EPAMEDP Charts for local client, run "helm repo add":
     ```bash
     helm repo add epamedp https://chartmuseum.demo.edp-epam.com/
     ```
2. Choose available Helm chart version:
     ```bash
     helm search repo epamedp/admin-console-operator
     NAME                               CHART VERSION   APP VERSION     DESCRIPTION
     epamedp/admin-console-operator      v2.4.0                          Helm chart for Golang application/service deplo...
     ```
  
    _**NOTE:** It is highly recommended to use the latest released version._

3. Deploy operator:

   Full available chart parameters list:
   ```
    - chart_version                                 # a version of Admin Console operator Helm chart;
    - global.edpName                                # a namespace or a project name (in case of OpenShift);
    - global.platform                               # openshift or kubernetes;
    - global.dnsWildCard                            # Developers of your tenant separated by comma (,) (eg --set 'global.developers={test@example.com}');
    - global.database.deploy                        # Deploy DB to current project or use from another;
    - global.database.name                          # Name of DB;
    - global.database.host                          # Host to DB (<db-name>.<namespace>);
    - global.database.port                          # Port of DB;
    - global.webConsole.url                         # URL to OpenShift/Kubernetes Web console;
    - image.name                                    # EDP image. The released image can be found on [Dockerhub](https://hub.docker.com/repository/docker/epamedp/admin-console-operator);
    - image.version                                 # EDP tag. The released image can be found on [Dockerhub](https://hub.docker.com/repository/docker/epamedp/admin-console-operator/tags);
    - adminConsole.image                            # EDP image. The released image can be found on [Dockerhub](https://hub.docker.com/repository/docker/epamedp/edp-admin-console);
    - adminConsole.version                          # EDP image. The released image can be found on [Dockerhub](https://hub.docker.com/repository/docker/epamedp/edp-admin-console/tags);
    - adminConsole.imagePullSecrets                 # Secrets to pull from private Docker registry;
    - adminConsole.basePath                         # Base path for Admin Console URL.
    - adminConsole.authKeycloakEnabled              # Authentication Keycloak enabled/disabled.
    - adminConsole.keycloakUrl                      # URL to Keycloak;
   ```

4. Install operator in the <edp_cicd_project> namespace with the helm command; find below the installation command example:
    ```bash
    helm install admin-console-operator epamedp/admin-console-operator --version <chart_version> --namespace <edp_cicd_project> --set name=admin-console-operator --set global.edpName=<edp_cicd_project> --set global.platform=<platform_type>
    ```
5. Check the <edp_cicd_project> namespace that should contain operator deployment with your operator in a running status.

## Local Development
In order to develop the operator, first set up a local environment. For details, please refer to the [Local Development](documentation/local_development.md) page.

### Related Articles

- [Architecture Scheme of Admin Console Operator](documentation/arch.md)