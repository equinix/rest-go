//Package rest implements Equinix REST client
package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/equinix/rest-go/internal/api"
	"github.com/go-resty/resty/v2"
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
}

func (e Error) Error() string {
	return fmt.Sprintf("Equinix rest error: httpCode: %v, message: %v", e.HTTPCode, e.Message)
}

//NewClient creates new Equinix REST client with a given HTTP context, URL and http client.
//Equinix REST client is based on github.com/go-resty
func NewClient(ctx context.Context, baseURL string, httpClient *http.Client) *Client {
	resty := resty.NewWithClient(httpClient)
	resty.SetHeader("Accept", "application/json")
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
	if path[0:1] == "/" {
		path = path[1:]
	}
	url := c.baseURL + "/" + path
	resp, err := req.SetContext(c.ctx).Execute(method, url)
	if err != nil {
		restErr := Error{Message: fmt.Sprintf("operation failed: %s", err)}
		if resp != nil {
			restErr.HTTPCode = resp.StatusCode()
		}
		return restErr
	}
	if resp.IsError() {
		err := transformErrorBody(resp.Body())
		err.HTTPCode = resp.StatusCode()
		return err
	}
	return nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported package methods
//_______________________________________________________________________

func transformErrorBody(body []byte) Error {
	apiError := api.ErrorResponse{}
	if err := json.Unmarshal(body, &apiError); err == nil {
		return mapErrorAPIToDomain(apiError)
	}
	apiErrors := api.ErrorResponses{}
	if err := json.Unmarshal(body, &apiErrors); err == nil {
		return mapErrorsAPIToDomain(apiErrors)
	}
	return Error{
		Message: string(body)}
}

func mapErrorAPIToDomain(apiError api.ErrorResponse) Error {
	return Error{
		Message: apiError.ErrorMessage,
		ApplicationErrors: []ApplicationError{{
			apiError.ErrorCode,
			fmt.Sprintf("[Error: Property: %v, %v]", apiError.Property, apiError.ErrorMessage),
		}},
	}
}

func mapErrorsAPIToDomain(apiErrors api.ErrorResponses) Error {
	errors := make([]ApplicationError, len(apiErrors))
	msg := ""
	for i, v := range apiErrors {
		errors[i] = ApplicationError{v.ErrorCode, v.ErrorMessage}
		msg = msg + fmt.Sprintf(" [Error %v: Property: %v, %v]", i+1, v.Property, v.ErrorMessage)
	}
	return Error{
		Message:           "Multiple errors occurred: " + msg,
		ApplicationErrors: errors,
	}
}
