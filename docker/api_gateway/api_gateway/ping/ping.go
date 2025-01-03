package ping

import (
	"encoding/json"
	"net/http"

	"api_gateway/config"
)

var version = config.GetEnv("VERSION", "0.0.0")

func PingHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "version": version})
}
