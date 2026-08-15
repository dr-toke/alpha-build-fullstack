package api

import "net/http"

// errorBody is `{ "error": "<message>" }` — the ACTUAL shape
// apps/web/src/lib/api/client.ts's apiFetch reads (`body?.error` as a
// plain string, then `new ApiError(status, message)`). 02-FRONTEND-CONTRACT.md
// §3 documents a richer `{ error: { code, message } }` envelope for stable
// machine codes, but the shipped ApiError class has no `.code` field at
// all — sending the object form would set ApiError's message to a
// stringified object instead of readable text. HTTP status itself (404 vs
// 503 vs 429 vs 400) is what the frontend actually branches on
// (`ApiError.status`); see API-DECISIONS.md for the full reasoning on why
// this deviates from the doc and matches the real client instead.
type errorBody struct {
	Error string `json:"error"`
}

// WriteError writes the error envelope the frontend actually parses.
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, errorBody{Error: message})
}

func writeNotFound(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusNotFound, message)
}

func writeInvalidFilter(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusBadRequest, message)
}

func writeUnavailable(w http.ResponseWriter, message string) {
	w.Header().Set("Retry-After", "5")
	WriteError(w, http.StatusServiceUnavailable, message)
}
