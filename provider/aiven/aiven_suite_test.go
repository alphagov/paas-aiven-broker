package aiven_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestAiven(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Aiven Suite")
}
