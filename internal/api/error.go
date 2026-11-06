package api

import "encoding/json"

//ErrorResponses describes error response built with
//multiple error responses
type ErrorResponses []ErrorResponse

//ErrorResponse describes error response with standardized
//application error description
type ErrorResponse struct {
	ErrorCode     string `json:"errorCode,omitempty"`
	ErrorMessage  string `json:"errorMessage,omitempty"`
	MoreInfo      string `json:"moreInfo,omitempty"`
	Property      string `json:"property,omitempty"`
	Details       string `json:"details,omitempty"`
	CorrelationID string `json:"correlationId,omitempty"`
	Help          string `json:"help,omitempty"`
	//AdditionalInfo is kept raw so that a non-standard shape cannot
	//fail decoding of the whole error response
	AdditionalInfo json.RawMessage `json:"additionalInfo,omitempty"`
}

//ErrorAdditionalInfo describes reserved properties of a single
//additionalInfo entry of standardized error response
type ErrorAdditionalInfo struct {
	Property string `json:"property,omitempty"`
	Reason   string `json:"reason,omitempty"`
}
