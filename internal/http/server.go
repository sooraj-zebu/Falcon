package http

import (
	"fmt"
	"html/template"
	"log"
	nethttp "net/http"

	"github.com/sooraj-zebu/falcon/internal/service"
)

func StartServer(port int, edgeService *service.EdgeService, logger *log.Logger) error {

	mux := nethttp.NewServeMux()

	// API routes
	RegisterEdgeRoutes(mux, edgeService)

	// Static files
	mux.Handle(
		"/static/",
		nethttp.StripPrefix(
			"/static/",
			nethttp.FileServer(nethttp.Dir("web/static")),
		),
	)

	// Dashboard
	tmpl := template.Must(template.ParseFiles("web/templates/index.html"))

	mux.HandleFunc("/", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.URL.Path != "/" {
			nethttp.NotFound(w, r)
			return
		}

		if err := tmpl.Execute(w, nil); err != nil {
			nethttp.Error(w, err.Error(), nethttp.StatusInternalServerError)
		}
	})

	logger.Println("HTTP Server started on port", port)

	return nethttp.ListenAndServe(
		":"+fmt.Sprint(port),
		mux,
	)
}
