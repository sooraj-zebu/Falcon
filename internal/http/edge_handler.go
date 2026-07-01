package http

import (
	"encoding/json"
	nethttp "net/http"

	"github.com/sooraj-zebu/falcon/internal/service"
)

func RegisterEdgeRoutes(
	mux *nethttp.ServeMux,
	edgeService *service.EdgeService,
) {

	mux.HandleFunc("/api/v1/edges", func(
		w nethttp.ResponseWriter,
		r *nethttp.Request,
	) {

		edges, err := edgeService.ListEdges()
		if err != nil {
			nethttp.Error(
				w,
				err.Error(),
				nethttp.StatusInternalServerError,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		json.NewEncoder(w).Encode(edges)
	})
}
