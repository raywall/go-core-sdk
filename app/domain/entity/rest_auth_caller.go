package entity

import "time"

// HTTPMethod restricts the accepted values for an outbound API call to the
// standard HTTP verbs supported by RestAuthCaller.
type HTTPMethod string

const (
	MethodGet    HTTPMethod = "GET"
	MethodPost   HTTPMethod = "POST"
	MethodPut    HTTPMethod = "PUT"
	MethodPatch  HTTPMethod = "PATCH"
	MethodDelete HTTPMethod = "DELETE"
)

// Valid reports whether m is one of the supported HTTP methods.
func (m HTTPMethod) Valid() bool {
	switch m {
	case MethodGet, MethodPost, MethodPut, MethodPatch, MethodDelete:
		return true
	default:
		return false
	}
}

// APIRequest describes a single outbound call that RestAuthCaller must
// execute on behalf of the caller, with the bearer token attached
// transparently. Every field beyond Method and URL is optional.
type APIRequest struct {
	Method     HTTPMethod
	URL        string
	Headers    map[string]string
	Body       []byte
	Retries    int           // number of retries after the first attempt
	RetryDelay time.Duration // wait time between attempts
	Timeout    time.Duration // per-attempt timeout; 0 means no explicit timeout
}

// APIResponse is the normalized result of an API call executed through
// RestAuthCaller.
type APIResponse struct {
	StatusCode int
	Headers    map[string][]string
	Body       []byte
}
