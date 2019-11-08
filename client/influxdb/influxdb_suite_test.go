package influxdb_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestInfluxdb(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Influxdb Suite")
}
