package webhook

import (
	"fmt"
	"github.com/monzo/typhon"
	"net/http"
	"webhookd/pkg/libwebhook"
)

func Validation(app libwebhook.App) typhon.Filter {
	// FIXME: if webhook module name changes, this breaks

	return func(req typhon.Request, svc typhon.Service) typhon.Response {
		module := app.Modules[WebhookNamespace].(Webhook)
		pattern := app.Router.Pattern(req)
		route, ok := module.Hooks[pattern]

		if ok && route.Active || !ok {
			return svc(req)
		}

		msg := fmt.Sprintf("Not found route %s", pattern)
		response := req.Response(libwebhook.GenericResponse{
			Message: "",
			Error:   &msg,
		})
		response.StatusCode = http.StatusNotFound
		return response

	}
}
