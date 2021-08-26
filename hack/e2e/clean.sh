# 1 arg - namespace to deploy testable data
clean() {
	local ns=$( kubectl get ns "$1" -o=jsonpath='{.metadata.name}' --ignore-not-found=true )
	if [ ! ${ns} ]; then
		return 0
	fi
	
	kubectl delete ns $1 --ignore-not-found=true
}

clean "$1"