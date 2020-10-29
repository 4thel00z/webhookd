package libwebhook

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/monzo/typhon"
	"log"
	"strings"
)

var (
	routes = map[string]Route{}
)

type App struct {
	Addr    string            `json:"addr"`
	Config  Config            `json:"config"`
	Modules map[string]Module `json:"modules"`
	Router  *typhon.Router
	Debug   bool
	Verbose bool
}

func NewApp(addr string, config Config, verbose, debug bool, modules ...Module) App {

	m := map[string]Module{}
	for _, module := range modules {
		m[module.Namespace()] = module
	}

	app := App{
		Addr:    addr,
		Config:  config,
		Modules: m,
		Debug:   debug,
		Verbose: verbose,
	}

	app.Router = &typhon.Router{}

	for _, module := range modules {
		for _, route := range module.Routes() {
			app.Router.Register(strings.ToUpper(route.Method), module.LongPath(route), route.Service(&app))
		}
	}

	return app
}

func (app App) Routes() map[string]Route {
	if len(routes) > 0 {
		return routes
	}

	addr := app.Addr

	for _, module := range app.Modules {
		version := module.Version()
		namespace := module.Namespace()

		for _, route := range module.Routes() {
			route.CurlExample = strings.ReplaceAll(route.CurlExample, "<addr>", addr)
			route.CurlExample = strings.ReplaceAll(route.CurlExample, "<version>", version)
			route.CurlExample = strings.ReplaceAll(route.CurlExample, "<namespace>", namespace)
			route.CurlExample = strings.ReplaceAll(route.CurlExample, "<path>", route.Path)
			if app.Debug {
				// Add module wise injections of f.e. the <auth> tag
			}

			routes[module.LongPath(route)] = route
		}
	}
	return routes
}
func (app App) PrintRoutes(addr string) {
	routes := app.Routes()
	if len(routes) > 0 {
		log.Println("👠\tThe routes 🛣️  are:")
	}
	for path, route := range routes {
		log.Printf("\thttp://%v%s with method: %s", addr, path, route.Method)
		log.Printf("\tQuery this endpoint like this:\n\t\t%s", route.CurlExample)

	}
}

func (app App) Register(module Module) {
	for path, route := range module.Routes() {
		fmt.Println("HANDLER", path, route.Service)
		app.Router.Register(strings.ToUpper(route.Method), path, route.Service(&app))
	}

}

func (app App) PrintConfig() {
	fmt.Print("======================\t✈️\tConfig incoming\t✈️\t======================\n")
	spew.Dump(app.Config)
	fmt.Print("======================\t✈️ Config landed! ✈️\t======================\n")
}
