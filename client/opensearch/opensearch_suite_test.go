package opensearch_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestOpenSearch(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OpenSearch Suite")
}
