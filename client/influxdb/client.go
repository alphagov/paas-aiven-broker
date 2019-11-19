package influxdb

import (
	"fmt"
	"net/http"
)

const (
	InfluxDBBuildHeader   = "X-Influxdb-Build"
	InfluxDBVersionHeader = "X-Influxdb-Version"
)

type Client struct {
	http *http.Client
	URI  string
}

type influxdbPingResponse struct {
	Build   string
	Version string
}

func New(uri string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{http: httpClient, URI: uri}
}

func (c *Client) Ping() (influxdbPingResponse, error) {
	url := fmt.Sprintf("%s/ping", c.URI)

	resp, err := c.http.Head(url)
	if err != nil {
		return influxdbPingResponse{}, err
	}

	if resp.StatusCode != 204 {
		return influxdbPingResponse{}, fmt.Errorf(
			"Expected HTTP 204, received HTTP %d", resp.StatusCode,
		)
	}

	return influxdbPingResponse{
		Build:   resp.Header.Get(InfluxDBBuildHeader),
		Version: resp.Header.Get(InfluxDBVersionHeader),
	}, nil
}

func (c *Client) Version() (string, error) {
	resp, err := c.Ping()
	if err != nil {
		return "", err
	}

	if resp.Version == "" {
		return "", fmt.Errorf(
			"Error getting version: %s header was empty",
			InfluxDBVersionHeader,
		)
	}

	return resp.Version, nil
}

func (c *Client) Build() (string, error) {
	resp, err := c.Ping()
	if err != nil {
		return "", err
	}

	if resp.Build == "" {
		return "", fmt.Errorf(
			"Error getting build: %s header was empty",
			InfluxDBBuildHeader,
		)
	}

	return resp.Build, nil
}
