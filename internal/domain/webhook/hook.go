package webhook

import (
	"errors"
	"net/http"
	"strings"
	"time"
)

type ID string

type Hook struct {
	ID      ID
	Method  string
	Body    string
	Headers map[string]string

	Active   bool
	Counter  int64
	LastCall time.Time
	Created  time.Time
}

func New(id ID, method, body string, headers map[string]string, now time.Time) (*Hook, error) {
	m := normalizeMethod(method)
	if !isAllowedMethod(m) {
		return nil, errors.New("unsupported method")
	}

	h := &Hook{
		ID:       id,
		Method:   m,
		Body:     body,
		Headers:  cloneHeaders(headers),
		Active:   true,
		Counter:  0,
		LastCall: time.Unix(0, 0).UTC(),
		Created:  now.UTC(),
	}
	return h, nil
}

func (h *Hook) Deactivate() {
	h.Active = false
}

func (h *Hook) Touch(now time.Time) {
	h.Counter++
	h.LastCall = now.UTC()
}

func (h *Hook) MatchesMethod(method string) bool {
	return strings.EqualFold(h.Method, normalizeMethod(method))
}

func normalizeMethod(m string) string {
	m = strings.ToUpper(strings.TrimSpace(m))
	if m == "" {
		return http.MethodGet
	}
	return m
}

func isAllowedMethod(m string) bool {
	switch m {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions:
		return true
	default:
		return false
	}
}

func cloneHeaders(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
