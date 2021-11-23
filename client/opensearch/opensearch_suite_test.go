package opensearch_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestOpenSearch(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OpenSearch Suite")
}
