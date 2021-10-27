package elasticsearch_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestElasticSearch(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ElasticSearch Suite")
}
