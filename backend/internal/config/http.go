package config

import (
	"fmt"
	"net/http"
	"time"
)

type Transport struct {
	Base    http.RoundTripper
	Token   string
	Headers map[string]string
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	reqBodyCopy := req.Clone(req.Context())
	reqBodyCopy.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.Token))
	for key, value := range t.Headers {
		reqBodyCopy.Header.Set(key, value)
	}
	return t.Base.RoundTrip(reqBodyCopy)
}

func NewGitHubClient(token string) *http.Client {
	return &http.Client{
		Timeout: time.Second * 30,
		Transport: &Transport{
			Base:  http.DefaultTransport,
			Token: token,
			Headers: map[string]string{
				"Accept":               "application/vnd.github+json",
				"X-GitHub-Api-Version": "2022-11-28",
			},
		},
	}
}
