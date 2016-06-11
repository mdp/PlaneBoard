package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/miekg/dns"
)

// Config holds our basic server config, pull from config.toml
type Config struct {
	ConsumerKey    string
	ConsumerSecret string
	TokenKey       string
	TokenSecret    string
	DBFile         string
	Host           string
	Port           int
	Groups         map[string]Group
}

// Group of accounts
type Group struct {
	Accounts []string
}

var config Config
var dataStore *DataStore

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

// Names returns either a single account name or all the members of a group
func (rq *RecordQuery) Names() []string {
	if rq.Topic {
		return config.Groups[strings.Title(rq.Name)].Accounts
	}
	return []string{rq.Name}
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

func handleRequest(w dns.ResponseWriter, r *dns.Msg) {
	q := r.Question[0]
	m := new(dns.Msg)
	m.SetReply(r)
	switch q.Qtype {
	case dns.TypeA:
		m.Answer = make([]dns.RR, 1)
		ip := net.IPv4(31, 33, 7, 3)
		m.Answer[0] = &dns.A{Hdr: dns.RR_Header{Name: m.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0}, A: ip}
	case dns.TypeTXT:
		log.Printf("Incoming request - %s", q.Name)
		query, err := parseRecordName(q.Name, config.Host)
		if err != nil {
			break
		}
		tweet := dataStore.FindTweet(query.Names(), query.Before, query.Page)
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

func serveDNS() {
	dns.HandleFunc(".", handleRequest)
	addr := fmt.Sprintf(":%d", config.Port)
	server := &dns.Server{Addr: addr, Net: "udp"}
	log.Printf("Serving")
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func streamTweets() {
	consumer := oauth1.NewConfig(config.ConsumerKey, config.ConsumerSecret)
	token := oauth1.NewToken(config.TokenKey, config.TokenSecret)
	httpClient := consumer.Client(oauth1.NoContext, token)
	demux := twitter.NewSwitchDemux()
	demux.Tweet = func(tweet *twitter.Tweet) {
		log.Printf("Incoming tweet: @%s - '%s'", tweet.User.ScreenName, tweet.Text)
		dataStore.AddTweet(tweet)
	}

	// Twitter client
	client := twitter.NewClient(httpClient)
	params := &twitter.StreamUserParams{
		StallWarnings: twitter.Bool(true),
	}
	stream, err := client.Streams.User(params)
	if err != nil {
		log.Fatal("Error", err)
	}
	go demux.HandleChan(stream.Messages)
}

func setupConfig() Config {
	conf := Config{}
	log.Println("Setting up Toml")
	if _, err := toml.DecodeFile("config.toml", &conf); err != nil {
		log.Fatal("Toml parsing error", err)
	}
	return conf
}

func main() {
	terminate := make(chan os.Signal)
	signal.Notify(terminate, os.Interrupt)
	config = setupConfig()
	arg := os.Args[len(os.Args)-1]
	if arg == "auth" {
		AuthWithTwitter(config.ConsumerKey, config.ConsumerSecret)
		return
	}
	dataStore, _ = SetupDataStore(config.DBFile)
	defer dataStore.Close()

	go serveDNS()
	go streamTweets()
	go func() {
		for {
			dataStore.Clean(time.Now().Unix() - 60*60*3)
			time.Sleep(15 * time.Minute)
		}
	}()

	<-terminate
	log.Printf("PlaneBoard: signal received, stopping")
}
