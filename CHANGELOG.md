## 1.4.0 (November 09, 2026)

NOTES:

* updated several dependencies to keep up with security fixes
* updated minimum required Go version to 1.26

ENHANCEMENTS:

* Expanded error structed to accommodate the `details`,  `correlationId`, `help` and `additionalInfo` fields of  error responses, consistent with recent Equinix API standards.

## 1.3.0 (February 18, 2021)

FEATURES:

* introduced `GetOffsetPaginated` function that supports pagination controls with
limit and offset attributes in a separate json object within a response
* introduced `Do` function that runs same logic as previous `Execute` but returns
not only (potential) error but response as well

## 1.2.2 (February 12, 2021)

BUG FIXES:

* `GetPaginated` properly creates result slice when API structs use pointers
[equinix/terraform-provider-equinix#41](https://github.com/equinix/terraform-provider-equinix/issues/41)

## 1.2.1 (January 28, 2021)

BUG FIXES:

* pagination functions support API response structures with pointers

## 1.2.0 (January 15, 2021)

ENHANCEMENTS:

* setting up `EQUINIX_REST_LOG` environmental variable to `DEBUG` enables Resty
debug logging

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