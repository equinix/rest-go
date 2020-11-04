package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
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
	assert.IsType(t, Error{}, err, "Error should be rest.Error type")
	restErr := err.(Error)
	assert.Equal(t, respCode, restErr.HTTPCode, "rest.Error should have valid httpCode")
	assert.Equal(t, http.StatusText(respCode), restErr.Message, "rest.Error should have valid Message")
	verifyErrorString(t, restErr, respCode, 1)
	assert.Equal(t, 1, len(restErr.ApplicationErrors), "rest.Error should have one application error")
	verifyApplicationError(t, resp, restErr.ApplicationErrors[0])
	verifyApplicationErrorString(t, restErr.ApplicationErrors[0].Error())
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
	assert.Equal(t, http.StatusText(respCode), restErr.Message, "rest.Error should have valid Message")
	verifyErrorString(t, restErr, respCode, len(resp))
	assert.Equal(t, len(resp), len(restErr.ApplicationErrors), "rest.Error should have valid number of application errors")
	for i := range restErr.ApplicationErrors {
		verifyApplicationError(t, resp[i], restErr.ApplicationErrors[i])
		verifyApplicationErrorString(t, restErr.ApplicationErrors[i].Error())
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

func verifyErrorString(t *testing.T, err Error, statusCode int, appErrorsLen int) {
	statusTxt := http.StatusText(statusCode)
	regexpStr := fmt.Sprintf("Message: \"%s\", HTTPCode: %d, ApplicationErrors: (\\[.+\\] ){%d}", statusTxt, statusCode, appErrorsLen)
	assert.Regexp(t, regexp.MustCompile(regexpStr), err.Error(), "Error produces valid error string")
}

func verifyApplicationError(t *testing.T, apiErr api.ErrorResponse, err ApplicationError) {
	assert.Equal(t, apiErr.ErrorCode, err.Code, "ErrorCode matches")
	assert.Equal(t, apiErr.Property, err.Property, "Property matches")
	assert.Equal(t, apiErr.ErrorMessage, err.Message, "ErrorMessage matches")
	assert.Equal(t, apiErr.MoreInfo, err.AdditionalInfo, "AdditionalInfo matches")
}

func verifyApplicationErrorString(t *testing.T, appErrorStr string) {
	assert.Regexp(t, regexp.MustCompile("^Code: \".*\", Property: \".*\", Message: \".*\", AdditionalInfo: \".*\"$"), appErrorStr, "ApplicationError produces valid error string")
}
