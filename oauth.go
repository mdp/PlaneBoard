package main

import (
	"fmt"

	"github.com/mrjones/oauth"
)

const (
	k_auth_url         = "https://api.twitter.com/oauth/authorize"
	k_token_url        = "https://api.twitter.com/oauth/request_token"
	k_access_token_url = "https://api.twitter.com/oauth/access_token"
)

type Oauth struct {
	Consumer *oauth.Consumer
}

type AuthenticationRequest struct {
	RequestToken *oauth.RequestToken
	Url          string
}

func NewOauth(ConsumerKey, ConsumerSecret string) Oauth {
	c := oauth.NewConsumer(
		ConsumerKey,
		ConsumerSecret,
		oauth.ServiceProvider{
			RequestTokenUrl:   k_token_url,
			AuthorizeTokenUrl: k_auth_url,
			AccessTokenUrl:    k_access_token_url,
		})

	return Oauth{c}
}

func (o Oauth) NewAuthenticationRequest() (*AuthenticationRequest, error) {
	requestToken, url, err := o.Consumer.GetRequestTokenAndUrl("oob")
	if err != nil {
		return nil, err
	}

	return &AuthenticationRequest{requestToken, url}, nil
}

func (o Oauth) GetAccessToken(RequestToken *oauth.RequestToken, code string) (*oauth.AccessToken, error) {
	accessToken, err := o.Consumer.AuthorizeToken(RequestToken, code)
	return accessToken, err
}

func AuthWithTwitter(consumerKey, consumerSecret string) {
	oauth := NewOauth(consumerKey, consumerSecret)
	ar, _ := oauth.NewAuthenticationRequest()
	fmt.Printf("In your browser, log in to your twitter account.  Then visit: \n%s\n", ar.Url)
	fmt.Println("After logged in, you will be promoted with a pin number")
	fmt.Println("Enter the pin number here:")
	pinCode := ""
	fmt.Scanln(&pinCode)
	accessToken, err := oauth.GetAccessToken(ar.RequestToken, pinCode)
	if err != nil {
		fmt.Printf("Error getting your access token: %s\n", err)
		return
	}
	fmt.Println("Here's your access token and secret. Update config.toml with these keys")
	fmt.Printf("TokenKey= \"%s\"\nTokenSecret: \"%s\"\n", accessToken.Token, accessToken.Secret)
}
