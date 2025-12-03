package handlers
import "net/http"

type HealthHandler struct {

}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	data := map[string]string{"status": "ok"}
  	respondJSON(w, http.StatusOK, data)
}