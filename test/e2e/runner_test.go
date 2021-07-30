package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"os"
	"strconv"
	"testing"
)

func TestAdminConsole(t *testing.T) {
	if enabled := os.Getenv("E2E_TESTS_ENABLED"); enabled == "" {
		t.Skip("Skipping E2E tests.")
	}

	if enabled, _ := strconv.ParseBool(os.Getenv("E2E_TESTS_ENABLED")); !enabled {
		t.Skip("Skipping E2E tests.")
	}

	RegisterFailHandler(Fail)
	RunSpecs(t, "Admin Console Suite")
}
