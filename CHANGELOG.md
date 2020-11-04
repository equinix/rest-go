## 1.1.0 (November 04, 2020)

ENHANCEMENTS:

* added `AdditionalInfo` property to `ApplicationError`
* string representation of both `Error` and `ApplicationError` was changed

## 1.0.0 (October 01, 2020)

NOTES:

* first version of Equinix rest-go module

FEATURES:

* Resty based client parses Equinix standardized error response body contents
* `GetPaginated` function queries for data on APIs with paginated responses. Pagination
 options can be configured by setting up attributes of `PagingConfig`