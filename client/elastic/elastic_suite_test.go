package elastic_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestElastic(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Elastic Suite")
}
