package filters

import (
	"context"
	"fmt"
	"github.com/monzo/typhon"
	"webhookd/pkg/libwebhook"
)

const (
	ValidationResult = "validation_result"
)

func Validation(app libwebhook.App) typhon.Filter {
	return func(req typhon.Request, svc typhon.Service) typhon.Response {
		pattern := app.Router.Pattern(req)
		routes := app.Routes()
		route, ok := routes[pattern]
		if !ok {
			return svc(req)
		}

		if route.Validator == nil {
			return svc(req)
		}

		val, err := (*route.Validator)(req)

		if err != nil {
			msg := err.Error()
			return req.Response(libwebhook.GenericResponse{
				Message: fmt.Sprintf("[%s] %s validation error", pattern, route.Method),
				Error:   &msg,
			})
		}

		req.Context = context.WithValue(req.Context, ValidationResult, val)
		return svc(req)

	}
}
