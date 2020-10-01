# Equinix REST client

Equinix REST client written in Go.
Implementation is based on [Resty client](https://github.com/go-resty/resty).

## Purpose

Purpose of this module is to wrap Resty REST client with Equinix specific error handling.
In addition, module adds support for paginated query requests.

Module is used by other Equinix client libraries, like [ECX Fabric Go client](https://github.com/equinix/ecx-go)
or [Network Edge Go client](https://github.com/equinix/ne-go).

## Features

* parses Equinix standardized error response body contents
* `GetPaginated` function queries for data on APIs with paginated responses. Pagination
 options can be configured by setting up attributes of `PagingConfig`

## Usage

1. Get recent equinix/rest-go module

   ```sh
   go get -d github.com/equinix/rest-go
   ```

2. Create new Equinix REST client with default HTTP client

   ```go
   import (
       "context"
       "net/http"
       "github.com/equinix/rest-go"
   )

   func main() {
     c := rest.NewClient(
          context.Background(),
          "https://api.equinix.com",
          &http.Client{})
   }
   ```

3. Use Equinix HTTP client with Equinix APIs

   ```go
    respBody := api.AccountResponse{}
    req := c.R().SetResult(&respBody)
    if err := c.Execute(req, "GET", "/ne/v1/device/account"); err != nil {
     //Equinix application error details will be included
     log.Printf("Got error: %s", err) 
    }
   ```
