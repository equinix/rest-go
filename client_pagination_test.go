package rest

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

type TestPaginatedResponse struct {
	T int          `json:"t"`
	P int          `json:"p"`
	S int          `json:"s"`
	L []TestObject `json:"l"`
}

type TestObject struct {
	Key string `json:"key"`
}

func TestGetPaginated(t *testing.T) {
	//given
	var pageOne, pageTwo, pageThree TestPaginatedResponse
	if err := ReadJSONData("./test-fixtures/paginated_resp_p0.json", &pageOne); err != nil {
		assert.Failf(t, "cannot read test response due to %s", err.Error())
	}
	if err := ReadJSONData("./test-fixtures/paginated_resp_p1.json", &pageTwo); err != nil {
		assert.Failf(t, "cannot read test response due to %s", err.Error())
	}
	if err := ReadJSONData("./test-fixtures/paginated_resp_p2.json", &pageThree); err != nil {
		assert.Failf(t, "cannot read test response due to %s", err.Error())
	}
	pageSize := 1
	testHc := &http.Client{}
	resourcePath := "/objects"
	httpmock.ActivateNonDefault(testHc)
	httpmock.RegisterResponder("GET", fmt.Sprintf("%s%s?s=%d", baseURL, resourcePath, pageSize),
		func(r *http.Request) (*http.Response, error) {
			resp, _ := httpmock.NewJsonResponse(200, pageOne)
			return resp, nil
		},
	)
	httpmock.RegisterResponder("GET", fmt.Sprintf("%s%s?p=2&s=%d", baseURL, resourcePath, pageSize),
		func(r *http.Request) (*http.Response, error) {
			resp, _ := httpmock.NewJsonResponse(200, pageTwo)
			return resp, nil
		},
	)
	httpmock.RegisterResponder("GET", fmt.Sprintf("%s%s?p=3&s=%d", baseURL, resourcePath, pageSize),
		func(r *http.Request) (*http.Response, error) {
			resp, _ := httpmock.NewJsonResponse(200, pageThree)
			return resp, nil
		},
	)

	//when
	c := NewClient(context.Background(), baseURL, testHc)
	c.SetPageSize(pageSize)
	content, err := c.GetPaginated(resourcePath, &TestPaginatedResponse{},
		DefaultPagingConfig().
			SetTotalCountFieldName("T").
			SetContentFieldName("L").
			SetPageParamName("p").
			SetSizeParamName("s").
			SetFirstPageNumber(1))

	//then
	assert.Nil(t, err, "Error should not be returned")
	assert.NotNil(t, content, "Content should not be nil")
	assert.Equal(t, 3, len(content), "")
	apiContent := make([]TestObject, 0, 3)
	apiContent = append(apiContent, pageOne.L...)
	apiContent = append(apiContent, pageTwo.L...)
	apiContent = append(apiContent, pageThree.L...)
	for i := range apiContent {
		assert.Equalf(t, apiContent[i].Key, content[i].(TestObject).Key, "Object %d key must match", i)
	}
}

func TestFieldValueFromStruct(t *testing.T) {
	//given
	type test struct {
		TestField *int
	}
	testFieldValue := 10
	input := test{&testFieldValue}
	//when
	value, err := getFieldValueFromStruct(input, "TestField", reflect.Int)
	//then
	assert.Nil(t, err, "Error is not returned")
	assert.Equal(t, testFieldValue, value.Interface().(int), "Value matches")
}
