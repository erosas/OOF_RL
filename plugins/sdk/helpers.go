package sdk

import "encoding/json"

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