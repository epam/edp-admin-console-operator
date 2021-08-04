RELEASE_NAME="admin-console-operator"

# 1 arg - namespace to deploy testable data
clean() {
	local ns=$( kubectl get ns "$1" -ojson | jq -r  '.metadata.name' )
	if [ ! ${ns} ]; then
		return 0
	fi
	
	local content=$( helm -n $1 status "${RELEASE_NAME}" -o json | jq -r  '.info.status' ) 
	if [ ${content} ]; then
		helm -n $1 delete "${RELEASE_NAME}"
	fi
	
	kubectl delete ns $1
}

clean "$1"