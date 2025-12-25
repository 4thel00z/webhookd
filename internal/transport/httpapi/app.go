package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humafiber"
	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"

	"webhookd/internal/application/webhooks"
	"webhookd/internal/domain/webhook"
	"webhookd/internal/infrastructure/auth/jwtmiddleware"
	"webhookd/internal/infrastructure/configfile"
)

type Deps struct {
	Version  string
	Config   configfile.Config
	Webhooks *webhooks.Service
}

type fiberCtxKey struct{}

func NewApp(d Deps) (*fiber.App, error) {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	// Request spans are no-ops unless a TracerProvider is configured (see internal/observability).
	app.Use(otelfiber.Middleware())

	// Non-API health check.
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	api := humafiber.New(app, huma.DefaultConfig("webhookd", d.Version))

	// Make the underlying *fiber.Ctx available to handlers (they only receive
	// context.Context), so we can set dynamic response headers for hooks.
	api.UseMiddleware(func(hctx huma.Context, next func(huma.Context)) {
		fc := humafiber.Unwrap(hctx)
		next(huma.WithValue(hctx, fiberCtxKey{}, fc))
	})

	auth := jwtmiddleware.New(jwtmiddleware.Config{
		EnableAuthOnOptions: d.Config.EnableAuthOnOptions,
		TokenExtractors:     d.Config.TokenExtractors,
		JWKSURL:             d.Config.OAuthJsonWebKeySetsURL,
		Issuer:              d.Config.OAuthIssuer,
		Audience:            d.Config.OAuthAudience,
	}).Huma(api)

	// Debug: list known routes + hooks.
	huma.Get(api, "/v1/debug/routes", func(ctx context.Context, _ *struct{}) (*struct {
		Body struct {
			Routes []struct {
				Method string `json:"method"`
				Path   string `json:"path"`
			} `json:"routes"`
			Hooks map[string]struct {
				ID      string            `json:"id"`
				Method  string            `json:"method"`
				Active  bool              `json:"active"`
				Counter int64             `json:"counter"`
				Headers map[string]string `json:"headers"`
			} `json:"hooks"`
		}
	}, error) {
		resp := &struct {
			Body struct {
				Routes []struct {
					Method string `json:"method"`
					Path   string `json:"path"`
				} `json:"routes"`
				Hooks map[string]struct {
					ID      string            `json:"id"`
					Method  string            `json:"method"`
					Active  bool              `json:"active"`
					Counter int64             `json:"counter"`
					Headers map[string]string `json:"headers"`
				} `json:"hooks"`
			}
		}{}

		resp.Body.Routes = []struct {
			Method string `json:"method"`
			Path   string `json:"path"`
		}{
			{Method: http.MethodPost, Path: "/v1/webhooks"},
			{Method: http.MethodDelete, Path: "/v1/webhooks/{id}"},
			{Method: http.MethodGet, Path: "/v1/hooks/{id}"},
			{Method: http.MethodPost, Path: "/v1/hooks/{id}"},
			{Method: http.MethodPut, Path: "/v1/hooks/{id}"},
			{Method: http.MethodPatch, Path: "/v1/hooks/{id}"},
			{Method: http.MethodDelete, Path: "/v1/hooks/{id}"},
			{Method: http.MethodOptions, Path: "/v1/hooks/{id}"},
			{Method: http.MethodGet, Path: "/v1/debug/routes"},
			{Method: http.MethodGet, Path: "/v1/debug/private"},
		}

		hooksMap, err := d.Webhooks.List(ctx)
		if err != nil {
			return nil, err
		}
		resp.Body.Hooks = make(map[string]struct {
			ID      string            `json:"id"`
			Method  string            `json:"method"`
			Active  bool              `json:"active"`
			Counter int64             `json:"counter"`
			Headers map[string]string `json:"headers"`
		}, len(hooksMap))
		for id, h := range hooksMap {
			resp.Body.Hooks[string(id)] = struct {
				ID      string            `json:"id"`
				Method  string            `json:"method"`
				Active  bool              `json:"active"`
				Counter int64             `json:"counter"`
				Headers map[string]string `json:"headers"`
			}{
				ID:      string(h.ID),
				Method:  h.Method,
				Active:  h.Active,
				Counter: h.Counter,
				Headers: h.Headers,
			}
		}

		return resp, nil
	})

	// Debug: private route (auth added in auth-middleware todo).
	huma.Get(api, "/v1/debug/private", func(ctx context.Context, _ *struct{}) (*struct {
		Body struct {
			Message string `json:"message"`
		}
	}, error) {
		resp := &struct {
			Body struct {
				Message string `json:"message"`
			}
		}{}
		tok, _ := ctx.Value(jwtmiddleware.TokenContextKey{}).(jwtmiddleware.TokenInfo)
		resp.Body.Message = "This is my token: " + tok.Raw + "!"
		return resp, nil
	}, func(o *huma.Operation) {
		o.Middlewares = append(o.Middlewares, auth)
	})

	// Webhook management: create
	huma.Post(api, "/v1/webhooks", func(ctx context.Context, input *struct {
		Body struct {
			Method  string            `json:"method" doc:"HTTP method for invoking the webhook" example:"GET"`
			Body    string            `json:"body" doc:"JSON string body returned by the webhook" example:"hello"`
			Headers map[string]string `json:"headers" doc:"Headers to include in webhook responses"`
		}
	}) (*struct {
		Body struct {
			ID   string `json:"id"`
			Path string `json:"path"`
		}
	}, error) {
		h, err := d.Webhooks.Create(ctx, webhooks.CreateParams{
			Method:  input.Body.Method,
			Body:    input.Body.Body,
			Headers: input.Body.Headers,
		})
		if err != nil {
			return nil, err
		}
		resp := &struct {
			Body struct {
				ID   string `json:"id"`
				Path string `json:"path"`
			}
		}{}
		resp.Body.ID = string(h.ID)
		resp.Body.Path = "/v1/hooks/" + string(h.ID)
		return resp, nil
	})

	// Webhook management: deactivate
	huma.Delete(api, "/v1/webhooks/{id}", func(ctx context.Context, input *struct {
		ID string `path:"id" doc:"Webhook id"`
	}) (*struct {
		Body struct {
			Message string `json:"message"`
			ID      string `json:"id"`
		}
	}, error) {
		_, ok, err := d.Webhooks.Deactivate(ctx, webhook.ID(input.ID))
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, huma.Error404NotFound("not found")
		}
		resp := &struct {
			Body struct {
				Message string `json:"message"`
				ID      string `json:"id"`
			}
		}{}
		resp.Body.Message = "deactivated"
		resp.Body.ID = input.ID
		return resp, nil
	})

	// Webhook execution for common methods.
	registerHookInvoke(api, d, http.MethodGet)
	registerHookInvoke(api, d, http.MethodPost)
	registerHookInvoke(api, d, http.MethodPut)
	registerHookInvoke(api, d, http.MethodPatch)
	registerHookInvoke(api, d, http.MethodDelete)
	registerHookInvoke(api, d, http.MethodOptions)

	_ = d.Config // will be used in auth todo

	return app, nil
}

func registerHookInvoke(api huma.API, d Deps, method string) {
	huma.Register(api, huma.Operation{
		OperationID: "invoke-hook-" + strings.ToLower(method),
		Method:      method,
		Path:        "/v1/hooks/{id}",
		Summary:     "Invoke a webhook",
		Errors:      []int{404, 405},
	}, func(ctx context.Context, input *struct {
		ID string `path:"id" doc:"Webhook id"`
	}) (*struct {
		Body string
	}, error) {
		h, ok, err := d.Webhooks.Get(ctx, webhook.ID(input.ID))
		if err != nil {
			return nil, err
		}
		if !ok || !h.Active {
			return nil, huma.Error404NotFound("not found")
		}
		if !h.MatchesMethod(method) {
			return nil, huma.Error405MethodNotAllowed("method not allowed")
		}

		h, _, err = d.Webhooks.Touch(ctx, webhook.ID(input.ID))
		if err != nil {
			return nil, err
		}

		fc, _ := ctx.Value(fiberCtxKey{}).(*fiber.Ctx)
		for k, v := range h.Headers {
			if strings.EqualFold(k, "Content-Length") {
				continue
			}
			if fc != nil {
				fc.Set(k, v)
			}
		}

		resp := &struct {
			Body string
		}{}
		resp.Body = h.Body
		return resp, nil
	})
}
