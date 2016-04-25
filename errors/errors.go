package errors

// AggregatedError is an error contains multiple errors
type AggregatedError struct {
	// Errors are contained errors
	Errors []error
}

// AddErr explicitly adds one error
// If error is nil, nothing happens
// If error is AggregatedError, the contained errors are merged
func (e *AggregatedError) AddErr(err error) error {
	if err != nil {
		if aggregatedErrs, ok := err.(*AggregatedError); ok {
			e.Errors = append(e.Errors, aggregatedErrs.Errors...)
		} else {
			e.Errors = append(e.Errors, err)
		}
	}
	return err
}

// Add adds one error and returns true if the error is added
func (e *AggregatedError) Add(err error) bool {
	return e.AddErr(err) != nil
}

// AddMany adds arbitrary number of errors
func (e *AggregatedError) AddMany(errs ...error) *AggregatedError {
	for _, err := range errs {
		e.AddErr(err)
	}
	return e
}

// Aggregate returns AggregatedError if it contains some errors, or returns nil
func (e *AggregatedError) Aggregate() error {
	if len(e.Errors) > 0 {
		return e
	}
	return nil
}

// Error implements error
func (e *AggregatedError) Error() string {
	if len(e.Errors) > 0 {
		msg := "Multiple Errors:"
		for _, err := range e.Errors {
			msg += "\n" + err.Error()
		}
		return msg
	}
	return ""
}
