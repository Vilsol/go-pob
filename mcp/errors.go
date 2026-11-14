package mcp

import (
	"fmt"
)

type ErrorCode string

const (
	CodeUnknownHandle   ErrorCode = "UNKNOWN_HANDLE"
	CodeInvalidArgument ErrorCode = "INVALID_ARGUMENT"
	CodeNotFound        ErrorCode = "NOT_FOUND"
	CodeAmbiguous       ErrorCode = "AMBIGUOUS"
	CodeInvalidSlot     ErrorCode = "INVALID_SLOT"
	CodeInvalidValue    ErrorCode = "INVALID_VALUE"
	CodeNotReachable    ErrorCode = "NOT_REACHABLE"
	CodeAffixLimit      ErrorCode = "AFFIX_LIMIT"
	CodeParseError      ErrorCode = "PARSE_ERROR"
	CodeImportFailed    ErrorCode = "IMPORT_FAILED"
	CodeUnsupportedGame ErrorCode = "UNSUPPORTED_GAME"
	CodeTreeNotLatest   ErrorCode = "TREE_NOT_LATEST"
	CodeDataLoading     ErrorCode = "DATA_LOADING"
	CodeSpaceTooLarge   ErrorCode = "SPACE_TOO_LARGE"
	CodeUnknownStat     ErrorCode = "UNKNOWN_STAT"
	CodeInternal        ErrorCode = "INTERNAL"
)

// ToolError is the body of every failed tool result: {"error": {...}}.
type ToolError struct {
	Code       ErrorCode `json:"code"`
	Message    string    `json:"message"`
	DidYouMean []string  `json:"did_you_mean,omitempty"`
	Hint       string    `json:"hint,omitempty"`
	OpIndex    *int      `json:"op_index,omitempty"`
	SlotIndex  *int      `json:"slot_index,omitempty"`

	ValidHandles []string `json:"valid_handles,omitempty"`
	Valid        []string `json:"valid,omitempty"`
	Candidates   []any    `json:"candidates,omitempty"`
	ValidSlots   []string `json:"valid_slots,omitempty"`
	Allowed      []any    `json:"allowed,omitempty"`
	Reason       string   `json:"reason,omitempty"`
	Chosen       []string `json:"chosen,omitempty"`
	AllowedBase  []string `json:"allowed_for_base,omitempty"`
	Line         string   `json:"line,omitempty"`
	Site         string   `json:"site,omitempty"`
	Status       int      `json:"upstream_status,omitempty"`
	RetryAfterMs int      `json:"retry_after_ms,omitempty"`
	SizePerSlot  []any    `json:"size_per_slot,omitempty"`
	Suggestions  []string `json:"suggestions,omitempty"`
	Revision     *int     `json:"revision,omitempty"`
}

func errf(code ErrorCode, format string, args ...any) *ToolError {
	return &ToolError{Code: code, Message: fmt.Sprintf(format, args...)}
}

func (e *ToolError) atOp(i int) {
	if e.OpIndex == nil {
		e.OpIndex = &i
	}
}

func (e *ToolError) atSlot(i int) *ToolError {
	if e.SlotIndex == nil {
		e.SlotIndex = &i
	}
	return e
}

func notFound(kind, name string, suggestions []string, hint string) *ToolError {
	e := errf(CodeNotFound, "No %s named %q.", kind, name)
	e.DidYouMean = suggestions
	if hint == "" && len(suggestions) > 0 {
		hint = "Retry with one of did_you_mean."
	}
	e.Hint = hint
	return e
}

func invalidArg(msg string, valid []string) *ToolError {
	e := errf(CodeInvalidArgument, "%s", msg)
	e.Valid = valid
	return e
}
