package sdk

import (
	"encoding/json"
	"net/url"
	"time"
)

// JSONResponse constructs a 200 OK HTTPResponse with a JSON body.
func JSONResponse(body []byte) HTTPResponse {
	return HTTPResponse{
		Status:  200,
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    string(body),
	}
}

// JSONError constructs an error HTTPResponse with a {"error":"..."} JSON body.
func JSONError(status int, msg string) HTTPResponse {
	b, _ := json.Marshal(map[string]string{"error": msg})
	return HTTPResponse{
		Status:  status,
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    string(b),
	}
}

// ParseTime parses a time string trying RFC3339Nano, RFC3339, and two
// common SQLite datetime formats. Returns the zero value on failure.
func ParseTime(s string) time.Time {
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// QueryParam extracts a single value from a URL-encoded query string.
// Returns an empty string if the key is absent or the query is malformed.
func QueryParam(query, key string) string {
	vals, err := url.ParseQuery(query)
	if err != nil {
		return ""
	}
	return vals.Get(key)
}

// ProxyResponse wraps an HTTPFetchResult into an HTTPResponse, passing the
// status and body through as-is. Returns a 502 JSON error if the fetch failed.
func ProxyResponse(res HTTPFetchResult) HTTPResponse {
	if res.Error != "" {
		return JSONError(502, res.Error)
	}
	return HTTPResponse{
		Status:  res.Status,
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    res.Body,
	}
}