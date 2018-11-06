package errors

// Interface is exposed by errors that can be converted and consumed by virtual-kubelet.
type Interface interface {
	Status() *ErrorStatus
}

// ErrorReason is an enumeration of possible failure causes. The provider should return them properly
// when pod operation fails, so that the corresponding operation will be performed by the virtual-kubelet.
type ErrorReason string

const (
	// ErrorReasonUnknown means the server has declined to indicate a specific reason.
	ErrorReasonUnknown ErrorReason = ""

	// ErrorReasonUnauthorized means the server can be reached and understood the request, but requires
	// the user to present appropriate authorization credentials in order for the action to be completed.
	ErrorReasonUnauthorized ErrorReason = "Unauthorized"

	// ErrorReasonNotFound means one or more resources required for this operation could not be found.
	ErrorReasonNotFound ErrorReason = "NotFound"

	// ErrorReasonInternalServerError indicates that an internal error occurred, it is unexpected and the outcome of the call is unknown.
	// Details (optional):
	//   "retryAfterSeconds" int32 - the number of seconds before the operation should be retried
	ErrorReasonInternalServerError ErrorReason = "InternalServerError"

	// ErrorReasonServiceUnavailable means that the request itself was valid, but the requested service is unavailable at this time.
	// Retrying the request after some time might succeed.
	// Details (optional):
	//   "retryAfterSeconds" int32 - the number of seconds before the operation should be retried
	ErrorReasonServiceUnavailable ErrorReason = "ServiceUnavailable"

	// ErrorReasonTooManyRequests means the server experienced too many requests within a
	// given window and that the client must wait to perform the action again.
	// Details (optional):
	//   "retryAfterSeconds" int32 - the number of seconds before the operation should be retried
	ErrorReasonTooManyRequests ErrorReason = "TooManyRequests"

	// ErrorReasonBadRequest means that the request data was invalid.
	// The client should never retry on this failure.
	ErrorReasonBadRequest ErrorReason = "BadRequest"
)

// ErrorDetails is a set of additional properties that MAY be set by the provider for additional information
type ErrorDetails struct {
	RetryAfterSeconds int32
}

// ErrorStatus contains an error status from the provider.
type ErrorStatus struct {
	// A machine-readable description of why it failed.
	Reason  ErrorReason
	// Extended data associated with the reason.
	Details *ErrorDetails
	// An optional human-readable description of this error.
	Message string
	// The original error
	OrigErr error
}

func reasonForError(err error) ErrorReason {
	switch t := err.(type) {
	case Interface:
		return t.Status().Reason
	}
	return ErrorReasonUnknown
}

// OperationError is an error intended for consumption by virtual-kubelet.
type OperationError struct {
	ErrStatus *ErrorStatus
}

// Error implements the Error interface.
func (e *OperationError) Error() string {
	return e.ErrStatus.Message
}

// Status allows access to e's status without knowing the detailed workings of OperationError.
func (e *OperationError) Status() *ErrorStatus {
	return e.ErrStatus
}

// IsNotFound returns true if the specified error was NotFound.
func IsNotFound(err error) bool {
	return reasonForError(err) == ErrorReasonNotFound
}

// IsRetryable determines if err is an error which indicates that the request is retryable.
func IsRetryable(err error) bool {
	switch reasonForError(err){
	case ErrorReasonTooManyRequests:
	case ErrorReasonServiceUnavailable:
	case ErrorReasonInternalServerError:
		return true
	}

	return false
}

// SuggestsClientDelay returns true if this error suggests a client delay as well as the
// suggested seconds to wait, or false if the error does not imply a wait. Notice that it does not
// address whether the error *should* be retried.
func SuggestsClientDelay(err error) (int, bool) {
	switch t := err.(type) {
	case Interface:
		if t.Status().Details != nil {
			switch t.Status().Reason {
			// this StatusReason explicitly requests the caller to delay the action
			case ErrorReasonServiceUnavailable:
				return int(t.Status().Details.RetryAfterSeconds), true
			}
			// If the client requests that we retry after a certain number of seconds
			if t.Status().Details.RetryAfterSeconds > 0 {
				return int(t.Status().Details.RetryAfterSeconds), true
			}
		}
	}
	return 0, false
}

// NewNotFound returns an error indicates that the resource was not found.
func NewNotFound(message string, err error) *OperationError {
	return &OperationError{&ErrorStatus{
		Reason:  ErrorReasonNotFound,
		Message: message,
		OrigErr: err,
	}}
}

// NewUnauthorized returns an error indicating the client is not authorized to perform the requested
// action.
func NewUnauthorized(message string, err error) *OperationError {
	return &OperationError{&ErrorStatus{
		Reason:  ErrorReasonUnauthorized,
		Message: message,
		OrigErr: err,
	}}
}

// NewBadRequest creates an error indicating that the request is invalid and can not be processed.
func NewBadRequest(message string, err error) *OperationError {
	return &OperationError{&ErrorStatus{
		Reason:  ErrorReasonBadRequest,
		Message: message,
		OrigErr: err,
	}}
}

// NewTooManyRequests creates an error indicating that the client must try again later because
// the specified endpoint is not accepting requests. More specific details should be provided
// if client should know why the failure was limited4.
func NewTooManyRequests(message string, retryAfterSeconds int, err error) *OperationError {
	return &OperationError{&ErrorStatus{
		Reason:  ErrorReasonTooManyRequests,
		Message: message,
		Details: &ErrorDetails{
			RetryAfterSeconds: int32(retryAfterSeconds),
		},
		OrigErr: err,
	}}
}

// NewInternalServerError returns an error indicating that an internal error occurred,
// it is unexpected and the outcome of the call is unknown.
func NewInternalServerError(message string, retryAfterSeconds int, err error) *OperationError {
	return &OperationError{&ErrorStatus{
		Reason:  ErrorReasonInternalServerError,
		Message: message,
		Details: &ErrorDetails{
			RetryAfterSeconds: int32(retryAfterSeconds),
		},
		OrigErr: err,
	}}
}

// NewServiceUnavailable creates an error indicating that the requested service is unavailable.
func NewServiceUnavailable(message string, retryAfterSeconds int, err error) *OperationError {
	return &OperationError{&ErrorStatus{
		Reason:  ErrorReasonServiceUnavailable,
		Message: message,
		Details: &ErrorDetails{
			RetryAfterSeconds: int32(retryAfterSeconds),
		},
		OrigErr: err,
	}}
}

// NewUnknownError returns an error indicates that the error is unknown.
func NewUnknownError(message string, err error) *OperationError {
	return &OperationError{&ErrorStatus{
		Reason:  ErrorReasonUnknown,
		Message: message,
		OrigErr: err,
	}}
}
