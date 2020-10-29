package debug

import "webhookd/pkg/libwebhook"

type GetRoutesResponse struct {
	Routes map[string]libwebhook.Route `json:"routes"`
	Error  *string                     `json:"error,omitempty"`
}
