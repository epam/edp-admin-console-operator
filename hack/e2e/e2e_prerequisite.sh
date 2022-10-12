# 1 arg - namespace to deploy testable data
# 2 arg - DNS-wildcard
# 3 arg - EDP version
setup() {
	kubectl create ns $1

	helm repo add epamedp https://epam.github.io/edp-helm-charts/stable

	kubectl -n $1 create configmap edp-config \
		--from-literal dns_wildcard=$2 \
		--from-literal edp_name=$1 \
		--from-literal edp_version=$3 \
		--from-literal vcs_integration_enabled=false \
		--from-literal perf_integration_enabled=false

	kubectl -n $1 create secret generic db-admin-console \
		--from-literal=username=fake_admin \
		--from-literal=password=fake_admin
}

setup "$1" "$2" "$3"
