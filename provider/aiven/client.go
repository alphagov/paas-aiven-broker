package aiven

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"code.cloudfoundry.org/lager"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o fakes/fake_client.go . Client
type Client interface {
	CreateService(params *CreateServiceInput) (string, error)
	GetService(params *GetServiceInput) (*Service, error)
	GetServiceTags(params *GetServiceTagsInput) (*ServiceTags, error)
	DeleteService(params *DeleteServiceInput) error
	CreateServiceUser(params *CreateServiceUserInput) (string, error)
	DeleteServiceUser(params *DeleteServiceUserInput) (string, error)
	UpdateService(params *UpdateServiceInput) (string, error)
	UpdateServiceTags(params *UpdateServiceTagsInput) (string, error)
	ForkService(params *ForkServiceInput) (string, error)
}

type HttpClient struct {
	BaseURL    string
	Token      string
	Project    string
	logger     lager.Logger
	HTTPClient *http.Client
}

func NewHttpClient(baseURL, token, project string, logger lager.Logger) *HttpClient {
	return &HttpClient{
		BaseURL:    baseURL,
		Token:      token,
		Project:    project,
		logger:     logger,
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
	Cloud       string      `json:"cloud,omitempty"`
	GroupName   string      `json:"group_name,omitempty"`
	Plan        string      `json:"plan,omitempty"`
	ServiceName string      `json:"service_name"`
	ServiceType string      `json:"service_type"`
	UserConfig  UserConfig  `json:"user_config"`
	Tags        ServiceTags `json:"tags"`
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
	ServiceType      string           `json:"service_type"`
	Backups          []ServiceBackup  `json:"backups"`
	Plan             string           `json:"plan"`
}

type ServiceStatus string

const (
	Running     ServiceStatus = "RUNNING"
	Rebuilding  ServiceStatus = "REBUILDING"
	Rebalancing ServiceStatus = "REBALANCING"
	PowerOff    ServiceStatus = "POWEROFF"
	Missing     ServiceStatus = "MISSING"
)

type ServiceUriParams struct {
	Host     string `json:"host"`
	Password string `json:"password"`
	Port     string `json:"port"`
	User     string `json:"user"`
}

type ServiceBackup struct {
	Name string    `json:"backup_name"`
	Time time.Time `json:"backup_time"`
	Size int       `json:"data_size"`
}

type ServiceTags struct {
	DeployEnv          string    `json:"deploy_env"`
	ServiceID          string    `json:"service_id"`
	PlanID             string    `json:"plan_id"`
	OrganizationID     string    `json:"organization_id"`
	SpaceID            string    `json:"space_id"`
	BrokerName         string    `json:"broker_name"`
	RestoredFromBackup string    `json:"restored_from_backup"`
	OriginServiceID    string    `json:"restored_from_service"`
	RestoredFromTime   time.Time `json:"restored_from_time"`
}

type GetServiceTagsInput struct {
	ServiceName string
}

type GetServiceTagsResponse struct {
	Tags ServiceTags `json:""`
}
type UpdateServiceInput struct {
	ServiceName string     `json:"-"`
	Plan        string     `json:"plan,omitempty"`
	UserConfig  UserConfig `json:"user_config"`
}
type UpdateServiceTagsInput struct {
	ServiceName string      `json:"-"`
	Tags        ServiceTags `json:"tags"`
}

type ForkServiceInput struct {
	Cloud       string      `json:"cloud,omitempty"`
	GroupName   string      `json:"group_name,omitempty"`
	Plan        string      `json:"plan,omitempty"`
	ServiceName string      `json:"service_name"`
	ServiceType string      `json:"service_type"`
	UserConfig  UserConfig  `json:"user_config"`
	Tags        ServiceTags `json:"tags"`
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
	a.logger.Info("create-service-body", lager.Data{
		"reqBody": reqBody})

	res, err := a.do("POST", fmt.Sprintf("/project/%s/service", a.Project), reqBody)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Error creating service: %d status code returned from Aiven: '%s'", res.StatusCode, b)
	}

	return string(b), nil
}

func (a *HttpClient) ForkService(params *ForkServiceInput) (string, error) {
	reqBody, err := json.Marshal(params)
	if err != nil {
		return "", err
	}

	res, err := a.do("POST", fmt.Sprintf("/project/%s/service", a.Project), reqBody)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Error creating service: %d status code returned from Aiven: '%s'", res.StatusCode, b)
	}

	return string(b), nil
}

var ErrInstanceDoesNotExist = errors.New("Error: service instance does not exist")

func (a *HttpClient) DeleteService(params *DeleteServiceInput) error {
	res, err := a.do("DELETE", fmt.Sprintf("/project/%s/service/%s", a.Project, params.ServiceName), nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		return nil
	}

	if res.StatusCode == http.StatusNotFound {
		return ErrInstanceDoesNotExist
	}

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	return fmt.Errorf("Error deleting service: %d status code returned from Aiven: '%s'", res.StatusCode, b)
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
		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return "", err
		}
		return "", fmt.Errorf("Error creating service user: %d status code returned from Aiven: '%s'", res.StatusCode, b)
	}

	createServiceUserResponse := &CreateServiceUserResponse{}
	if err := json.NewDecoder(res.Body).Decode(createServiceUserResponse); err != nil {
		return "", err
	}

	if createServiceUserResponse.User.Password == "" {
		return "", errors.New("Error creating service user: password was empty")
	}
	return createServiceUserResponse.User.Password, nil
}

var ErrInstanceUserDoesNotExist = errors.New("Error: service instance user does not exist")

func (a *HttpClient) DeleteServiceUser(params *DeleteServiceUserInput) (string, error) {
	res, err := a.do("DELETE", fmt.Sprintf("/project/%s/service/%s/user/%s", a.Project, params.ServiceName, params.Username), nil)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		var errorResponse AivenErrorResponse
		jsonErr := json.Unmarshal(b, &errorResponse)

		// if this response changes (again), just call it quits and sniff for 404.
		// the only reason we're not is in case we mistake "api not here at all"
		// for "service user doesn't exist".
		expectedMessageIfUserWasAlreadyDeleted := "Service user does not exist"
		if jsonErr == nil && errorResponse.Message == expectedMessageIfUserWasAlreadyDeleted {
			return "", ErrInstanceUserDoesNotExist
		}

		return "", fmt.Errorf("Error deleting service user: %d status code returned from Aiven: '%s'", res.StatusCode, b)
	}

	return string(b), nil
}

func (a *HttpClient) GetService(params *GetServiceInput) (*Service, error) {
	res, err := a.do("GET", fmt.Sprintf("/project/%s/service/%s", a.Project, params.ServiceName), nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusNotFound:
		return &Service{
			State:      Missing,
			UpdateTime: time.Now(),
		}, ErrInstanceDoesNotExist
	case http.StatusOK:
		break
	default:
		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("Error getting service: %d status code returned from Aiven: '%s'", res.StatusCode, b)
	}

	getServiceResponse := &GetServiceResponse{}
	if err := json.NewDecoder(res.Body).Decode(getServiceResponse); err != nil {
		return nil, err
	}

	service := getServiceResponse.Service

	if service.ServiceType == "" {
		return nil, errors.New("Error getting service: no service type found in response JSON")
	}

	if service.State == "" {
		return nil, errors.New("Error getting service: no state found in response JSON")
	}

	defaultTime := time.Time{}
	if service.UpdateTime == defaultTime {
		return nil, errors.New("Error getting service: no update_time found in response JSON")
	}

	return &service, nil
}

func (a *HttpClient) GetServiceTags(params *GetServiceTagsInput) (*ServiceTags, error) {
	res, err := a.do("GET", fmt.Sprintf("/project/%s/service/%s/tags", a.Project, params.ServiceName), nil)
	if err != nil {
		return &ServiceTags{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("Error getting service tags: %d status code returned from Aiven: '%s'", res.StatusCode, b)
	}

	getServiceResponse := &GetServiceTagsResponse{}
	if err := json.NewDecoder(res.Body).Decode(getServiceResponse); err != nil {
		return nil, err
	}

	return &getServiceResponse.Tags, nil

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
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

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

func (a *HttpClient) UpdateServiceTags(params *UpdateServiceTagsInput) (string, error) {
	reqBody, err := json.Marshal(params)
	if err != nil {
		return "", err
	}

	res, err := a.do("PUT", fmt.Sprintf("/project/%s/service/%s/tags", a.Project, params.ServiceName), reqBody)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	if res.StatusCode == http.StatusBadRequest {
		var errorResponse AivenErrorResponse
		jsonErr := json.Unmarshal(b, &errorResponse)
		if jsonErr != nil {
			return "", fmt.Errorf("Error updating service tags: %d status code returned from Aiven: '%s'", res.StatusCode, b)
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
	req, err := http.NewRequest(method, fmt.Sprintf("%s/v1%s", a.BaseURL, path), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("aivenv1 %s", a.Token))

	return req, err
}
