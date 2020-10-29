package webhook

import (
	"time"
	"webhookd/pkg/libwebhook"
)

const (
	WebhookNamespace = "webhook"
)

type RouteMetaInformation struct {
	Route    libwebhook.Route
	Counter  int64
	Active   bool
	LastCall time.Time
}

type Webhook struct {
	Hooks map[string]RouteMetaInformation
}

var (
	Module = Webhook{
		Hooks: map[string]RouteMetaInformation{},
	}
)

func (w Webhook) Version() string {
	return "v1"
}

func (w Webhook) Namespace() string {
	return WebhookNamespace
}

func (w Webhook) Routes() map[string]libwebhook.Route {
	return map[string]libwebhook.Route{
		"generate": {
			Path:        "generate",
			Method:      "POST",
			CurlExample: "curl -X POST http://<addr>/<version>/<namespace>/generate",
			Service:     PostGenerateWebhookHandler,
			Validator:   libwebhook.GenerateRequestValidator(PostGenerateWebhookRequest{}),
		},
		"unregister": {
			Path:        "unregister",
			Method:      "POST",
			CurlExample: "curl -X POST http://<addr>/<version>/<namespace>/unregister",
			Service:     PostUnregisterWebhookHandler,
			Validator:   libwebhook.GenerateRequestValidator(PostUnregisterWebhookRequest{}),
		},
	}
}

func (w Webhook) LongPath(route libwebhook.Route) string {
	return libwebhook.DefaultLongPath(w, route)
}
