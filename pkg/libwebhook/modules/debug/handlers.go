package debug

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/monzo/typhon"
	"webhookd/pkg/libwebhook"
	libjwt "webhookd/pkg/libwebhook/jwt"
)

func GetRoutesHandler(app *libwebhook.App) typhon.Service {
	return func(req typhon.Request) typhon.Response {

		response := req.Response(&GetRoutesResponse{
			Routes: app.Routes(),
		})

		response.StatusCode = 200
		return response
	}
}

func GetPrivateMessageHandler(app *libwebhook.App) typhon.Service {
	return func(req typhon.Request) typhon.Response {
		token := req.Value(libjwt.DefaultUserProperty).(*jwt.Token)
		response := req.Response(&libwebhook.GenericResponse{
			Message: fmt.Sprintf("This is my token: %s!", token.Raw),
		})
		response.StatusCode = 200
		return response
	}
}
