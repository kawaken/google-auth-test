package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

const (
	AUTH_URL      = "https://accounts.google.com/o/oauth2/device/code"
	CLIENT_ID     = ""
	CLIENT_SECRET = ""
	POLLING_URL   = "https://www.googleapis.com/oauth2/v3/token"
	SCOPE         = "email profile"
	GRANT_TYPE    = "http://oauth.net/grant_type/device/1.0"
)

type DeviceToken struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationUrl string `json:"verification_url"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int
}

type AuthToken struct {
	AccessToken      string `json:"access_token"`
	TokenType        string `json:"token_type"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshToken     string `json:"refresh_token"`
	IdToken          string `json:"id_token"`
	Error            string
	ErrorDescription string `json:"error_description"`
}

type Pooling struct {
	url     string
	result  chan string
	closing chan bool
	err     error
}

func (p *Pooling) Close() error {
	close(p.closing)
	return p.err
}

func main() {
	values := url.Values{}
	values.Add("client_id", CLIENT_ID)
	values.Add("scope", SCOPE)

	resp, err := http.PostForm(AUTH_URL, values)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	var dt DeviceToken
	err = json.Unmarshal(b, &dt)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Access the following url:", dt.VerificationUrl)
	fmt.Println("And enter the code:", dt.UserCode)

	for i := 0; i < 10; i++ {
		time.Sleep(time.Duration(dt.Interval) * time.Second)

		values = url.Values{}
		values.Add("client_id", CLIENT_ID)
		values.Add("client_secret", CLIENT_SECRET)
		values.Add("code", dt.DeviceCode)
		values.Add("grant_type", GRANT_TYPE)
		resp, err = http.PostForm(POLLING_URL, values)

		if err != nil {
			fmt.Println(err)
			return
		}

		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
			return
		}

		var at AuthToken
		err = json.Unmarshal(b, &at)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println(at)

		if at.Error == "" {
			fmt.Println("Verified")
			break
		}

		fmt.Printf("%s: %s", at.Error, at.ErrorDescription)
	}

}
