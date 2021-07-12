package zone_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestDNSUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "dnsutils Suite")
}

var _ = Describe("DpfZone", func() {
})
