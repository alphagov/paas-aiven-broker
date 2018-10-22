package aiven

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

//go:generate counterfeiter -o fakes/fake_client.go . Client
type Client interface {
	CreateService(params *CreateServiceInput) (string, error)
	GetServiceStatus(params *GetServiceInput) (ServiceStatus, time.Time, error)
	GetServiceConnectionDetails(params *GetServiceInput) (string, string, error)
	DeleteService(params *DeleteServiceInput) error
	CreateServiceUser(params *CreateServiceUserInput) (string, error)
	DeleteServiceUser(params *DeleteServiceUserInput) (string, error)
	UpdateService(params *UpdateServiceInput) (string, error)
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

type ErrInvalidUpdate struct {
	Message string
}

func (p ErrInvalidUpdate) Error() string {
	return p.Message
}

type CreateServiceInput struct {
	Cloud       string     `json:"cloud,omitempty"`
	GroupName   string     `json:"group_name,omitempty"`
	Plan        string     `json:"plan,omitempty"`
	ServiceName string     `json:"service_name"`
	ServiceType string     `json:"service_type"`
	UserConfig  UserConfig `json:"user_config"`
}

type UserConfig struct {
	ElasticsearchVersion string   `json:"elasticsearch_version"`
	IPFilter             []string `json:"ip_filter,omitempty"`
}

type DeleteServiceInput struct {
	ServiceName string
}

type CreateServiceUserInput struct {
	ServiceName string `json:"-"`
	Username    string `json:"username"`
}

type CreateServiceUserResponse struct {
	Message string `json:"message"`
	User    User   `json:"user"`
}

type User struct {
	Password string `json:"password"`
	Type     string `json:"type"`
	Username string `json:"username"`
}

type DeleteServiceUserInput struct {
	ServiceName string
	Username    string
}

type GetServiceInput struct {
	ServiceName string
}

type GetServiceResponse struct {
	Service Service `json:"service"`
}

type Service struct {
	State            ServiceStatus    `json:"state"`
	UpdateTime       time.Time        `json:"update_time"`
	ServiceUriParams ServiceUriParams `json:"service_uri_params"`
}

type ServiceStatus string

const (
	Running     ServiceStatus = "RUNNING"
	Rebuilding  ServiceStatus = "REBUILDING"
	Rebalancing ServiceStatus = "REBALANCING"
	PowerOff    ServiceStatus = "POWEROFF"
)

type ServiceUriParams struct {
	Host     string `json:"host"`
	Password string `json:"password"`
	Port     string `json:"port"`
	User     string `json:"user"`
}

type UpdateServiceInput struct {
	ServiceName string     `json:"-"`
	Plan        string     `json:"plan,omitempty"`
	UserConfig  UserConfig `json:"user_config"`
}

type AivenErrorResponse struct {
	Errors []struct {
		Message string `json:"message"`
		Status  int    `json:"status"`
	} `json:"errors"`
	Message string `json:"message"`
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
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Error creating service: %d status code returned from Aiven", res.StatusCode)
	}

	b, _ := ioutil.ReadAll(res.Body)
	return string(b), nil
}

func (a *HttpClient) GetServiceStatus(params *GetServiceInput) (ServiceStatus, time.Time, error) {
	getServiceResponse, err := a.getService(params)
	if err != nil {
		return "", time.Time{}, err
	}

	service := getServiceResponse.Service

	if service.State == "" {
		return "", time.Time{}, errors.New("Error getting service status: no state found in response JSON")
	}

	defaultTime := time.Time{}
	if service.UpdateTime == defaultTime {
		return "", time.Time{}, errors.New("Error getting service status: no update_time found in response JSON")
	}

	return service.State, service.UpdateTime, nil
}

func (a *HttpClient) GetServiceConnectionDetails(params *GetServiceInput) (string, string, error) {
	getServiceResponse, err := a.getService(params)
	if err != nil {
		return "", "", err
	}

	uriParams := getServiceResponse.Service.ServiceUriParams
	host := uriParams.Host
	port := uriParams.Port
	if host == "" || port == "" {
		return "", "", errors.New("Error getting service connection details: no connection details found in response JSON")
	}
	return host, port, nil
}

var ErrInstanceDoesNotExist = errors.New("Error deleting service: service instance does not exist")

func (a *HttpClient) DeleteService(params *DeleteServiceInput) error {
	res, err := a.do("DELETE", fmt.Sprintf("/project/%s/service/%s", a.Project, params.ServiceName), nil)
	if err != nil {
		return err
	}

	if res.StatusCode == http.StatusOK {
		return nil
	}

	if res.StatusCode == http.StatusNotFound {
		return ErrInstanceDoesNotExist
	}

	return fmt.Errorf("Error deleting service: %d status code returned from Aiven", res.StatusCode)
}

func (a *HttpClient) CreateServiceUser(params *CreateServiceUserInput) (string, error) {
	reqBody, err := json.Marshal(params)
	if err != nil {
		return "", err
	}

	res, err := a.do("POST", fmt.Sprintf("/project/%s/service/%s/user", a.Project, params.ServiceName), reqBody)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Error creating service user: %d status code returned from Aiven", res.StatusCode)
	}

	b, _ := ioutil.ReadAll(res.Body)
	createServiceUserResponse := &CreateServiceUserResponse{}
	if err := json.Unmarshal(b, createServiceUserResponse); err != nil {
		return "", err
	}

	if createServiceUserResponse.User.Password == "" {
		return "", errors.New("Error creating service user: password was empty")
	}
	return createServiceUserResponse.User.Password, nil
}

func (a *HttpClient) DeleteServiceUser(params *DeleteServiceUserInput) (string, error) {
	res, err := a.do("DELETE", fmt.Sprintf("/project/%s/service/%s/user/%s", a.Project, params.ServiceName, params.Username), nil)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Error deleting service user: %d status code returned from Aiven", res.StatusCode)
	}

	b, _ := ioutil.ReadAll(res.Body)
	return string(b), nil
}

func (a *HttpClient) getService(params *GetServiceInput) (*GetServiceResponse, error) {
	res, err := a.do("GET", fmt.Sprintf("/project/%s/service/%s", a.Project, params.ServiceName), nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Error getting service: %d status code returned from Aiven", res.StatusCode)
	}

	b, _ := ioutil.ReadAll(res.Body)
	getServiceResponse := &GetServiceResponse{}
	if err := json.Unmarshal(b, getServiceResponse); err != nil {
		return nil, err
	}
	return getServiceResponse, nil
}

func (a *HttpClient) UpdateService(params *UpdateServiceInput) (string, error) {
	reqBody, err := json.Marshal(params)
	if err != nil {
		return "", err
	}

	res, err := a.do("PUT", fmt.Sprintf("/project/%s/service/%s", a.Project, params.ServiceName), reqBody)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	b, _ := ioutil.ReadAll(res.Body)

	if res.StatusCode == http.StatusBadRequest {
		var errorResponse AivenErrorResponse
		jsonErr := json.Unmarshal(b, &errorResponse)
		if jsonErr != nil {
			return "", fmt.Errorf("Error updating service: %d status code returned from Aiven: '%s'", res.StatusCode, b)
		}
		return "", ErrInvalidUpdate{fmt.Sprintf("Invalid Update: %s", errorResponse.Message)}
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Error updating service: %d status code returned from Aiven: '%s'", res.StatusCode, b)
	}

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
