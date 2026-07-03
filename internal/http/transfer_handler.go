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

	mux.HandleFunc("/api/v1/transfers/", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodPost {
			w.WriteHeader(nethttp.StatusMethodNotAllowed)
			return
		}

		path := r.URL.Path
		prefix := "/api/v1/transfers/"
		suffix := "/progress"
		if len(path) <= len(prefix)+len(suffix) || path[len(path)-len(suffix):] != suffix {
			nethttp.NotFound(w, r)
			return
		}

		jobID := path[len(prefix) : len(path)-len(suffix)]
		var req struct {
			Status           string `json:"status"`
			CurrentFile      string `json:"current_file"`
			BytesTotal       int64  `json:"bytes_total"`
			BytesTransferred int64  `json:"bytes_transferred"`
			ErrorMessage     string `json:"error_message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			nethttp.Error(w, err.Error(), nethttp.StatusBadRequest)
			return
		}

		if err := transferService.UpdateJobStatus(
			jobID,
			req.Status,
			req.CurrentFile,
			req.BytesTotal,
			req.BytesTransferred,
			req.ErrorMessage,
		); err != nil {
			nethttp.Error(w, err.Error(), nethttp.StatusInternalServerError)
			return
		}

		writeJSON(w, map[string]string{"status": "ok"})
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

	mux.HandleFunc("/api/v1/transfer-workers/", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodDelete {
			w.WriteHeader(nethttp.StatusMethodNotAllowed)
			return
		}

		path := r.URL.Path
		prefix := "/api/v1/transfer-workers/"
		workerId := path[len(prefix):]

		if workerId == "" {
			nethttp.NotFound(w, r)
			return
		}

		if err := transferService.DeleteWorker(workerId); err != nil {
			nethttp.Error(w, err.Error(), nethttp.StatusInternalServerError)
			return
		}

		writeJSON(w, map[string]string{"status": "deleted"})
	})
}

func writeJSON(w nethttp.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
