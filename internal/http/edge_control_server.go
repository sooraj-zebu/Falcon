package http

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	nethttp "net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	pb "github.com/sooraj-zebu/falcon/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type EdgeTransferRequest struct {
	JobID               string `json:"job_id"`
	SourcePath          string `json:"source_path"`
	DestinationPath     string `json:"destination_path"`
	DestinationHost     string `json:"destination_host"`
	DestinationHTTPPort int    `json:"destination_http_port"`
	DestinationGRPCPort int    `json:"destination_grpc_port"`
	CoreHost            string `json:"core_host"`
	CoreHTTPPort        int    `json:"core_http_port"`
	ChunkSize           int64  `json:"chunk_size"`
}

type EdgeTransferResponse struct {
	Status string `json:"status"`
}

func StartEdgeControlServer(
	port int,
	mountPath string,
	logger *log.Logger,
) error {
	mux := nethttp.NewServeMux()

	mux.HandleFunc("/api/v1/edge/health", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodGet {
			w.WriteHeader(nethttp.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, map[string]any{
			"status":     "running",
			"mount_path": mountPath,
		})
	})

	mux.HandleFunc("/api/v1/edge/file-status", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodGet {
			w.WriteHeader(nethttp.StatusMethodNotAllowed)
			return
		}

		path, err := resolveMountedPath(mountPath, r.URL.Query().Get("path"))
		if err != nil {
			nethttp.Error(w, err.Error(), nethttp.StatusBadRequest)
			return
		}

		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				writeJSON(w, map[string]any{"exists": false, "size": 0})
				return
			}
			nethttp.Error(w, err.Error(), nethttp.StatusInternalServerError)
			return
		}

		writeJSON(w, map[string]any{
			"exists": true,
			"size":   info.Size(),
		})
	})

	mux.HandleFunc("/api/v1/edge/transfers", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.Method != nethttp.MethodPost {
			w.WriteHeader(nethttp.StatusMethodNotAllowed)
			return
		}

		var req EdgeTransferRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			nethttp.Error(w, err.Error(), nethttp.StatusBadRequest)
			return
		}

		go func() {
			if err := runEdgeTransfer(mountPath, req); err != nil {
				logger.Println("edge transfer failed:", req.JobID, err)
				_ = reportTransferFailure(req, err)
			}
		}()

		writeJSON(w, EdgeTransferResponse{Status: "accepted"})
	})

	logger.Println("[Edge HTTP] control server running on", port)

	return nethttp.ListenAndServe(fmt.Sprintf(":%d", port), mux)
}

func runEdgeTransfer(mountPath string, req EdgeTransferRequest) error {
	if req.JobID == "" {
		return fmt.Errorf("job_id is required")
	}
	if req.ChunkSize <= 0 {
		req.ChunkSize = 64 * 1024 * 1024
	}

	sourcePath, err := resolveMountedPath(mountPath, req.SourcePath)
	if err != nil {
		return err
	}

	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()

	info, err := source.Stat()
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("directory edge-to-edge transfer is not implemented yet")
	}

	offset, err := destinationSize(req)
	if err != nil {
		return err
	}
	if offset > info.Size() {
		offset = 0
	}

	if _, err := source.Seek(offset, io.SeekStart); err != nil {
		return err
	}

	conn, err := grpc.Dial(
		fmt.Sprintf("%s:%d", req.DestinationHost, req.DestinationGRPCPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pb.NewTransferServiceClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.TransferFile(ctx)
	if err != nil {
		return err
	}

	if err := reportTransferProgress(req, sourcePath, info.Size(), offset); err != nil {
		return err
	}

	buffer := make([]byte, req.ChunkSize)
	transferred := offset
	for {
		n, readErr := source.Read(buffer)
		if n > 0 {
			sum := sha256.Sum256(buffer[:n])
			if err := stream.Send(&pb.FileChunk{
				TransferId: req.JobID,
				FileId:     req.DestinationPath,
				ChunkIndex: transferred,
				Data:       buffer[:n],
				Checksum:   hex.EncodeToString(sum[:]),
			}); err != nil {
				return err
			}

			transferred += int64(n)
			if err := reportTransferProgress(req, sourcePath, info.Size(), transferred); err != nil {
				return err
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}

	status, err := stream.CloseAndRecv()
	if err != nil {
		return err
	}
	if status.Status != "success" {
		return fmt.Errorf("destination returned %s", status.Status)
	}

	return reportTransferComplete(req, info.Size())
}

func destinationSize(req EdgeTransferRequest) (int64, error) {
	url := fmt.Sprintf(
		"http://%s:%d/api/v1/edge/file-status?path=%s",
		req.DestinationHost,
		req.DestinationHTTPPort,
		url.QueryEscape(req.DestinationPath),
	)

	client := nethttp.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("destination status check failed: %s", strings.TrimSpace(string(body)))
	}

	var payload struct {
		Size int64 `json:"size"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, err
	}

	return payload.Size, nil
}

func reportTransferProgress(req EdgeTransferRequest, currentFile string, total, transferred int64) error {
	return reportTransfer(req, "running", currentFile, total, transferred, "")
}

func reportTransferComplete(req EdgeTransferRequest, total int64) error {
	return reportTransfer(req, "completed", "", total, total, "")
}

func reportTransferFailure(req EdgeTransferRequest, cause error) error {
	message := ""
	if cause != nil {
		message = cause.Error()
	}
	return reportTransfer(req, "failed", "", 0, 0, message)
}

func reportTransfer(req EdgeTransferRequest, status, currentFile string, total, transferred int64, message string) error {
	if req.CoreHost == "" || req.CoreHTTPPort == 0 {
		return nil
	}

	url := fmt.Sprintf("http://%s:%d/api/v1/transfers/%s/progress", req.CoreHost, req.CoreHTTPPort, req.JobID)
	payload := map[string]any{
		"status":            status,
		"current_file":      currentFile,
		"bytes_total":       total,
		"bytes_transferred": transferred,
		"error_message":     message,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := nethttp.Post(url, "application/json", strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("core progress update failed: %s", strings.TrimSpace(string(data)))
	}

	return nil
}

func resolveMountedPath(mountPath, requestedPath string) (string, error) {
	if requestedPath == "" {
		return "", fmt.Errorf("path is required")
	}

	cleanPath := filepath.Clean(requestedPath)
	mount := filepath.Clean(mountPath)
	if mount == "" || mount == "." {
		return cleanPath, nil
	}

	if filepath.IsAbs(cleanPath) {
		if cleanPath == mount || strings.HasPrefix(cleanPath, mount+string(os.PathSeparator)) {
			return cleanPath, nil
		}
		return "", fmt.Errorf("path must be inside mount path %s", mount)
	}

	return filepath.Join(mount, cleanPath), nil
}
