package main

import (
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
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

func streamTweets(d *DataStore, config *Config) {
	consumer := oauth1.NewConfig(config.ConsumerKey, config.ConsumerSecret)
	token := oauth1.NewToken(config.TokenKey, config.TokenSecret)

	httpClient := consumer.Client(oauth1.NoContext, token)
	demux := twitter.NewSwitchDemux()
	demux.Tweet = func(tweet *twitter.Tweet) {
		log.Printf("Incoming tweet: @%s - '%s'", tweet.User.ScreenName, tweet.Text)
		d.AddTweet(tweet)
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
	log.Println("Streaming tweets")

	go demux.HandleChan(stream.Messages)
}

func parseConfig() Config {
	conf := Config{}
	log.Println("Parsing config.toml")
	if _, err := toml.DecodeFile("config.toml", &conf); err != nil {
		log.Fatal("Toml parsing error", err)
	}
	return conf
}

func main() {
	log.Println("Starting server")

	terminate := make(chan os.Signal)
	signal.Notify(terminate, os.Interrupt)

	config := parseConfig()

	arg := os.Args[len(os.Args)-1]
	if arg == "auth" {
		AuthWithTwitter(config.ConsumerKey, config.ConsumerSecret)
		return
	}

	dataStore, _ := SetupDataStore(config.DBFile)
	defer dataStore.Close()

	dnsServer := &DNSServer{
		Host:      config.Host,
		DataStore: dataStore,
		Port:      config.Port,
		Groups:    config.Groups,
	}

	go dnsServer.Serve()
	go streamTweets(dataStore, &config)
	go func() {
		for {
			dataStore.Clean(time.Now().Unix() - 60*60*3)
			time.Sleep(15 * time.Minute)
		}
	}()

	<-terminate
	log.Printf("PlaneBoard: signal received, stopping")
}
