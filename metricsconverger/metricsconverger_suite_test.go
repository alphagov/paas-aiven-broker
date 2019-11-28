package metricsconverger_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMetricsconverger(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metricsconverger Suite")
}
