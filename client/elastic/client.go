package elastic

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

type Client struct {
	http *http.Client
	URI  string
}

type elasticsearchResponseVersion struct {
	Number string `json:"number,omitempty"`
}

type elasticsearchResponse struct {
	Error   string                       `json:"error,omitempty"`
	Version elasticsearchResponseVersion `json:"version,omitempty"`
}

func New(uri string, httpClient *http.Client) (*Client, error) {
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	return &Client{http: httpClient, URI: uri}, nil
}

func (c *Client) Ping() (*elasticsearchResponse, error) {
	resp, err := c.http.Get(c.URI)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	r, err := c.readBody(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 300 || resp.StatusCode < 200 {
		return nil, fmt.Errorf("request error: %s", r.Error)
	}

	return r, nil
}

func (c *Client) readBody(body io.ReadCloser) (*elasticsearchResponse, error) {
	bodyBytes, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, err
	}

	data := elasticsearchResponse{}
	err = json.Unmarshal(bodyBytes, &data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (c *Client) Version() (string, error) {
	resp, err := c.Ping()
	if err != nil {
		return "", err
	}

	if resp.Version.Number == "" {
		return "", fmt.Errorf("version number: empty")
	}

	return resp.Version.Number, nil
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.http.Do(req)
}

func (c *Client) Get(uri string) (*http.Response, error) {
	return c.http.Get(uri)
}
