package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/BurntSushi/toml"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"
)

const (
	AUTH_URL          = "https://accounts.google.com/o/oauth2/device/code"
	POLLING_URL       = "https://www.googleapis.com/oauth2/v3/token"
	SCOPE             = "email profile"
	GRANT_TYPE        = "http://oauth.net/grant_type/device/1.0"
	POLLING_RETRY_MAX = 60
)

type Config struct {
	CLIENT_ID     string
	CLIENT_SECRET string
	AccessToken   string
	RefreshToken  string
	ExpiredAt     time.Time
}

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

func loadConf() (conf *Config, err error) {
	_, err = toml.DecodeFile("conf.toml", &conf)
	return
}

func writeConf(conf *Config) (err error) {

	var buffer bytes.Buffer
	encoder := toml.NewEncoder(&buffer)
	err = encoder.Encode(conf)
	if err != nil {
		return
	}
	err = ioutil.WriteFile("conf.toml", buffer.Bytes(), os.ModePerm)
	return
}

func initAccessToken(conf *Config) {

	values := url.Values{}
	values.Add("client_id", conf.CLIENT_ID)
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

	for i := 0; i < POLLING_RETRY_MAX; i++ {
		fmt.Printf("Retry: %d/%d\r", i, POLLING_RETRY_MAX)

		time.Sleep(time.Duration(dt.Interval) * time.Second)

		values = url.Values{}
		values.Add("client_id", conf.CLIENT_ID)
		values.Add("client_secret", conf.CLIENT_SECRET)
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

		if at.Error == "" && at.AccessToken != "" {
			fmt.Println("\nVerified")
			conf.AccessToken = at.AccessToken
			conf.RefreshToken = at.RefreshToken
			conf.ExpiredAt = time.Now().Add(time.Duration(at.ExpiresIn) * time.Second)

			err := writeConf(conf)
			if err != nil {
				fmt.Println(err)
			}
			return
		}

	}

	fmt.Println("Unauthorized. Retry")

}

func refreshAccessToken(conf *Config) {
	fmt.Println("refresh!!!!")

	values := url.Values{}
	values.Add("client_id", conf.CLIENT_ID)
	values.Add("client_secret", conf.CLIENT_SECRET)
	values.Add("refresh_token", conf.RefreshToken)
	values.Add("grant_type", "refresh_token")

	resp, err := http.PostForm(POLLING_URL, values)

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

	if at.Error == "" && at.AccessToken != "" {
		fmt.Println("\nVerified")
		conf.AccessToken = at.AccessToken
		if at.RefreshToken != "" {
			conf.RefreshToken = at.RefreshToken
		}
		conf.ExpiredAt = time.Now().Add(time.Duration(at.ExpiresIn) * time.Second)

		err := writeConf(conf)
		if err != nil {
			fmt.Println(err)
		}
		return
	}
}

func main() {

	conf, err := loadConf()
	if err != nil {
		fmt.Println(err)
		return
	}

	if conf.AccessToken == "" {
		initAccessToken(conf)
		return
	}

	if conf.ExpiredAt.Before(time.Now()) {
		refreshAccessToken(conf)
		return
	}

	fmt.Println("Nothing to be done")

}
