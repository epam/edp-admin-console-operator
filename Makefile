PACKAGE=github.com/epam/edp-common/pkg/config
CURRENT_DIR=$(shell pwd)
DIST_DIR=${CURRENT_DIR}/dist
BIN_NAME=go-binary

HOST_OS:=$(shell go env GOOS)
HOST_ARCH:=$(shell go env GOARCH)

VERSION=$(shell cat ${CURRENT_DIR}/VERSION)
BUILD_DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT=$(shell git rev-parse HEAD)
GIT_TAG=$(shell if [ -z "`git status --porcelain`" ]; then git describe --exact-match --tags HEAD 2>/dev/null; fi)
KUBECTL_VERSION=$(shell go list -m all | grep k8s.io/client-go| cut -d' ' -f2)

override LDFLAGS += \
  -X ${PACKAGE}.version=${VERSION} \
  -X ${PACKAGE}.buildDate=${BUILD_DATE} \
  -X ${PACKAGE}.gitCommit=${GIT_COMMIT} \
  -X ${PACKAGE}.kubectlVersion=${KUBECTL_VERSION}\

ifneq (${GIT_TAG},)
LDFLAGS += -X ${PACKAGE}.gitTag=${GIT_TAG}
endif

.DEFAULT_GOAL:=help
# set default shell
SHELL=/bin/bash -o pipefail -o errexit
help:  ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=deploy-templates/crds

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) rbac:roleName=manager-role object paths="./..."

.PHONY: validate-docs
validate-docs: api-docs helm-docs  ## Validate helm and api docs
	@git diff -s --exit-code deploy-templates/README.md || (echo "Run 'make helm-docs' to address the issue." && git diff && exit 1)
	@git diff -s --exit-code docs/api.md || (echo " Run 'make api-docs' to address the issue." && git diff && exit 1)

# Run tests
test: fmt vet
	go test ./... -coverprofile=coverage.out `go list ./...`

fmt:  ## Run go fmt
	go fmt ./...

vet:  ## Run go vet
	go vet ./...

lint: ## Run go lint
	golangci-lint run

.PHONY: build
build: clean ## build operator's binary
	CGO_ENABLED=0 GOOS=${HOST_OS} GOARCH=${HOST_ARCH} go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/${BIN_NAME} ./cmd/manager/main.go

.PHONY: clean
clean:  ## clean up
	-rm -rf ${DIST_DIR}

## //TODO E2E section below to refactor

# namespace to deploy testable data
E2E_NS="${E2E_NS:-default}"
# admin-console image tag
ADMIN_CONSOLE_TAG="${ADMIN_CONSOLE_TAG:-default}"
# admin-console-operator tag to test
ADMIN_CONSOLE_OPERATOR_TAG="${ADMIN_CONSOLE_OPERATOR_TAG:-default}"
# registry from where to fetch images
DOCKER_REGISTRY_REPO_URL="${DOCKER_REGISTRY_REPO_URL:-default}"

# sets KUBECONFIG env variable
export KUBECONFIG := "${KUBECONFIG}"

# clean up whole testable namespace (delete helm releases/delete ns)
e2e-clean:
	./hack/e2e/clean.sh "${E2E_NS}"

# set all resources required to correct admin-console-operator work
setup_prerequisite:
	./hack/e2e/e2e_prerequisite.sh "${E2E_NS}" "${DNS_WILDCARD}" "${ADMIN_CONSOLE_TAG}"

# deploy admin-console-operator chart to testable ns
deploy:
	./hack/e2e/deploy.sh "${E2E_NS}" "${ADMIN_CONSOLE_TAG}" "${ADMIN_CONSOLE_OPERATOR_TAG}" "${DOCKER_REGISTRY_REPO_URL}"

# run E2E tests
run_tests:
	./hack/e2e/tests.sh "${E2E_NS}"

# main target to run all targets needed for correct E2E testing
execute: e2e-clean setup_prerequisite deploy run_tests e2e-clean

# use https://github.com/git-chglog/git-chglog/
.PHONY: changelog
changelog: ## generate changelog
ifneq (${NEXT_RELEASE_TAG},)
	@git-chglog --next-tag v${NEXT_RELEASE_TAG} -o CHANGELOG.md v2.7.0..
else
	@git-chglog -o CHANGELOG.md v2.7.0..
endif

.PHONY: api-docs
api-docs: ## generate CRD docs
	crdoc --resources deploy-templates/crds --output docs/api.md

.PHONY: helm-docs
helm-docs: ## generate helm docs
	helm-docs

CONTROLLER_GEN = ${CURRENT_DIR}/bin/controller-gen
.PHONY: controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,v0.8.0)
# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
go get -d $(2)@$(3) ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef