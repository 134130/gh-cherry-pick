package cli

import "errors"

type ReportedError struct {
	Err error
}

func (e *ReportedError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *ReportedError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func reported(err error) error {
	if err == nil {
		return nil
	}
	return &ReportedError{Err: err}
}

func IsReportedError(err error) bool {
	var reported *ReportedError
	return errors.As(err, &reported)
}
