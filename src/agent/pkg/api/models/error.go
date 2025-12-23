
// input: N/A (model definition)
// output: API error response models
// pos: API error models - if file updated, must sync with this header comment and pkg/api/CLAUDE.md
package models

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error   string      `json:"error"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
	Code    int         `json:"code"`
}

// NewErrorResponse creates a new error response
func NewErrorResponse(code int, err string, message string, details interface{}) *ErrorResponse {
	return &ErrorResponse{
		Error:   err,
		Message: message,
		Details: details,
		Code:    code,
	}
}

// ValidationError represents a field validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}
