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
clean:
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
execute: clean setup_prerequisite deploy run_tests clean
