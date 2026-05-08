package rpc

// ErrorCode identifies an application error in a stable, client-parseable form.
//
// The RPCError.Message field carries the ErrorCode enum string (e.g. "WIDGET_OVERLAP")
// rather than a free-form English sentence. The frontend looks up a localized
// message via i18n key errors.<code>. Numeric Code is retained for JSON-RPC
// interoperability but is a secondary identifier.
type ErrorCode string

// Canonical application error codes. Keep these in sync with
// web/src/api/rpcError.ts and i18n files.
const (
	ErrCodeValidation         ErrorCode = "VALIDATION"
	ErrCodeNotFound           ErrorCode = "NOT_FOUND"
	ErrCodeInternal           ErrorCode = "INTERNAL"
	ErrCodeWidgetNotFound     ErrorCode = "WIDGET_NOT_FOUND"
	ErrCodeWidgetOverlap      ErrorCode = "WIDGET_OVERLAP"
	ErrCodeWidgetInvalid      ErrorCode = "WIDGET_INVALID"
	ErrCodeDataSourceNotFound ErrorCode = "DATASOURCE_NOT_FOUND"
	ErrCodeDataSourceInvalid  ErrorCode = "DATASOURCE_INVALID"
	ErrCodeDataSourceConnect  ErrorCode = "DATASOURCE_CONNECT_FAILED"
	ErrCodeIconifyFailed      ErrorCode = "ICONIFY_FAILED"
	ErrCodeMetricFetchFailed  ErrorCode = "METRIC_FETCH_FAILED"
	ErrCodeBoardNotFound      ErrorCode = "BOARD_NOT_FOUND"
)

// Numeric aliases (historical JSON-RPC app band -32000..-32099). New code
// should prefer ErrorCode; numeric values here exist so existing clients keep
// working during the migration window.
const (
	NumCodeWidgetNotFound     = -32010
	NumCodeWidgetOverlap      = -32011
	NumCodeWidgetInvalid      = -32012
	NumCodeDataSourceNotFound = -32020
	NumCodeDataSourceInvalid  = -32021
	NumCodeDataSourceConnect  = -32022
	NumCodeMetricFetchFailed  = -32030
	NumCodeIconifyFailed      = -32050
	NumCodeBoardNotFound      = -32060
)

// RPCError is the error payload attached to failing responses.
//
// Message holds the enum ErrorCode string for stable client-side i18n lookup;
// human-facing text is resolved at the rendering layer. Data may carry
// structured context (e.g., overlap coordinates) for tooltip use.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Error satisfies the builtin error interface.
func (e *RPCError) Error() string { return e.Message }

// NewError allocates an RPC error payload with the given numeric code and
// free-form message. Prefer NewAppError for codified app errors.
func NewError(code int, message string, data any) *RPCError {
	return &RPCError{Code: code, Message: message, Data: data}
}

// NewAppError builds an application error keyed by ErrorCode. The numeric
// code is a stable alias; Message is the enum string for i18n lookup.
func NewAppError(code ErrorCode, data any) *RPCError {
	return &RPCError{Code: numericFor(code), Message: string(code), Data: data}
}

func numericFor(c ErrorCode) int {
	switch c {
	case ErrCodeWidgetNotFound:
		return NumCodeWidgetNotFound
	case ErrCodeWidgetOverlap:
		return NumCodeWidgetOverlap
	case ErrCodeWidgetInvalid, ErrCodeValidation:
		return NumCodeWidgetInvalid
	case ErrCodeDataSourceNotFound:
		return NumCodeDataSourceNotFound
	case ErrCodeDataSourceInvalid:
		return NumCodeDataSourceInvalid
	case ErrCodeDataSourceConnect:
		return NumCodeDataSourceConnect
	case ErrCodeMetricFetchFailed:
		return NumCodeMetricFetchFailed
	case ErrCodeIconifyFailed:
		return NumCodeIconifyFailed
	case ErrCodeBoardNotFound:
		return NumCodeBoardNotFound
	case ErrCodeNotFound:
		return CodeServerErrorRangeEnd
	case ErrCodeInternal:
		return CodeInternalError
	default:
		return CodeInternalError
	}
}
