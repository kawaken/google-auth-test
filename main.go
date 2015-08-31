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

type Tokener interface {
	Token() string
}

type DeviceToken struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationUrl string `json:"verification_url"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int
}

func (d *DeviceToken) Token() string {
	return d.DeviceCode
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

func (a *AuthToken) Token() string {
	return a.AccessToken
}

func loadConfig() (*Config, error) {
	conf := new(Config)
	_, err := toml.DecodeFile("conf.toml", conf)
	return conf, err
}

func (conf *Config) save() (err error) {

	var buffer bytes.Buffer
	encoder := toml.NewEncoder(&buffer)
	err = encoder.Encode(conf)
	if err != nil {
		return
	}
	err = ioutil.WriteFile("conf.toml", buffer.Bytes(), os.ModePerm)
	return
}

func requestToken(url string, values url.Values, obj interface{}) error {
	resp, err := http.PostForm(url, values)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(b, obj)
	if err != nil {
		return err
	}

	return nil
}

func (conf *Config) refresh(at *AuthToken) error {
	if at.Error != "" {
		return fmt.Errorf("%s: %s", at.Error, at.ErrorDescription)
	}

	if at.AccessToken == "" {
		return fmt.Errorf("Token is empty")
	}

	conf.AccessToken = at.AccessToken
	// RefreshTokenを更新する必要がない場合は空文字になるので、上書きしない
	if at.RefreshToken != "" {
		conf.RefreshToken = at.RefreshToken
	}
	conf.ExpiredAt = time.Now().Add(time.Duration(at.ExpiresIn) * time.Second)

	err := conf.save()
	return err
}

func initAccessToken(conf *Config) {
	fmt.Println("Init AccessToken")

	values := url.Values{}
	values.Add("client_id", conf.CLIENT_ID)
	values.Add("scope", SCOPE)

	dt := &DeviceToken{}
	err := requestToken(AUTH_URL, values, dt)
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

		at := &AuthToken{}
		err = requestToken(POLLING_URL, values, at)
		if err != nil {
			fmt.Println(err)
			return
		}

		err := conf.refresh(at)
		if err == nil {
			fmt.Println("Verified")
			return
		}
	}

	fmt.Println("Unauthorized. Retry")

}

func refreshAccessToken(conf *Config) {
	fmt.Println("Refresh AccessToken")

	values := url.Values{}
	values.Add("client_id", conf.CLIENT_ID)
	values.Add("client_secret", conf.CLIENT_SECRET)
	values.Add("refresh_token", conf.RefreshToken)
	values.Add("grant_type", "refresh_token")

	at := &AuthToken{}
	err := requestToken(POLLING_URL, values, at)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = conf.refresh(at)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("Verified")
	}
	return
}

func main() {

	conf, err := loadConfig()
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
