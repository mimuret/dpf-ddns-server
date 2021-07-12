package zone

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/miekg/dns"
	"github.com/mimuret/golang-iij-dpf/pkg/api"
	"github.com/mimuret/golang-iij-dpf/pkg/apis/core"
)

type DpfZoneReader struct {
	sync.Mutex
	client api.ClientInterface
	cache  atomic.Value
	log    api.Logger
}

func NewDpfZoneReader(cl api.ClientInterface, log api.Logger) *DpfZoneReader {
	r := &DpfZoneReader{}
	r.client = cl
	r.log = log
	zoneCache := map[string]*core.Zone{}
	r.cache.Store(zoneCache)
	return r
}

// get zone data
func (z *DpfZoneReader) GetZone(msg *dns.Msg) (*DpfZone, error) {
	if len(msg.Question) != 1 {
		return nil, fmt.Errorf("zone section must be 1")
	}
	name := dns.CanonicalName(msg.Question[0].Name)

	cZone, err := z.GetCache(name)
	if err != nil {
		return nil, err
	}
	if cZone == nil {
		return nil, nil
	}
	return NewDpfZone(z.client, cZone, z.log)
}

func (z *DpfZoneReader) GetCache(name string) (*core.Zone, error) {
	z.Lock()
	defer z.Unlock()
	cache := z.cache.Load().(map[string]*core.Zone)
	if cache[name] != nil {
		zoneList := &core.ZoneList{}
		params := &core.ZoneListSearchKeywords{
			Name: api.KeywordsString{name},
		}
		if _, err := z.client.ListALL(zoneList, params); err != nil {
			return nil, fmt.Errorf("failed to get zone data: %w", err)
		}
		if cache == nil {
			cache = map[string]*core.Zone{}
		}
		for _, cZone := range zoneList.Items {
			cache[cZone.Name] = &cZone
		}
		z.cache.Store(cache)
	}

	return cache[name], nil
}
