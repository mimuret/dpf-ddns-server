package zone

import (
	"fmt"
	"strings"

	"github.com/miekg/dns"
	"github.com/mimuret/dnsutils"
	"github.com/mimuret/golang-iij-dpf/pkg/apis/zones"
)

var (
	ErrDuplicate = fmt.Errorf("Err Duplicate")
)
var _ dnsutils.RRSetInterface = &DpfRRSet{}

type DpfRRSet struct {
	zones.Record
}

func NewDpfRRSet(r zones.Record) *DpfRRSet {
	return &DpfRRSet{r}
}
func NewDpfRRSetFromRR(rr dns.RR) *DpfRRSet {
	set := &DpfRRSet{
		zones.Record{
			Name:   dns.CanonicalName(rr.Header().Name),
			TTL:    int32(rr.Header().Ttl),
			RRType: zones.Type(dns.TypeToString[rr.Header().Rrtype]),
		},
	}
	if err := set.AddRR(rr); err != nil {
		return nil
	}
	return set
}
func NewDpfRRSetFromRRSet(set dnsutils.RRSetInterface) *DpfRRSet {
	newSet := &DpfRRSet{
		zones.Record{
			Name:   set.GetName(),
			TTL:    int32(set.GetTTL()),
			RRType: zones.Type(dns.TypeToString[set.GetRRtype()]),
		},
	}
	return newSet
}

// return canonical name
func (r *DpfRRSet) GetName() string {
	return r.Name
}

// return rtype
func (r *DpfRRSet) GetTTL() uint32 {
	return uint32(r.TTL)
}

// return rtype
func (r *DpfRRSet) SetTTL(ttl uint32) {
	r.TTL = int32(ttl)
}

// return rtype
func (r *DpfRRSet) GetRRtype() uint16 {
	if r.RRType != zones.TypeANAME {
		return dns.StringToType[string(r.RRType)]
	}
	return 65280
}

// return dns.Class
func (r *DpfRRSet) GetClass() dns.Class {
	return dns.ClassINET
}

// return rr slice
func (r *DpfRRSet) GetRRs() []dns.RR {
	if r.RRType == "ANAME" {
		return []dns.RR{}
	}
	rrs := []dns.RR{}
	for _, rdata := range r.RData {
		rr, _ := dns.NewRR(fmt.Sprintf("%s %d IN %s %s", r.Name, r.TTL, r.RRType, rdata.Value))
		rrs = append(rrs, rr)
	}
	return rrs
}

// number of rdata
func (r *DpfRRSet) Len() int {
	return len(r.Record.RData)
}

func (r *DpfRRSet) AddRR(rr dns.RR) error {
	if err := r.addCheck(rr); err != nil {
		return err
	}
	if r.Len() > 0 {
		switch r.RRType {
		case "CNAME", "SOA":
			return dnsutils.ErrConflict
		}
	}
	r.RData = append(r.RData, zones.RecordRDATA{Value: dnsutils.GetRDATA(rr)})
	return nil
}

func (r *DpfRRSet) addCheck(rr dns.RR) error {
	if !dnsutils.Equals(r.Name, rr.Header().Name) {
		return dnsutils.ErrRRName
	}
	if rr.Header().Rrtype != r.GetRRtype() {
		return dnsutils.ErrRRType
	}
	if rr.Header().Class != dns.ClassINET {
		return dnsutils.ErrClass
	}
	for _, rdata := range r.RData {
		if rdata.Value == dnsutils.GetRDATA(rr) {
			return ErrDuplicate
		}
	}

	return nil
}

func (r *DpfRRSet) RemoveRR(rr dns.RR) error {
	v := strings.SplitN(rr.String(), "\t", 5)
	new := []zones.RecordRDATA{}
	for _, rdata := range r.RData {
		if v[4] != rdata.Value {
			new = append(new, rdata)
		}
	}
	r.RData = new
	return nil
}

func (r *DpfRRSet) Copy() dnsutils.RRSetInterface {
	c := &DpfRRSet{}
	c.Record = *r.Record.DeepCopy()
	return c
}

func (r *DpfRRSet) ReplaceRRs(rrs []dns.RR) error {
	newRDATA := zones.RecordRDATAs{}
	for _, rr := range rrs {
		if err := r.addCheck(rr); err != nil {
			return err
		}
		newRDATA = append(newRDATA, zones.RecordRDATA{
			Value: dnsutils.GetRDATA(rr),
		})
	}
	if len(newRDATA) > 0 {
		switch r.RRType {
		case "CNAME", "SOA":
			return dnsutils.ErrConflict
		}
	}
	r.RData = newRDATA
	return nil
}
