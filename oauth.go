package main

import (
	"fmt"

	"github.com/mrjones/oauth"
)

const (
	twAuthURL        = "https://api.twitter.com/oauth/authorize"
	twTokenURL       = "https://api.twitter.com/oauth/request_token"
	twAccessTokenURL = "https://api.twitter.com/oauth/access_token"
)

// Oauth struct for holding a consumer
type Oauth struct {
	Consumer *oauth.Consumer
}

// AuthenticationRequest holds our request for later confirmation
type AuthenticationRequest struct {
	RequestToken *oauth.RequestToken
	URL          string
}

func newOauth(ConsumerKey, ConsumerSecret string) Oauth {
	c := oauth.NewConsumer(
		ConsumerKey,
		ConsumerSecret,
		oauth.ServiceProvider{
			RequestTokenUrl:   twTokenURL,
			AuthorizeTokenUrl: twAuthURL,
			AccessTokenUrl:    twAccessTokenURL,
		})

	return Oauth{c}
}

func (o Oauth) newAuthenticationRequest() (*AuthenticationRequest, error) {
	requestToken, url, err := o.Consumer.GetRequestTokenAndUrl("oob")
	if err != nil {
		return nil, err
	}

	return &AuthenticationRequest{requestToken, url}, nil
}

func (o Oauth) getAccessToken(RequestToken *oauth.RequestToken, code string) (*oauth.AccessToken, error) {
	accessToken, err := o.Consumer.AuthorizeToken(RequestToken, code)
	return accessToken, err
}

// AuthWithTwitter call with the consumer key and secret to start
// an OOB/PIN authentication with Twitter
func AuthWithTwitter(consumerKey, consumerSecret string) {
	oauth := newOauth(consumerKey, consumerSecret)
	ar, _ := oauth.newAuthenticationRequest()
	fmt.Printf("In your browser, log in to your twitter account.  Then visit: \n%s\n", ar.URL)
	fmt.Println("After logged in, you will be promoted with a pin number")
	fmt.Println("Enter the pin number here:")

	pinCode := ""
	fmt.Scanln(&pinCode)

	accessToken, err := oauth.getAccessToken(ar.RequestToken, pinCode)
	if err != nil {
		fmt.Printf("Error getting your access token: %s\n", err)
		return
	}

	fmt.Println("Success! The following are your access token and secret. Update config.toml with these keys")
	fmt.Printf("TokenKey = \"%s\"\nTokenSecret = \"%s\"\n", accessToken.Token, accessToken.Secret)
}
