package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/dghubble/go-twitter/twitter"
)

// TwitterTimeLayout is the format Twitter returns for CreatedAt
var TwitterTimeLayout = "Mon Jan 02 15:04:05 -0700 2006"

// Key lets us quickly iterater through tweets and get only
// the ones we care about without unmarshalling json
// createdAt is critical to deleting older tweets to
// keep the database manageable
type Key struct {
	ID         int64
	ScreenName string
	CreatedAt  int64
}

// Serialize a Key to a string
func (k *Key) Serialize() (string, error) {
	return fmt.Sprintf("%d:%s:%d", k.CreatedAt, k.ScreenName, k.ID), nil
}

// Deserialize a Key from a string
func (k *Key) Deserialize(str string) error {
	keys := strings.Split(str, ":")
	id, err := strconv.ParseInt(keys[2], 10, 64)
	if err != nil {
		return err
	}
	timestamp, err := strconv.ParseInt(keys[0], 10, 64)
	if err != nil {
		return err
	}
	k.ID = id
	k.ScreenName = keys[1]
	k.CreatedAt = timestamp
	return nil
}

// SetupDataStore is fucking obvious
func SetupDataStore(dbFile string) (*DataStore, error) {
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("Tweets"))
		return err
	})
	return &DataStore{
		DB: db,
	}, nil
}

// DataStore holds our database instance and config
type DataStore struct {
	DB *bolt.DB
}

// AddTweet to the database
func (d *DataStore) AddTweet(tweet *twitter.Tweet) {
	tweetJSON, err := json.Marshal(tweet)
	if err != nil {
		log.Println(err)
		return
	}
	createdAt, err := time.Parse(TwitterTimeLayout, tweet.CreatedAt)
	key := &Key{
		ID:         tweet.ID,
		ScreenName: strings.ToLower(tweet.User.ScreenName),
		CreatedAt:  createdAt.Unix(),
	}
	keyStr, err := key.Serialize()
	if err != nil {
		log.Println(err)
		return
	}
	d.DB.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("Tweets"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		err = b.Put([]byte(keyStr), []byte(tweetJSON))
		return err
	})

}

// FindTweet to find the most recent tweets for a series of accounts
func (d *DataStore) FindTweet(screenNames []string, before int64, page int) *twitter.Tweet {
	if before == 0 {
		before = time.Now().Unix()
	}
	var matchingRecords = [][]byte{}
	d.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Tweets"))

		b.ForEach(func(k, v []byte) error {
			key := &Key{}
			key.Deserialize(string(k))
			if len(screenNames) == 1 && screenNames[0] == "home" {
				if key.CreatedAt < before {
					matchingRecords = append(matchingRecords, k)
				}
			} else {
				for _, sn := range screenNames {
					if sn == key.ScreenName && key.CreatedAt < before {
						matchingRecords = append(matchingRecords, k)
						break
					}
				}
			}
			return nil
		})
		return nil
	})
	if len(matchingRecords)-1 < page {
		return nil
	}
	record := matchingRecords[len(matchingRecords)-page-1]
	tweet := &twitter.Tweet{}
	d.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Tweets"))
		tweetJSON := b.Get(record)
		json.Unmarshal(tweetJSON, tweet)
		return nil
	})
	return tweet
}

// Clean gets rid of old tweets we no longer care about
func (d *DataStore) Clean(before int64) {
	log.Printf("DataStore Clean - Deleting keys older than %d\n", before)
	var matchingRecords = [][]byte{}
	d.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Tweets"))
		b.ForEach(func(k, v []byte) error {
			key := &Key{}
			key.Deserialize(string(k))
			if key.CreatedAt < before {
				matchingRecords = append(matchingRecords, k)
			}
			return nil
		})
		return nil
	})
	d.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Tweets"))
		for _, k := range matchingRecords {
			fmt.Printf("Deleting Old Tweet %s\n", k)
			b.Delete(k)
		}
		return nil
	})
}

// Close the database
func (d *DataStore) Close() {
	d.DB.Close()
}
