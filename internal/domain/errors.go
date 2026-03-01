package domain

import "fmt"

var (
	ErrMissingQuery     = fmt.Errorf("title or --isbn is required (or use --batch)")
	ErrInvalidTitle     = fmt.Errorf("invalid title")
	ErrInvalidISBN      = fmt.Errorf("invalid isbn")
	ErrInvalidLang      = fmt.Errorf("invalid lang: allowed values are ko, en, ja, zh-tw")
	ErrInvalidBatchPath = fmt.Errorf("invalid batch path")
	ErrInvalidOutput    = fmt.Errorf("invalid output path")
	ErrMissingAPIKey    = fmt.Errorf("missing api key: set OPENROUTER_API_KEY or pass --api-key")
	ErrInvalidResponse  = fmt.Errorf("invalid response structure after retries")
	ErrBookNotFound     = fmt.Errorf("book not found")
)

type RetryableError struct {
	Code int
	Err  error
}

func (e *RetryableError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("retryable error (code=%d)", e.Code)
	}
	return fmt.Sprintf("retryable error (code=%d): %v", e.Code, e.Err)
}

func (e *RetryableError) Unwrap() error { return e.Err }

func IsRetryable(err error) bool {
	var re *RetryableError
	return err != nil && (fmt.Errorf("%w", err) != nil) && AsRetryable(err, &re)
}

func AsRetryable(err error, target **RetryableError) bool {
	if err == nil || target == nil {
		return false
	}
	re, ok := err.(*RetryableError)
	if ok {
		*target = re
		return true
	}
	type unwrapper interface {
		Unwrap() error
	}
	if uw, ok := err.(unwrapper); ok {
		return AsRetryable(uw.Unwrap(), target)
	}
	return false
}
