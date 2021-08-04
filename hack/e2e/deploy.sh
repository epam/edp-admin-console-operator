# 1 arg - namespace to deploy testable data
# 2 arg - admin-console tag
# 3 arg - admin-console-operator tag
# 4 arg - docker registry host
deploy() {
helm install admin-console-operator --wait --timeout=600s --namespace $1 \
--set name=admin-console-operator \
--set global.edpName=$1 \
--set global.platform=kubernetes \
--set image.name=$4/admin-console-operator \
--set image.version=$3 \
--set global.database.deploy=false \
--set global.dnsWildCard=stub \
--set global.version=$2 \
--set adminConsole.authKeycloakEnabled=false \
--set adminConsole.image=$4/edp-admin-console \
--set adminConsole.version=$2 \
deploy-templates  --debug
}

deploy "$1" "$2" "$3" "$4"