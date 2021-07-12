package zone

import (
	"fmt"

	"github.com/miekg/dns"
	"github.com/mimuret/dnsutils"
	"github.com/mimuret/dnsutils/ddns"
	"github.com/mimuret/golang-iij-dpf/pkg/api"
	"github.com/mimuret/golang-iij-dpf/pkg/apis/core"
	"github.com/mimuret/golang-iij-dpf/pkg/apis/zones"
	"github.com/mimuret/golang-iij-dpf/pkg/apiutils"
	"go.uber.org/multierr"
)

var _ dnsutils.ZoneInterface = &DpfZone{}
var _ ddns.GetZoneInterface = &DpfZone{}
var _ ddns.UpdateInterface = &DpfZone{}

type DpfZone struct {
	cl    api.ClientInterface
	cZone *core.Zone
	*dnsutils.Zone
	log        api.Logger
	operations Operations
}

func NewDpfZone(cl api.ClientInterface, cz *core.Zone, log api.Logger) (*DpfZone, error) {
	z := &DpfZone{
		cl:    cl,
		cZone: cz,
		log:   log,
		Zone:  dnsutils.NewZone(cz.Name, dns.ClassINET),
	}
	return z, z.Sync()
}

func (z *DpfZone) GetZone(msg *dns.Msg) (dnsutils.ZoneInterface, error) {
	return z, nil
}

func (z *DpfZone) Sync() error {
	list := &zones.RecordList{}
	list.ZoneId = z.cZone.Id

	if reqId, err := z.cl.ListALL(list, nil); err != nil {
		return fmt.Errorf("failed to get records from API reqId: %s : %w", reqId, err)
	}

	zone := dnsutils.NewZone(z.cZone.Name, dns.ClassINET)
	for _, item := range list.Items {
		set := NewDpfRRSet(item)
		nn, ok := zone.GetRootNode().GetNameNode(item.Name)
		if !ok || nn == nil {
			nn = dnsutils.NewNameNode(item.Name, z.GetClass())
		}
		if err := nn.SetRRSet(set); err != nil {
			return fmt.Errorf("failed to set rrset: %w", err)
		}
		if err := z.GetRootNode().SetNameNode(nn); err != nil {
			return fmt.Errorf("failed to set node: %w", err)
		}
	}
	z.Zone = zone
	return nil
}

// add or create RR
func (d *DpfZone) AddRR(rr dns.RR) error {
	ope := d.GetOperationOrCreate(rr.Header().Name, rr.Header().Rrtype)
	if ope == nil {
		ope = &Operation{
			Set: NewDpfRRSetFromRR(rr),
		}
		d.operations = append(d.operations, ope)
	}
	ope.Set.AddRR(rr)
	return nil
}

// replace rrset
func (d *DpfZone) ReplaceRRSet(set dnsutils.RRSetInterface) error {
	ope := d.GetOperationOrCreate(set.GetName(), set.GetRRtype())
	if ope == nil {
		ope = &Operation{}
		d.operations = append(d.operations, ope)
	}
	ope.Set = NewDpfRRSetFromRRSet(set)
	return nil

}

// remove zone apex name rr other than SOA,NS
func (d *DpfZone) RemoveNameApex(name string) error {
	nn, ok := d.GetRootNode().GetNameNode(name)
	if ok {
		err := nn.IterateNameRRSet(func(set dnsutils.RRSetInterface) error {
			if set.GetRRtype() == dns.TypeSOA || set.GetRRtype() == dns.TypeNS {
				return nil
			}
			return d.RemoveRRSet(name, set.GetRRtype())
		})
		if err != nil {
			return fmt.Errorf("failed to remove apex rrsets: %w", err)
		}
	}
	return nil
}

// remove name rr ignore SOA, NS
func (d *DpfZone) RemoveName(name string) error {
	nn, ok := d.GetRootNode().GetNameNode(name)
	if ok {
		err := nn.IterateNameRRSet(func(set dnsutils.RRSetInterface) error {
			return d.RemoveRRSet(name, set.GetRRtype())
		})
		if err != nil {
			return fmt.Errorf("failed to remove name: %w", err)
		}
	}
	return nil
}

// remove name rr ignore SOA, NS
func (d *DpfZone) RemoveRRSet(name string, rrtype uint16) error {
	ope := d.GetOperationOrCreate(name, rrtype)
	if ope == nil {
		return nil
	}
	ope.Set.RData = nil
	return nil
}

// remove name rr ignore SOA, NS
func (d *DpfZone) RemoveRR(rr dns.RR) error {
	ope := d.GetOperationOrCreate(rr.Header().Name, rr.Header().Rrtype)
	if ope == nil {
		return nil
	}
	if err := ope.Set.RemoveRR(rr); err != nil {
		return err
	}
	return nil
}

// it can rollback zone records when UpdateProcessing returns error
func (d *DpfZone) UpdateFailedPostProcess(err error) {
	// notthing
}

func (d *DpfZone) rollBack() {
	reqId, err := d.cl.Cancel(d.cZone)
	if err != nil {
		d.log.Errorf("failed to rollback request RequestId: %s err: %w", reqId, err)
	}
	job, err := apiutils.WaitJob(d.cl, reqId)
	if err != nil {
		d.log.Errorf("failed to rollback process watch RequestId: %s err: %v", reqId, err)
	}
	if job.GetError() != nil {
		d.log.Errorf("failed to rollback process RequestId: %s err: %v", reqId, err)
	}

}

// it can apply zone records when UpdateProcessing is successful
func (d *DpfZone) UpdateSuccessPostProcess() error {
	var err error
	// start async records operations
	for _, ope := range d.operations {
		if ope.Set.Id == "" && len(ope.Set.RData) > 0 {
			if ope.ReqID, err = d.cl.Create(&ope.Set.Record, nil); err != nil {
				return fmt.Errorf("failed to create record name %s rrtype %s: %w", ope.Set.GetName(), dns.TypeToString[ope.Set.GetRRtype()], err)
			}
		} else if len(ope.Set.RData) > 0 {
			if ope.ReqID, err = d.cl.Update(&ope.Set.Record, nil); err != nil {
				return fmt.Errorf("failed to update record name %s rrtype %s: %w", ope.Set.GetName(), dns.TypeToString[ope.Set.GetRRtype()], err)
			}
		} else if len(ope.Set.RData) == 0 {
			if ope.ReqID, err = d.cl.Delete(&ope.Set.Record); err != nil {
				return fmt.Errorf("failed to delete record name %s rrtype %s: %w", ope.Set.GetName(), dns.TypeToString[ope.Set.GetRRtype()], err)
			}
		}
	}
	// wait async records operations
	if err := d.WaitAsyncRequest(); err != nil {
		d.rollBack()
		return err
	}

	// start async zone apply operation
	zoneApply := &core.ZoneApply{
		Id:          d.cZone.Id,
		Description: "update by DDNS server",
	}
	if err := d.WaitAsyncRequest(); err != nil {
		d.rollBack()
		return err
	}
	reqId, err := d.cl.Apply(zoneApply, nil)
	if err != nil {
		return fmt.Errorf("failed to zone apply request RequestId: %s err: %w", reqId, err)
	}
	job, err := apiutils.WaitJob(d.cl, reqId)
	if err != nil {
		d.log.Errorf("failed to zone apply process watch RequestId: %s err: %v", reqId, err)
	}
	if job.GetError() != nil {
		d.log.Errorf("failed to zone apply process RequestId: %s err: %v", reqId, err)
	}
	return nil
}

func (d *DpfZone) WaitAsyncRequest() error {
	var results error
	for _, ope := range d.operations {
		job, err := apiutils.WaitJob(d.cl, ope.ReqID)
		// job already get
		if ok, _ := api.IsNotFound(err); ok {
			continue
		}
		// get job failed
		if err != nil {
			results = multierr.Append(results, err)
		}
		// async process failed
		if job.GetError() != nil {
			results = multierr.Append(results, job.GetError())
		}
	}
	return results
}

func (d *DpfZone) IsPrecheckSupportedRtype(rrtype uint16) bool {
	switch rrtype {
	case dns.TypeANY, dns.TypeNone, dns.TypeSOA, dns.TypeA,
		dns.TypeAAAA, dns.TypeCAA, dns.TypeCNAME,
		dns.TypeDS, dns.TypeNS, dns.TypeMX, dns.TypeNAPTR,
		dns.TypeSRV, dns.TypeTXT, dns.TypeTLSA, dns.TypePTR:
		return true
	}
	return false
}

func (d *DpfZone) IsUpdateSupportedRtype(rrtype uint16) bool {
	switch rrtype {
	case dns.TypeANY, dns.TypeNone, dns.TypeSOA, dns.TypeA,
		dns.TypeAAAA, dns.TypeCAA, dns.TypeCNAME,
		dns.TypeDS, dns.TypeNS, dns.TypeMX, dns.TypeNAPTR,
		dns.TypeSRV, dns.TypeTXT, dns.TypeTLSA, dns.TypePTR:
		return true
	}
	return false

}

func (d *DpfZone) GetOperationOrCreate(name string, rrtype uint16) *Operation {
	ope := d.operations.Get(name, rrtype)
	if ope == nil {
		nn, ok := d.GetRootNode().GetNameNode(name)
		if ok {
			set := nn.GetRRSet(rrtype).(*DpfRRSet)
			if set != nil {
				ope = &Operation{
					Set: set,
				}
				d.operations = append(d.operations, ope)
			}
		}
	}
	return ope
}
