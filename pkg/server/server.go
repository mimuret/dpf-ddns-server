package server

import (
	"context"

	"github.com/miekg/dns"
	"github.com/mimuret/dnsutils/ddns"
	"github.com/mimuret/dpf-ddns-server/pkg/zone"
	"github.com/mimuret/golang-iij-dpf/pkg/api"
)

var _ dns.Handler = &Server{}

type Server struct {
	tcp    dns.Server
	udp    dns.Server
	listen string
	reader *zone.DpfZoneReader
	log    api.Logger
}

func New(listen string, reader *zone.DpfZoneReader, logger api.Logger) *Server {
	s := &Server{}
	s.log = logger
	s.tcp = dns.Server{
		Addr:    listen,
		Net:     "tcp",
		Handler: s,
	}
	s.udp = dns.Server{
		Addr:    listen,
		Net:     "udp",
		Handler: s,
	}
	return s
}

func (s *Server) Run(ctx context.Context) error {
	var ch = make(chan error, 2)
	go func() {
		ch <- s.tcp.ListenAndServe()
	}()
	defer s.tcp.Shutdown()
	go func() {
		ch <- s.udp.ListenAndServe()
	}()
	defer s.udp.Shutdown()
	select {
	case <-ctx.Done():
	case err := <-ch:
		if err != nil {
			s.log.Errorf("error: %s", err)
			return err
		}
	}
	return nil
}

func (s *Server) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	res := &dns.Msg{}
	res.SetReply(r)
	if r.MsgHdr.Opcode != dns.OpcodeUpdate {
		res.Rcode = dns.RcodeNotImplemented
		w.WriteMsg(res)
		return
	}
	zone, err := s.reader.GetZone(r)
	if err != nil {
		res.Rcode = dns.RcodeRefused
		w.WriteMsg(res)
		return
	}
	d := ddns.NewDDNS(zone, zone)
	rcode, err := d.ServeUpdate(r)
	if err != nil {
		res.Rcode = dns.RcodeServerFailure
		w.WriteMsg(res)
		return
	}
	res.Rcode = rcode
	w.WriteMsg(res)
	return
}
