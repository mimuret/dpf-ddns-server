package zone_test

import (
	"github.com/miekg/dns"
	"github.com/mimuret/dpf-ddns-server/pkg/zone"
	"github.com/mimuret/golang-iij-dpf/pkg/apis/zones"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func MustNewRR(s string) dns.RR {
	rr, err := dns.NewRR(s)
	if err != nil {
		panic(err)
	}
	return rr
}

var _ = Describe("DpfRRSet", func() {
	var (
		a11    = MustNewRR("example.jp. 300 IN A 192.168.0.1")
		a12    = MustNewRR("example.jp. 300 IN A 192.168.0.2")
		soa1   = MustNewRR("example.jp. 300 IN SOA localhost. root.localhost. 1 3600 600 86400 900")
		soa2   = MustNewRR("example.jp. 300 IN SOA localhost. root.localhost. 2 3600 600 86400 900")
		cname1 = MustNewRR("example.jp. 300 IN CNAME www1.example.jp.")
		cname2 = MustNewRR("example.jp. 300 IN CNAME www2.example.jp.")
		aset   = zone.NewDpfRRSet(zones.Record{
			Name:   "example.jp.",
			RRType: "A",
			TTL:    300,
			RData: []zones.RecordRDATA{
				{
					Value: "192.168.0.1",
				},
				{
					Value: "192.168.0.2",
				},
			},
			State: zones.RecordStateApplied,
		})
		anameSet = zone.NewDpfRRSet(zones.Record{
			Name:   "example.jp.",
			RRType: "ANAME",
			TTL:    300,
			RData: []zones.RecordRDATA{
				{
					Value: "www.example.net.",
				},
			},
			State: zones.RecordStateApplied,
		})
	)
	Context("GetName", func() {
		It("return rrset name", func() {
			Expect(aset.GetName()).To(Equal("example.jp."))
		})
	})
	Context("GetRRtype", func() {
		It("return uint16", func() {
			Expect(aset.GetRRtype()).To(Equal(dns.TypeA))
		})
		When("type is ANAME", func() {
			It("return 65280", func() {
				Expect(anameSet.GetRRtype()).To(Equal(uint16(65280)))
			})
		})
	})
	Context("GetClass", func() {
		It("return ClassINET", func() {
			Expect(aset.GetClass()).To(Equal(dns.Class(dns.ClassINET)))
		})
	})
	Context("GetRRs", func() {
		It("return RR", func() {
			Expect(aset.GetRRs()).To(Equal([]dns.RR{a11, a12}))
		})
		When("type is ANAME", func() {
			It("return []dns.RR{}", func() {
				Expect(anameSet.GetRRs()).To(Equal([]dns.RR{}))
			})
		})
	})
	Context("Len", func() {
		It("return the number of rdata", func() {
			Expect(aset.Len()).To(Equal(2))
			Expect(anameSet.Len()).To(Equal(1))
		})
	})
	Context("test for AddRR (Normal)", func() {
		It("can be add uniq RR", func() {
			rrset := zone.NewDpfRRSet(zones.Record{
				Name:   "example.jp.",
				RRType: "A",
				TTL:    300,
				RData:  []zones.RecordRDATA{},
				State:  zones.RecordStateApplied,
			})
			err := rrset.AddRR(a11)
			Expect(err).To(BeNil())
			Expect(rrset.GetRRs()).To(Equal([]dns.RR{a11}))
			err = rrset.AddRR(a11)
			Expect(err).To(BeNil())
			Expect(rrset.GetRRs()).To(Equal([]dns.RR{a11}))
			err = rrset.AddRR(a12)
			Expect(err).To(BeNil())
			Expect(rrset.GetRRs()).To(Equal([]dns.RR{a11, a12}))
		})
	})
	Context("test for AddRR(SOA RR)", func() {
		It("can not be add multiple RR", func() {
			rrset := zone.NewDpfRRSet(zones.Record{
				Name:   "example.jp.",
				RRType: "SOA",
				TTL:    300,
				RData: []zones.RecordRDATA{
					{
						Value: "localhost. root.localhost. 1 3600 600 86400 900",
					},
				},
				State: zones.RecordStateApplied,
			})
			Expect(rrset.GetRRs()).To(Equal([]dns.RR{soa1}))
			err := rrset.AddRR(soa2)
			Expect(err).NotTo(BeNil())
			Expect(rrset.GetRRs()).To(Equal([]dns.RR{soa1}))
		})
	})
	Context("test for AddRR(CNAME RR)", func() {
		It("can not be add multiple RR", func() {
			rrset := zone.NewDpfRRSet(zones.Record{
				Name:   "example.jp.",
				RRType: "CNAME",
				TTL:    300,
				RData: []zones.RecordRDATA{
					{
						Value: "www1.example.jp.",
					},
				},
				State: zones.RecordStateApplied,
			})
			Expect(rrset.GetRRs()).To(Equal([]dns.RR{cname1}))

			err := rrset.AddRR(cname2)
			Expect(err).NotTo(BeNil())
			Expect(rrset.GetRRs()).To(Equal([]dns.RR{cname1}))
		})
	})
	Context("RemoteRR", func() {
		It("can remove RR", func() {
			aset.RemoveRR(a11)
			Expect(aset.GetRRs()).To(Equal([]dns.RR{a12}))
		})
	})
	Context("Copy", func() {
		It("can remove RR", func() {
			copy := aset.Copy()
			Expect(copy).To(Equal(aset))
		})
	})
})
