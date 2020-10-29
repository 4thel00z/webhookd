package debug

import (
	"webhookd/pkg/libwebhook"
	"webhookd/pkg/libwebhook/jwt"
)

type Debug struct{}

var (
	Module = Debug{}
)

func (Y Debug) Version() string {
	return "v1"
}

func (Y Debug) Namespace() string {
	return "debug"
}

func (Y Debug) Routes() map[string]libwebhook.Route {
	// Add route definitions here
	return map[string]libwebhook.Route{
		"routes": {
			Path:        "routes",
			Method:      "GET",
			CurlExample: "curl http://<addr>/<version>/<namespace>/routes",
			Service:     GetRoutesHandler,
		},
		"private": {
			Path:        "private",
			Method:      "GET",
			CurlExample: "curl http://<addr>/<version>/<namespace>/private",
			Service:     GetPrivateMessageHandler,
			TokenValidator: jwt.New(
				jwt.WithDebug(),
			).Middleware,
		},
	}
}

func (Y Debug) LongPath(route libwebhook.Route) string {
	return libwebhook.DefaultLongPath(Y, route)
}
