//Package rest implements Equinix REST client
package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/equinix/rest-go/internal/api"
	"github.com/go-resty/resty/v2"
)

const (
	//LogLevelEnvVar is OS variable name that controlls logging level
	LogLevelEnvVar = "EQUINIX_REST_LOG"
)

//Client describes Equinix REST client implementation.
//Implementation is based on github.com/go-resty
type Client struct {
	//PageSize determines default page size for GET requests on resource collections
	PageSize int
	baseURL  string
	ctx      context.Context
	*resty.Client
}

//Error describes REST API error
type Error struct {
	//HTTPCode is HTTP status code
	HTTPCode int
	//Message is textual, general description of an error
	Message string
	//ApplicationErrors is list of one or more application sub-errors
	ApplicationErrors []ApplicationError
}

//ApplicationError describes standardized application error
type ApplicationError struct {
	//Code is short error identifier
	Code string
	//Message is textual description of an error
	Message string
	//Property is a name of resource property that is related to an error
	Property string
	//AdditionalInfo provides additional information about an error
	AdditionalInfo string
	//CorrelationID identifies the failed request, for troubleshooting
	CorrelationID string
	//Help is a URL to a help page for the error code
	Help string
	//PropertyErrors lists property specific reasons for an error
	PropertyErrors []PropertyError
}

//PropertyError describes an error related to a single request property
type PropertyError struct {
	//Property is a name of request property (query, path or body) that is related to an error
	Property string
	//Reason is textual description of an error specific to the property
	Reason string
}

func (e Error) Error() string {
	var errorStr = fmt.Sprintf("Equinix REST API error: Message: %q", e.Message)
	if e.HTTPCode > 0 {
		errorStr = fmt.Sprintf("%s, HTTPCode: %d", errorStr, e.HTTPCode)
	}
	var appErrorsStr string
	for _, appError := range e.ApplicationErrors {
		appErrorsStr += "[" + appError.Error() + "] "
	}
	if len(appErrorsStr) > 0 {
		errorStr += ", ApplicationErrors: " + appErrorsStr
	}
	return errorStr
}

func (e ApplicationError) Error() string {
	errorStr := fmt.Sprintf("Code: %q, Property: %q, Message: %q, AdditionalInfo: %q", e.Code, e.Property, e.Message, e.AdditionalInfo)
	if e.CorrelationID != "" {
		errorStr += fmt.Sprintf(", CorrelationID: %q", e.CorrelationID)
	}
	if e.Help != "" {
		errorStr += fmt.Sprintf(", Help: %q", e.Help)
	}
	if len(e.PropertyErrors) > 0 {
		propErrorStrs := make([]string, len(e.PropertyErrors))
		for i, propError := range e.PropertyErrors {
			propErrorStrs[i] = fmt.Sprintf("{Property: %q, Reason: %q}", propError.Property, propError.Reason)
		}
		errorStr += ", PropertyErrors: [" + strings.Join(propErrorStrs, ", ") + "]"
	}
	return errorStr
}

//NewClient creates new Equinix REST client with a given HTTP context, URL and http client.
//Equinix REST client is based on github.com/go-resty
func NewClient(ctx context.Context, baseURL string, httpClient *http.Client) *Client {
	resty := resty.NewWithClient(httpClient)
	resty.SetHeader("Accept", "application/json")
	resty.SetDebug(isDebugEnabled(osEnvProvider{}))
	return &Client{
		100,
		baseURL,
		ctx,
		resty}
}

//SetPageSize sets  page size used by Equinix REST client for paginated queries
func (c *Client) SetPageSize(pageSize int) *Client {
	c.PageSize = pageSize
	return c
}

//Execute runs provided request using provider http method and path
func (c *Client) Execute(req *resty.Request, method string, path string) error {
	_, err := c.Do(method, path, req)
	return err
}

//Do runs given method on a given path with given request and returns response and error
func (c *Client) Do(method string, path string, req *resty.Request) (*resty.Response, error) {
	if path[0:1] == "/" {
		path = path[1:]
	}
	url := c.baseURL + "/" + path
	resp, err := req.SetContext(c.ctx).Execute(method, url)
	if err != nil {
		restErr := Error{Message: "HTTP operation failed: " + err.Error()}
		if resp != nil {
			restErr.HTTPCode = resp.StatusCode()
		}
		return resp, restErr
	}
	if resp.IsError() {
		return resp, createError(resp)
	}
	return resp, nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported package methods
//_______________________________________________________________________

func mapErrorBodyAPIToDomain(body []byte) ([]ApplicationError, bool) {
	apiError := api.ErrorResponse{}
	if err := json.Unmarshal(body, &apiError); err == nil {
		return mapApplicationErrorsAPIToDomain([]api.ErrorResponse{apiError}), true
	}
	apiErrors := api.ErrorResponses{}
	if err := json.Unmarshal(body, &apiErrors); err == nil {
		return mapApplicationErrorsAPIToDomain(apiErrors), true
	}
	return nil, false
}

func mapApplicationErrorsAPIToDomain(apiErrors api.ErrorResponses) []ApplicationError {
	transformed := make([]ApplicationError, len(apiErrors))
	for i := range apiErrors {
		transformed[i] = mapApplicationErrorAPIToDomain(apiErrors[i])
	}
	return transformed
}

func mapApplicationErrorAPIToDomain(apiError api.ErrorResponse) ApplicationError {
	// older APIs put additional information about the error in a field called
	// `moreInfo`. Current API standards names this field `details`.
	var additionalInfo = apiError.Details
	if additionalInfo == "" && apiError.MoreInfo != "" {
		additionalInfo = apiError.MoreInfo
	}

	return ApplicationError{
		Code:           apiError.ErrorCode,
		Property:       apiError.Property,
		Message:        apiError.ErrorMessage,
		AdditionalInfo: additionalInfo,
		CorrelationID:  apiError.CorrelationID,
		Help:           apiError.Help,
		PropertyErrors: mapAdditionalInfoAPIToDomain(apiError.AdditionalInfo),
	}
}

func mapAdditionalInfoAPIToDomain(rawAdditionalInfo json.RawMessage) []PropertyError {
	if len(rawAdditionalInfo) == 0 {
		return nil
	}
	apiAdditionalInfo := []api.ErrorAdditionalInfo{}
	// additionalInfo with a non-standard shape is ignored, it must not
	// prevent the rest of the error from being mapped
	if err := json.Unmarshal(rawAdditionalInfo, &apiAdditionalInfo); err != nil {
		return nil
	}
	var transformed []PropertyError
	for _, info := range apiAdditionalInfo {
		if info.Property == "" && info.Reason == "" {
			continue
		}
		transformed = append(transformed, PropertyError{Property: info.Property, Reason: info.Reason})
	}
	return transformed
}

func createError(resp *resty.Response) Error {
	respBody := resp.Body()
	err := Error{}
	err.HTTPCode = resp.StatusCode()
	err.Message = http.StatusText(err.HTTPCode)
	appErrors, ok := mapErrorBodyAPIToDomain(respBody)
	if !ok {
		err.Message = string(respBody)
		return err
	}
	err.ApplicationErrors = appErrors
	return err
}

type envProvider interface {
	getEnv(key string) string
}

type osEnvProvider struct {
}

func (osEnvProvider) getEnv(key string) string {
	return os.Getenv(key)
}

func isDebugEnabled(envProvider envProvider) bool {
	envLevel := envProvider.getEnv(LogLevelEnvVar)
	switch envLevel {
	case "DEBUG":
		return true
	default:
		return false
	}
}
