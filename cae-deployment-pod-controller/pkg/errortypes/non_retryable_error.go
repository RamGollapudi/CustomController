package errortypes

import "fmt"

// New returns an error that formats as the given text.
func New(text string) error {
	return &NonRetryableError{text}
}

// NonRetryableError is an implementation of error that is meant
// to be non-retryable in nature. You can type check any error
// to see if it is retryable or not.
type NonRetryableError struct {
	s string
}

func (e *NonRetryableError) Error() string {
	return e.s
}

// Errorf formats according to a format specifier and returns the string
// as a value that satisfies error.
func Errorf(format string, a ...interface{}) error {
	return New(fmt.Sprintf(format, a...))
}
