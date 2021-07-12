package zone

import (
	"github.com/miekg/dns"
	"github.com/mimuret/golang-iij-dpf/pkg/api"
)

type Operations []*Operation

func (o Operations) Get(name string, rrtype uint16) *Operation {
	name = dns.CanonicalName(name)
	for _, ope := range o {
		if ope.Set.GetName() == name && ope.Set.GetRRtype() == rrtype {
			return ope
		}
	}
	return nil
}

type Operation struct {
	ReqID  string
	Action api.Action
	Set    *DpfRRSet
}
