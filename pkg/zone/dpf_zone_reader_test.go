package zone_test

import (
	"github.com/mimuret/dpf-ddns-server/pkg/zone"
	"github.com/mimuret/golang-iij-dpf/pkg/api"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DpfZoneReader", func() {
	var (
		cl     = &api.NopClient{}
		reader = zone.NewDpfZoneReader(cl, &api.StdLogger{})
	)
	Context("New", func() {
		Expect(reader).NotTo(BeNil())
	})
})
