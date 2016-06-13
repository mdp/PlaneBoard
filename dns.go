package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"
)

// DNSServer holds our Server config and state
type DNSServer struct {
	Host      string
	Port      int
	DataStore *DataStore
	Groups    map[string]Group
}

// RecordQuery represent and parsed incoming txt request
// Queries look like:
// "cachebuster123.sinceId.accountName.host.com"
// or "accountName.host.com"
type RecordQuery struct {
	Name        string
	Before      int64
	Page        int
	CacheBuster string
	Topic       bool
}

func parseFlagInt(s string, def int64) int64 {
	i, err := strconv.ParseInt(s[1:len(s)], 10, 64)
	if err != nil {
		return def
	}
	return i
}

func parseRecordName(name string, host string) (*RecordQuery, error) {
	name = strings.ToLower(name)
	if strings.Index(name, host) == -1 {
		return nil, errors.New("Query doesn't contain a matching host")
	}
	name = strings.TrimSuffix(name, host+".")
	name = strings.TrimSuffix(name, ".")
	names := strings.Split(name, ".")
	if len(names) > 5 {
		return nil, errors.New("Invalid query - Should be pPageNum.bTimestamp.accountName.host.com")
	}

	rq := &RecordQuery{
		Name:   "home",
		Before: time.Now().Unix(),
		Page:   0,
	}

	for _, name := range names {
		if strings.HasPrefix(name, "p") {
			rq.Page = int(parseFlagInt(name, 0))
		} else if strings.HasPrefix(name, "b") {
			rq.Before = parseFlagInt(name, time.Now().Unix())
		} else if strings.HasPrefix(name, "c") {
			rq.CacheBuster = name[1:len(name)]
		} else if name == "t" {
			rq.Topic = true
		} else if len(name) > 0 {
			rq.Name = name
		}
	}
	return rq, nil
}

// HandleRequest - Handle inbound DNS queries and return Tweets
func (s *DNSServer) HandleRequest(w dns.ResponseWriter, r *dns.Msg) {
	q := r.Question[0]
	m := new(dns.Msg)
	m.SetReply(r)

	switch q.Qtype {
	case dns.TypeA:
		m.Answer = make([]dns.RR, 1)
		ip := net.IPv4(3, 1, 33, 7)
		m.Answer[0] = &dns.A{Hdr: dns.RR_Header{Name: m.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0}, A: ip}
	case dns.TypeTXT:
		log.Printf("Incoming request - %s", q.Name)
		query, err := parseRecordName(q.Name, s.Host)
		if err != nil {
			break
		}

		names := []string{query.Name}
		if query.Topic {
			names = s.Groups[strings.Title(query.Name)].Accounts
		}

		tweet := s.DataStore.FindTweet(names, query.Before, query.Page)
		tweetTxt := "Sorry, no tweets found"
		if tweet != nil {
			tweetTxt = tweet.Text + " - @" + tweet.User.ScreenName
		}

		m.Answer = make([]dns.RR, 1)
		m.Answer[0] = &dns.TXT{Hdr: dns.RR_Header{
			Name:   m.Question[0].Name,
			Rrtype: dns.TypeTXT,
			Class:  dns.ClassINET, Ttl: 0},
			Txt: []string{tweetTxt}}
	default:
		log.Println("Uhandled qtype")
	}
	w.WriteMsg(m)
}

// Serve - Start the DNS Server
func (s *DNSServer) Serve() error {
	dns.HandleFunc(".", s.HandleRequest)

	addr := fmt.Sprintf(":%d", s.Port)
	server := &dns.Server{Addr: addr, Net: "udp"}
	log.Printf("DNS Serving to %s", addr)

	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
	return nil
}
