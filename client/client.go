package client

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
)

type Client struct {
	client *http.Client // http.client
}

func New(timeout time.Duration) *Client {
	return &Client{
		client: &http.Client{
			Transport: Transport(),
			Timeout:   timeout,
		},
	}
}

func (c *Client) Read(URL string) ([]byte, error) {

	if resp, err := c.client.Get(URL); err != nil {
		return []byte{}, err
	} else {
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return []byte{}, errors.Wrap(err, "Read Response")
		}
		return body, nil
	}
}

func (c *Client) Proxy(u url.URL) *Client {
	transport := c.client.Transport.(*http.Transport)
	transport.Proxy = http.ProxyURL(&u)
	c.client.Transport = transport
	c.client.Timeout = time.Duration(10 * time.Second)
	return c
}

func Transport() *http.Transport {
	var transport = http.Transport{
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
		DisableCompression:    false,
		ResponseHeaderTimeout: 2 * time.Second,
	}

	return &transport
}
