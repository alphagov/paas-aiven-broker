package aiven

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

//go:generate counterfeiter -o fakes/fake_client.go . Client
type Client interface {
	CreateService(params *CreateServiceInput) (string, error)
	GetServiceStatus(params *GetServiceStatusInput) (ServiceStatus, error)
	DeleteService(params *DeleteServiceInput) (string, error)
}

type HttpClient struct {
	BaseURL    string
	Token      string
	Project    string
	HTTPClient *http.Client
}

func NewHttpClient(baseURL, token, project string) *HttpClient {
	return &HttpClient{
		BaseURL:    baseURL,
		Token:      token,
		Project:    project,
		HTTPClient: &http.Client{},
	}
}

type UserConfig struct {
	ElasticsearchVersion string `json:"elasticsearch_version"`
}

type CreateServiceInput struct {
	Cloud       string     `json:"cloud,omitempty"`
	GroupName   string     `json:"group_name,omitempty"`
	Plan        string     `json:"plan,omitempty"`
	ServiceName string     `json:"service_name"`
	ServiceType string     `json:"service_type"`
	UserConfig  UserConfig `json:"user_config"`
}

type GetServiceStatusResponse struct {
	Service Service `json:"service"`
}

type Service struct {
	State ServiceStatus `json:"state"`
}

type ServiceStatus string

const (
	Running     ServiceStatus = "RUNNING"
	Rebuilding  ServiceStatus = "REBUILDING"
	Rebalancing ServiceStatus = "REBALANCING"
	PowerOff    ServiceStatus = "POWEROFF"
)

type GetServiceStatusInput struct {
	ServiceName string
}

type DeleteServiceInput struct {
	ServiceName string
}

func (a *HttpClient) CreateService(params *CreateServiceInput) (string, error) {
	reqBody, err := json.Marshal(params)
	if err != nil {
		return "", err
	}

	res, err := a.do("POST", fmt.Sprintf("/project/%s/service", a.Project), reqBody)
	if err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Error creating service: %d status code returned from Aiven", res.StatusCode)
	}

	b, _ := ioutil.ReadAll(res.Body)
	return string(b), nil
}

func (a *HttpClient) GetServiceStatus(params *GetServiceStatusInput) (ServiceStatus, error) {
	res, err := a.do("GET", fmt.Sprintf("/project/%s/service/%s", a.Project, params.ServiceName), nil)
	if err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Error getting service status: %d status code returned from Aiven", res.StatusCode)
	}

	b, _ := ioutil.ReadAll(res.Body)
	getServiceStatusResponse := &GetServiceStatusResponse{}
	if err := json.Unmarshal(b, getServiceStatusResponse); err != nil {
		return "", err
	}

	if getServiceStatusResponse.Service.State == "" {
		return "", errors.New("Error getting service status: no state found in response JSON")
	}
	return getServiceStatusResponse.Service.State, nil
}

func (a *HttpClient) DeleteService(params *DeleteServiceInput) (string, error) {
	res, err := a.do("DELETE", fmt.Sprintf("/project/%s/service/%s", a.Project, params.ServiceName), nil)
	if err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Error deleting service: %d status code returned from Aiven", res.StatusCode)
	}

	b, _ := ioutil.ReadAll(res.Body)
	return string(b), nil
}

func (a *HttpClient) do(method, path string, body []byte) (*http.Response, error) {
	req, err := a.requestBuilder(method, path, body)
	if err != nil {
		return nil, err
	}

	res, err := a.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (a *HttpClient) requestBuilder(method, path string, body []byte) (*http.Request, error) {
	req, err := http.NewRequest(method, fmt.Sprintf("%s/v1beta%s", a.BaseURL, path), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("aivenv1 %s", a.Token))

	return req, err
}
