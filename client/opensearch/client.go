package opensearch

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	http *http.Client
	URI  string
}

type opensearchResponseVersion struct {
	Number string `json:"number,omitempty"`
}

type opensearchResponse struct {
	Error   string                    `json:"error,omitempty"`
	Version opensearchResponseVersion `json:"version,omitempty"`
}

func New(uri string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{http: httpClient, URI: uri}
}

func (c *Client) Ping() (*opensearchResponse, error) {
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

func (c *Client) readBody(body io.Reader) (*opensearchResponse, error) {
	data := opensearchResponse{}
	err := json.NewDecoder(body).Decode(&data)
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
