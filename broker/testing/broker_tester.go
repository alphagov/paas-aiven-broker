package testing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"

	"github.com/pivotal-cf/brokerapi"
)

type BrokerTester struct {
	credentials brokerapi.BrokerCredentials
	brokerAPI   http.Handler
}

func New(credentials brokerapi.BrokerCredentials, brokerAPI http.Handler) BrokerTester {
	return BrokerTester{
		credentials: credentials,
		brokerAPI:   brokerAPI,
	}
}

type RequestBody struct {
	ServiceID        string       `json:"service_id,omitempty"`
	PlanID           string       `json:"plan_id,omitempty"`
	OrganizationGUID string       `json:"organization_guid,omitempty"`
	SpaceGUID        string       `json:"space_guid,omitempty"`
	AppGUID          string       `json:"app_guid,omitempty"`
	PreviousValues   *RequestBody `json:"previous_values,omitempty"`
}

func (bt BrokerTester) Services() *httptest.ResponseRecorder {
	return bt.Get("/v2/catalog", url.Values{})
}

func (bt BrokerTester) Provision(instanceID string, body RequestBody, async bool) *httptest.ResponseRecorder {
	bodyJSON, _ := json.Marshal(body)
	return bt.Put(
		"/v2/service_instances/"+instanceID,
		bytes.NewBuffer(bodyJSON),
		url.Values{"accepts_incomplete": []string{strconv.FormatBool(async)}},
	)
}

func (bt BrokerTester) Deprovision(instanceID string, async bool) *httptest.ResponseRecorder {
	return bt.Delete(
		"/v2/service_instances/"+instanceID,
		nil,
		url.Values{"accepts_incomplete": []string{strconv.FormatBool(async)}},
	)
}

func (bt BrokerTester) Bind(instanceID, bindingID string, body RequestBody) *httptest.ResponseRecorder {
	bodyJSON, _ := json.Marshal(body)
	return bt.Put(
		fmt.Sprintf("/v2/service_instances/%s/service_bindings/%s", instanceID, bindingID),
		bytes.NewBuffer(bodyJSON),
		url.Values{},
	)
}

func (bt BrokerTester) Unbind(instanceID, bindingID string, body RequestBody) *httptest.ResponseRecorder {
	bodyJSON, _ := json.Marshal(body)
	return bt.Delete(
		fmt.Sprintf(
			"/v2/service_instances/%s/service_bindings/%s",
			instanceID,
			bindingID,
		),
		bytes.NewBuffer(bodyJSON),
		url.Values{},
	)
}

func (bt BrokerTester) Update(instanceID string, body RequestBody, async bool) *httptest.ResponseRecorder {
	bodyJSON, _ := json.Marshal(body)
	return bt.Patch(
		"/v2/service_instances/"+instanceID,
		bytes.NewBuffer(bodyJSON),
		url.Values{"accepts_incomplete": []string{strconv.FormatBool(async)}},
	)
}

func (bt BrokerTester) LastOperation(instanceID, serviceID, planID, operation string) *httptest.ResponseRecorder {
	urlValues := url.Values{}
	if serviceID != "" {
		urlValues.Add("service_id", serviceID)
	}
	if planID != "" {
		urlValues.Add("plan_id", planID)
	}
	if operation != "" {
		urlValues.Add("operation", operation)
	}
	return bt.Get(
		fmt.Sprintf("/v2/service_instances/%s/last_operation", instanceID),
		urlValues,
	)
}

func (bt BrokerTester) Get(path string, params url.Values) *httptest.ResponseRecorder {
	return bt.do(bt.newRequest("GET", path, nil, params))
}

func (bt BrokerTester) Put(path string, body io.Reader, params url.Values) *httptest.ResponseRecorder {
	return bt.do(bt.newRequest("PUT", path, body, params))
}

func (bt BrokerTester) Patch(path string, body io.Reader, params url.Values) *httptest.ResponseRecorder {
	return bt.do(bt.newRequest("PATCH", path, body, params))
}

func (bt BrokerTester) Delete(path string, body io.Reader, params url.Values) *httptest.ResponseRecorder {
	return bt.do(bt.newRequest("DELETE", path, body, params))
}

func (bt BrokerTester) newRequest(method, path string, body io.Reader, params url.Values) *http.Request {
	url := fmt.Sprintf("http://%s", "127.0.0.1:8080"+path)
	req := httptest.NewRequest(method, url, body)
	req.URL.RawQuery = params.Encode()
	return req
}

func (bt BrokerTester) do(req *http.Request) *httptest.ResponseRecorder {
	res := httptest.NewRecorder()
	req.SetBasicAuth(bt.credentials.Username, bt.credentials.Password)
	bt.brokerAPI.ServeHTTP(res, req)
	return res
}
