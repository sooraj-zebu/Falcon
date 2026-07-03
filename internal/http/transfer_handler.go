package http

import (
	"encoding/json"
	nethttp "net/http"

	"github.com/sooraj-zebu/falcon/internal/service"
)

func RegisterTransferRoutes(
	mux *nethttp.ServeMux,
	transferService *service.TransferService,
) {
	mux.HandleFunc("/api/v1/transfers", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		switch r.Method {
		case nethttp.MethodGet:
			jobs, err := transferService.ListJobs(100)
			if err != nil {
				nethttp.Error(w, err.Error(), nethttp.StatusInternalServerError)
				return
			}
			writeJSON(w, jobs)
		case nethttp.MethodPost:
			var req service.CreateTransferRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				nethttp.Error(w, err.Error(), nethttp.StatusBadRequest)
				return
			}

			job, err := transferService.CreateJob(req)
			if err != nil {
				nethttp.Error(w, err.Error(), nethttp.StatusBadRequest)
				return
			}

			writeJSON(w, job)
		default:
			w.WriteHeader(nethttp.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/transfer-workers", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		switch r.Method {
		case nethttp.MethodGet:
			workers, err := transferService.ListWorkers()
			if err != nil {
				nethttp.Error(w, err.Error(), nethttp.StatusInternalServerError)
				return
			}
			writeJSON(w, workers)
		case nethttp.MethodPost:
			var req struct {
				Name string `json:"name"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				nethttp.Error(w, err.Error(), nethttp.StatusBadRequest)
				return
			}

			worker, err := transferService.CreateWorker(req.Name)
			if err != nil {
				nethttp.Error(w, err.Error(), nethttp.StatusBadRequest)
				return
			}

			writeJSON(w, worker)
		default:
			w.WriteHeader(nethttp.StatusMethodNotAllowed)
		}
	})
}

func writeJSON(w nethttp.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
