package webhook

import (
	"fmt"
	"github.com/monzo/typhon"
	uuid "github.com/nu7hatch/gouuid"
	"net/http"
	"time"
	"webhookd/pkg/libwebhook"
	"webhookd/pkg/libwebhook/filters"
)

func PostGenerateWebhookHandler(app *libwebhook.App) typhon.Service {
	return func(raw typhon.Request) typhon.Response {
		module := app.Modules[WebhookNamespace].(Webhook)
		req := raw.Value(filters.ValidationResult).(*PostGenerateWebhookRequest)
		v4, err := uuid.NewV4()
		if err != nil {
			errMsg := fmt.Sprintf("Could not generate random path: %s", err.Error())
			res := raw.Response(libwebhook.GenericResponse{
				Message: nil,
				Error:   &errMsg,
			})
			res.StatusCode = http.StatusInternalServerError
			return res
		}

		route := libwebhook.Route{
			Path:   v4.String(),
			Method: req.Method,
		}

		longPath := module.LongPath(route)
		route.Service = func(app *libwebhook.App) typhon.Service {
			return func(innerReq typhon.Request) typhon.Response {
				hook := module.Hooks[longPath]
				hook.Counter += 1
				module.Hooks[longPath] = hook
				response := innerReq.Response(req.Body)
				for k, v := range req.Headers {
					response.Header.Add(k, v)
				}
				return response
			}
		}

		module.Hooks[longPath] = RouteMetaInformation{
			Route:    route,
			Counter:  0,
			Active:   true,
			LastCall: time.Unix(0, 0),
		}

		app.Router.Register(req.Method, longPath, route.Service(app))

		res := raw.Response(PostGenerateWebhookResponse{
			Path:  longPath,
			Error: nil,
		})

		res.StatusCode = 200
		return res
	}
}

func PostUnregisterWebhookHandler(app *libwebhook.App) typhon.Service {
	return func(raw typhon.Request) typhon.Response {
		module := app.Modules[WebhookNamespace].(Webhook)
		req := raw.Value(filters.ValidationResult).(*PostUnregisterWebhookRequest)

		route := libwebhook.Route{
			Path:   req.UUID,
			Method: req.Method,
		}

		path := module.LongPath(route)

		_, _, _, ok := app.Router.Lookup(req.Method, path)
		if !ok {

			errMsg := fmt.Sprintf("Could not find the path: %s", path)
			res := raw.Response(libwebhook.GenericResponse{
				Message: nil,
				Error:   &errMsg,
			})
			res.StatusCode = http.StatusNotFound
			return res
		}

		hook, ok := module.Hooks[path]
		if !ok {
			fmt.Println(module.Hooks)
			errMsg := fmt.Sprintf("Could not find the path: %s", path)
			res := raw.Response(libwebhook.GenericResponse{
				Message: nil,
				Error:   &errMsg,
			})
			res.StatusCode = http.StatusNotFound
			return res
		}

		hook.Active = false
		module.Hooks[path] = hook

		res := raw.Response(libwebhook.GenericResponse{
			Message: fmt.Sprintf("Deactivated succesfully %s", path),
			Error:   nil,
		})

		res.StatusCode = http.StatusOK
		return res
	}
}
