# PlaneBoard
#### Read your tweets, even behind captive WiFi portals, using DNS TXT records

![aeropuerto](https://cloud.githubusercontent.com/assets/2868/15951028/737d6ea8-2e69-11e6-8eda-a9a82d57a0ee.png)

## Quick Demo

From your command line, lets fire up 'dig' and try it out!

`dig txt p0.t.news.pb.mdp.im`

## Why

This project has no serious application, it's merely a fun experiment that I prototyped while waiting on a delayed flight. There are numerous methods to tunnel traffic through DNS queries, iodine being the most popular. This is just a simple demonstration of how it's possible to get up to date, human readable information from a DNS query.

## How

Nearly all captive portals will still proxy outbound DNS requests. We can use this proxy of DNS requests to 'leak' information we might be interested in. In this case I'm returning my Twitter stream as TXT DNS records.

## Installation

    git clone https://github.com/mdp/planeboard
    cd planeboard
    go get -u
    go build
    cp config.toml.sample config.toml
    vi config.toml # Update with relevant information

What you'll need to use this:

1. An NS record pointing at the host you're running this on
2. A [Twitter OAuth application](https://apps.twitter.com/) and it's consumer keys
3. A Twitter account to authenticate with the Twitter OAuth application

#### NS record example

    pb		    IN	NS	server.myhost.com.
    server		IN	A	  192.168.1.1

#### Authentication with Twitter

Once you got config.toml setup with your Twitter keys, just run `./planeboard auth` and follow the instructions. You'll get back a set of access keys tied to the Twitter account you want to read from.

#### Running planeboard

    sudo ./planeboard

## Usage

Lets say I'm using pb.mdp.im as my host

    # The following are example 'dig' commands you would enter on your command line

    # Paginate with 'p'
    dig txt p0.pb.mdp.im
    # Gives me the first tweet
    dig txt p1.pb.mdp.im
    # Gives me the second tweet

    # Toss in a 'b'(before) with a unix timestamp to help with proper pagination
    dig txt b1465514642.p1.pb.mdp.im

    # Toss in a 'c'(cachebuster) to prevent caching
    dig txt c8y7tnpynb0.b1465514642.p0.pb.mdp.im

    # Finally, you can have topics setup in config.toml to help you filter
    # tweets into relevant groups. For example, lets say we have a 'news' topic
    # which consists of @cnn, @ap and @nytimes. We just need to add a 't' flag
    # and group name to the request
    dig txt c8y7tnpynb0.b1465514642.p0.t.news.pb.mdp.im

### Using the bash script 'fetch.sh'

This is all automated in [a bash script](https://github.com/mdp/PlaneBoard/blob/master/fetch.sh) to help with fetching a large number of tweets in a timeline

    ./fetch.sh -t -n 20 -h pb.mdp.im news
    # grabs 20 most recent tweets from the news topic


## The nitty gritty details

How it works in a nutshell:

- All built in Go
- Tweets are gathered by consuming the Twitter streaming API
- Tweets are stored in a BoltDB database
- Inbound queries are parsed and relevant tweets are returned from the database

## License

MIT of course. Do with it as you please.
