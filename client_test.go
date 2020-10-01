package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/equinix/rest-go/internal/api"
	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

const (
	baseURL = "http://localhost:8888"
)

func TestSingleError(t *testing.T) {
	//given
	resp := api.ErrorResponse{}
	if err := ReadJSONData("./test-fixtures/error_resp.json", &resp); err != nil {
		assert.Fail(t, "Cannot read test response")
	}
	respCode := 500
	resourcePath := "/myObjects"
	testHc := SetupMockedClient(resty.MethodGet, baseURL+resourcePath, respCode, resp)
	defer httpmock.DeactivateAndReset()

	//when
	cli := NewClient(context.Background(), baseURL, testHc)
	err := cli.Execute(cli.R(), resty.MethodGet, resourcePath)

	//then
	assert.NotNil(t, err, "Error should be returned")
	assert.Contains(t, err.Error(), fmt.Sprintf("httpCode: %d", respCode), "Error message contains http status code")
	assert.IsType(t, Error{}, err, "Error should be rest.Error type")
	restErr := err.(Error)
	assert.Equal(t, respCode, restErr.HTTPCode, "rest.Error should have valid httpCode")
	assert.Equal(t, 1, len(restErr.ApplicationErrors), "rest.Error should have one application error")
	appError := restErr.ApplicationErrors[0]
	assert.Equal(t, resp.ErrorCode, appError.Code, "Application error code matches")
	assert.Contains(t, appError.Message, appError.Message, "Application error message contains response message")
}

func ReadJSONData(filePath string, target interface{}) error {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return err
	}
	return nil
}

func TestMultipleError(t *testing.T) {
	//given
	resp := api.ErrorResponses{}
	if err := ReadJSONData("./test-fixtures/errors_resp.json", &resp); err != nil {
		assert.Fail(t, "Cannot read test response")
	}
	respCode := 500
	resourcePath := "/myObjects"
	testHc := SetupMockedClient("GET", baseURL+resourcePath, respCode, resp)
	defer httpmock.DeactivateAndReset()

	//when
	cli := NewClient(context.Background(), baseURL, testHc)
	err := cli.Execute(cli.R(), resty.MethodGet, resourcePath)

	//then
	assert.NotNil(t, err, "Error should be returned")
	assert.IsType(t, Error{}, err, "Error should be rest.Error type")
	restErr := err.(Error)
	assert.Equal(t, respCode, restErr.HTTPCode, "rest.Error should have valid httpCode")
	assert.Equal(t, len(resp), len(restErr.ApplicationErrors), "rest.Error should have valid number of application errors")
	for i := range restErr.ApplicationErrors {
		assert.Equal(t, resp[i].ErrorCode, restErr.ApplicationErrors[i].Code, "Application error code matches")
		assert.Contains(t, restErr.ApplicationErrors[i].Message, resp[i].ErrorMessage, "Application error message contains response message")
	}
}

func SetupMockedClient(method string, url string, respCode int, resp interface{}) *http.Client {
	testHc := &http.Client{}
	httpmock.ActivateNonDefault(testHc)
	httpmock.RegisterResponder(method, url,
		func(r *http.Request) (*http.Response, error) {
			resp, _ := httpmock.NewJsonResponse(respCode, resp)
			return resp, nil
		},
	)
	return testHc
}
