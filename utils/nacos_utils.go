package utils

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/go-resty/resty/v2"
)

const LOGIN_PATH = "/nacos/v1/auth/login"

var commonHeader = map[string]string{
	"namespace": "nacos",
	"service":   "nacos-client",
	"port":      "8848",
}

type nacosRoundTripWrapper struct {
	Proxy             func(*http.Request) (*url.URL, error)
	InnerRoundWrapper http.RoundTripper
}

func (rw nacosRoundTripWrapper) RoundTrip(req *http.Request) (*http.Response, error) {
	wrapperdReq, err := rw.Proxy(req)
	if err != nil {
		return nil, err
	}
	req.URL = wrapperdReq
	for k, v := range commonHeader {
		req.Header.Add(k, v)
	}
	return rw.InnerRoundWrapper.RoundTrip(req)
}

type NacosRoundTripper struct {
	addr             string
	username         string
	password         string
	baseRoundTripper http.RoundTripper
	authInfo         *AuthInfo
	lastLogin        time.Time
}

func NewNacosRoundTripper(addr, username, password string, rt http.RoundTripper) *NacosRoundTripper {
	baseRoundTripper := rt
	if baseRoundTripper == nil {
		baseRoundTripper = http.DefaultTransport
	}
	return &NacosRoundTripper{
		addr:             addr,
		username:         username,
		password:         password,
		baseRoundTripper: baseRoundTripper,
	}

}

func (p *NacosRoundTripper) getToken() (string, error) {
	if p.authInfo == nil {
		if err := p.doAuth(); err != nil {
			return "", err
		}
		return p.authInfo.AccessToken, nil
	}
	if time.Now().Unix()-p.lastLogin.Unix() >= p.authInfo.TokenTTL-300 {
		if err := p.doAuth(); err != nil {
			return "", err
		}
	}
	return p.authInfo.AccessToken, nil
}

type AuthInfo struct {
	AccessToken string `json:"accessToken"`
	TokenTTL    int64  `json:"tokenTtl"`
	GlobalAdmin bool   `json:"globalAdmin"`
}

func (p *NacosRoundTripper) doAuth() error {
	bodyData := map[string]string{
		"username": p.username,
		"password": p.password,
	}
	authResponse := &AuthInfo{}
	cli := resty.New()
	if p.baseRoundTripper != nil {
		cli.SetTransport(p.baseRoundTripper)
	} else {
		p.baseRoundTripper = cli.GetClient().Transport
	}
	resp, err := cli.R().
		SetHeaders(commonHeader).
		SetFormData(bodyData).
		SetResult(authResponse).
		Post(p.addr + LOGIN_PATH)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("failed to login nacos, code is %d", resp.StatusCode())
	}
	p.authInfo = authResponse
	p.lastLogin = time.Now()
	return nil
}

// GetRoundTripper 自动添加认证信息和额外的headers信息
func (p *NacosRoundTripper) GetRoundTripper() http.RoundTripper {
	fn := func(r *http.Request) (*url.URL, error) {
		token, err := p.getToken()
		if err != nil {
			return r.URL, err
		}
		q := r.URL.Query()
		q.Add("accessToken", token)
		r.URL.RawQuery = q.Encode()
		return r.URL, nil
	}
	return nacosRoundTripWrapper{
		Proxy:             fn,
		InnerRoundWrapper: p.baseRoundTripper,
	}
}
