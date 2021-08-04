# 1 arg - namespace to deploy testable data
run() {
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 E2E_TESTS_ENABLED=true WATCH_NAMESPACE="$1" go test ./test/... -v ./test/... cmd/manager/main.go
}

run "$1"