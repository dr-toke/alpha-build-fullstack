package api

import "net/http"

// Error codes — 02-FRONTEND-CONTRACT.md §3's exact stable set, so the
// frontend's ApiError can branch on `error.code` rather than parsing
// `error.message`. All nine are declared here even though this build's
// handlers only ever emit a few of them — the vocabulary is the contract,
// not just whatever this milestone happens to produce.
const (
	CodeNotFound         = "not_found"
	CodeMoved            = "moved"
	CodeRateLimited      = "rate_limited"
	CodeUnavailable      = "unavailable"
	CodeInvalidFilter    = "invalid_filter"
	CodeAuthRequired     = "auth_required"
	CodeAuthInvalid      = "auth_invalid"
	CodeBanned           = "banned"
	CodeValidationFailed = "validation_failed"
)

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// WriteError writes 02-FRONTEND-CONTRACT.md §3's error envelope:
// `{ "error": { "code", "message" } }`.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	var body errorBody
	body.Error.Code = code
	body.Error.Message = message
	WriteJSON(w, status, body)
}

func writeNotFound(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusNotFound, CodeNotFound, message)
}

func writeInvalidFilter(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusBadRequest, CodeInvalidFilter, message)
}

func writeUnavailable(w http.ResponseWriter, message string) {
	w.Header().Set("Retry-After", "5")
	WriteError(w, http.StatusServiceUnavailable, CodeUnavailable, message)
}
