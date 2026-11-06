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

type mockedEnvProvider struct {
	data map[string]string
}

func (e mockedEnvProvider) getEnv(key string) string {
	return e.data[key]
}

func TestIsDebugEnabled(t *testing.T) {
	//given
	envToLevelMapping := map[string]bool{
		"DEBUG": true,
		"fake":  false,
		"":      false,
	}
	for k, v := range envToLevelMapping {
		envProvider := mockedEnvProvider{map[string]string{LogLevelEnvVar: k}}
		//when
		level := isDebugEnabled(envProvider)
		//then
		assert.Equalf(t, v, level, "LogLevel for env variable %q = %q matches", LogLevelEnvVar, k)
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
	expectedAdditionalInfo := apiErr.Details
	if expectedAdditionalInfo == "" {
		expectedAdditionalInfo = apiErr.MoreInfo
	}
	assert.Equal(t, expectedAdditionalInfo, err.AdditionalInfo, "AdditionalInfo matches details, falling back to moreInfo")
	assert.Equal(t, apiErr.CorrelationID, err.CorrelationID, "CorrelationID matches")
	assert.Equal(t, apiErr.Help, err.Help, "Help matches")
}

func readFixture(t *testing.T, filePath string) string {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		t.Fatalf("cannot read fixture %s: %v", filePath, err)
	}
	return string(data)
}

func TestMapErrorBodyAPIToDomain(t *testing.T) {
	testCases := []struct {
		name     string
		body     string
		expected []ApplicationError
	}{
		{
			name: "details only",
			body: `{"errorCode": "EQ-1", "errorMessage": "msg", "details": "the details"}`,
			expected: []ApplicationError{
				{Code: "EQ-1", Message: "msg", AdditionalInfo: "the details"},
			},
		},
		{
			name: "legacy moreInfo only",
			body: `{"errorCode": "IC-1", "errorMessage": "msg", "property": "prop", "moreInfo": "the more info"}`,
			expected: []ApplicationError{
				{Code: "IC-1", Message: "msg", Property: "prop", AdditionalInfo: "the more info"},
			},
		},
		{
			name: "details preferred over moreInfo",
			body: `{"errorCode": "EQ-1", "errorMessage": "msg", "details": "the details", "moreInfo": "the more info"}`,
			expected: []ApplicationError{
				{Code: "EQ-1", Message: "msg", AdditionalInfo: "the details"},
			},
		},
		{
			name: "empty details falls back to moreInfo",
			body: `{"errorCode": "EQ-1", "errorMessage": "msg", "details": "", "moreInfo": "the more info"}`,
			expected: []ApplicationError{
				{Code: "EQ-1", Message: "msg", AdditionalInfo: "the more info"},
			},
		},
		{
			name: "correlationId and help",
			body: `[{"errorCode": "EQ-1", "errorMessage": "msg", "correlationId": "06ce37f6", "help": "https://example.com/help/error/EQ-1"}]`,
			expected: []ApplicationError{
				{Code: "EQ-1", Message: "msg", CorrelationID: "06ce37f6", Help: "https://example.com/help/error/EQ-1"},
			},
		},
		{
			name: "additionalInfo with property and reason",
			body: `[{"errorCode": "EQ-1", "errorMessage": "msg", "additionalInfo": [{"property": "speed", "reason": "out of range"}]}]`,
			expected: []ApplicationError{
				{Code: "EQ-1", Message: "msg", PropertyErrors: []PropertyError{{Property: "speed", Reason: "out of range"}}},
			},
		},
		{
			name: "additionalInfo with reason only",
			body: `[{"errorCode": "EQ-1", "errorMessage": "msg", "additionalInfo": [{"reason": "something is wrong"}]}]`,
			expected: []ApplicationError{
				{Code: "EQ-1", Message: "msg", PropertyErrors: []PropertyError{{Reason: "something is wrong"}}},
			},
		},
		{
			name: "additionalInfo entries without property and reason are dropped",
			body: `[{"errorCode": "EQ-1", "errorMessage": "msg", "additionalInfo": [
				{}, {"property": null, "reason": null}, {"key": "k", "value": "v"}, {"property": "speed", "reason": "out of range"}]}]`,
			expected: []ApplicationError{
				{Code: "EQ-1", Message: "msg", PropertyErrors: []PropertyError{{Property: "speed", Reason: "out of range"}}},
			},
		},
		{
			name: "additionalInfo with only null property and reason",
			body: `[{"errorCode": "EQ-1", "errorMessage": "msg", "additionalInfo": [{"property": null, "reason": null}]}]`,
			expected: []ApplicationError{
				{Code: "EQ-1", Message: "msg"},
			},
		},
		{
			name: "additionalInfo null",
			body: `[{"errorCode": "EQ-1", "errorMessage": "msg", "additionalInfo": null}]`,
			expected: []ApplicationError{
				{Code: "EQ-1", Message: "msg"},
			},
		},
		{
			name: "additionalInfo as string is ignored",
			body: `[{"errorCode": "EQ-1", "errorMessage": "msg", "details": "the details", "additionalInfo": "not an array"}]`,
			expected: []ApplicationError{
				{Code: "EQ-1", Message: "msg", AdditionalInfo: "the details"},
			},
		},
		{
			name: "additionalInfo as object is ignored",
			body: `{"errorCode": "EQ-1", "errorMessage": "msg", "moreInfo": "the more info", "additionalInfo": {"property": "speed"}}`,
			expected: []ApplicationError{
				{Code: "EQ-1", Message: "msg", AdditionalInfo: "the more info"},
			},
		},
		{
			name: "network edge delete device with active connections",
			body: readFixture(t, "./test-fixtures/ne_delete_device_resp.json"),
			expected: []ApplicationError{
				{
					Code:           "EQ-4006901",
					Message:        "Unable to delete device.",
					AdditionalInfo: "That Virtual Device has active Connections. Remove those active Connections before deleting the Virtual Device.",
					Help:           "https://<domain.name>/help/error/EQ-4006901",
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//when
			appErrors, ok := mapErrorBodyAPIToDomain([]byte(tc.body))
			//then
			assert.True(t, ok, "error body should be parsed")
			assert.Equal(t, tc.expected, appErrors, "application errors match")
		})
	}
}

func TestLegacyErrorString(t *testing.T) {
	//given
	resp := json.RawMessage(readFixture(t, "./test-fixtures/error_resp.json"))
	resourcePath := "/myObjects"
	testHc := SetupMockedClient(resty.MethodGet, baseURL+resourcePath, 500, resp)
	defer httpmock.DeactivateAndReset()

	//when
	cli := NewClient(context.Background(), baseURL, testHc)
	err := cli.Execute(cli.R(), resty.MethodGet, resourcePath)

	//then
	expected := `Equinix REST API error: Message: "Internal Server Error", HTTPCode: 500, ApplicationErrors: ` +
		`[Code: "IC-NE-ERR-500", Property: "", Message: "We're unable to process your request.", ` +
		`AdditionalInfo: "Please contact Network-Edge-Support@equinix.com for assistance."] `
	assert.EqualError(t, err, expected, "legacy moreInfo error string is unchanged")
}

func TestNetworkEdgeDeleteDeviceErrorString(t *testing.T) {
	//given
	resp := json.RawMessage(readFixture(t, "./test-fixtures/ne_delete_device_resp.json"))
	resourcePath := "/ne/v1/devices/myDevice"
	testHc := SetupMockedClient(resty.MethodDelete, baseURL+resourcePath, 403, resp)
	defer httpmock.DeactivateAndReset()

	//when
	cli := NewClient(context.Background(), baseURL, testHc)
	err := cli.Execute(cli.R(), resty.MethodDelete, resourcePath)

	//then
	expected := `Equinix REST API error: Message: "Forbidden", HTTPCode: 403, ApplicationErrors: ` +
		`[Code: "EQ-4006901", Property: "", Message: "Unable to delete device.", ` +
		`AdditionalInfo: "That Virtual Device has active Connections. Remove those active Connections before deleting the Virtual Device.", ` +
		`Help: "https://<domain.name>/help/error/EQ-4006901"] `
	assert.EqualError(t, err, expected, "null additionalInfo entry adds nothing to error string")
}

func TestStandardErrors(t *testing.T) {
	//given
	resp := api.ErrorResponses{}
	if err := ReadJSONData("./test-fixtures/errors_resp_standard.json", &resp); err != nil {
		assert.Fail(t, "Cannot read test response")
	}
	// serve fixture as-is, round trip through api structs would drop custom additionalInfo properties
	rawResp := json.RawMessage(readFixture(t, "./test-fixtures/errors_resp_standard.json"))
	respCode := 400
	resourcePath := "/myObjects"
	testHc := SetupMockedClient(resty.MethodPost, baseURL+resourcePath, respCode, rawResp)
	defer httpmock.DeactivateAndReset()

	//when
	cli := NewClient(context.Background(), baseURL, testHc)
	err := cli.Execute(cli.R(), resty.MethodPost, resourcePath)

	//then
	assert.NotNil(t, err, "Error should be returned")
	assert.IsType(t, Error{}, err, "Error should be rest.Error type")
	restErr := err.(Error)
	assert.Equal(t, respCode, restErr.HTTPCode, "rest.Error should have valid httpCode")
	verifyErrorString(t, restErr, respCode, len(resp))
	assert.Equal(t, len(resp), len(restErr.ApplicationErrors), "rest.Error should have valid number of application errors")
	for i := range restErr.ApplicationErrors {
		verifyApplicationError(t, resp[i], restErr.ApplicationErrors[i])
		verifyApplicationErrorString(t, restErr.ApplicationErrors[i].Error())
	}
	assert.Nil(t, restErr.ApplicationErrors[0].PropertyErrors, "error without additionalInfo has no property errors")
	assert.Equal(t, []PropertyError{
		{Property: "work.emailId", Reason: "Work email address is mandatory"},
	}, restErr.ApplicationErrors[1].PropertyErrors, "single property error is mapped")
	assert.Nil(t, restErr.ApplicationErrors[2].PropertyErrors, "error without additionalInfo has no property errors")
	assert.Equal(t, []PropertyError{
		{Property: "peerIp", Reason: "Invalid IP address format for peerId"},
		{Property: "secondary.peerIp", Reason: "Invalid IP address format for secondary peerId"},
		{Property: "speed", Reason: "Connection speed value should be in the range of 1G-10G"},
	}, restErr.ApplicationErrors[3].PropertyErrors, "multiple property errors are mapped, custom properties ignored")
	assert.Contains(t, restErr.Error(),
		`PropertyErrors: [{Property: "work.emailId", Reason: "Work email address is mandatory"}]`,
		"single property error is in error string")
	assert.Contains(t, restErr.Error(),
		`PropertyErrors: [{Property: "peerIp", Reason: "Invalid IP address format for peerId"}, `+
			`{Property: "secondary.peerIp", Reason: "Invalid IP address format for secondary peerId"}, `+
			`{Property: "speed", Reason: "Connection speed value should be in the range of 1G-10G"}]`,
		"multiple property errors are in error string")
}

func TestApplicationErrorString(t *testing.T) {
	testCases := []struct {
		name     string
		appError ApplicationError
		expected string
	}{
		{
			name:     "legacy fields only",
			appError: ApplicationError{Code: "IC-1", Property: "prop", Message: "msg", AdditionalInfo: "info"},
			expected: `Code: "IC-1", Property: "prop", Message: "msg", AdditionalInfo: "info"`,
		},
		{
			name: "correlationId and help",
			appError: ApplicationError{Code: "EQ-1", Message: "msg", AdditionalInfo: "info",
				CorrelationID: "06ce37f6", Help: "https://example.com/help/error/EQ-1"},
			expected: `Code: "EQ-1", Property: "", Message: "msg", AdditionalInfo: "info", ` +
				`CorrelationID: "06ce37f6", Help: "https://example.com/help/error/EQ-1"`,
		},
		{
			name:     "empty property errors",
			appError: ApplicationError{Code: "EQ-1", Message: "msg", PropertyErrors: []PropertyError{}},
			expected: `Code: "EQ-1", Property: "", Message: "msg", AdditionalInfo: ""`,
		},
		{
			name: "all fields",
			appError: ApplicationError{Code: "EQ-1", Property: "prop", Message: "msg", AdditionalInfo: "info",
				CorrelationID: "06ce37f6", Help: "https://example.com/help/error/EQ-1",
				PropertyErrors: []PropertyError{
					{Property: "peerIp", Reason: "Invalid IP address"},
					{Reason: "Something else"},
				}},
			expected: `Code: "EQ-1", Property: "prop", Message: "msg", AdditionalInfo: "info", ` +
				`CorrelationID: "06ce37f6", Help: "https://example.com/help/error/EQ-1", ` +
				`PropertyErrors: [{Property: "peerIp", Reason: "Invalid IP address"}, {Property: "", Reason: "Something else"}]`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.appError.Error(), "error string matches")
		})
	}
}

func verifyApplicationErrorString(t *testing.T, appErrorStr string) {
	assert.Regexp(t, regexp.MustCompile("^Code: \".*\", Property: \".*\", Message: \".*\", AdditionalInfo: \".*\"(, .+)?$"), appErrorStr, "ApplicationError produces valid error string")
}
