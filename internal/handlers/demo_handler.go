package handlers

import (
	"net/http"

	"github.com/tranmh/gassigeher/internal/config"
	"github.com/tranmh/gassigeher/internal/database"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
	"github.com/tranmh/gassigeher/internal/services"
)

// DemoHandler handles demo tenant API endpoints
type DemoHandler struct {
	db            *database.DB
	cfg           *config.Config
	tenantRepo    *repository.TenantRepository
	demoStateRepo *repository.DemoTenantRepository
}

// NewDemoHandler creates a new demo handler
func NewDemoHandler(db *database.DB, cfg *config.Config) *DemoHandler {
	return &DemoHandler{
		db:            db,
		cfg:           cfg,
		tenantRepo:    repository.NewTenantRepository(db),
		demoStateRepo: repository.NewDemoTenantRepository(db),
	}
}

// GetCredentials returns the demo admin credentials
// GET /api/demo/credentials
func (h *DemoHandler) GetCredentials(w http.ResponseWriter, r *http.Request) {
	// Get demo tenant
	tenant, err := h.tenantRepo.GetDemoTenant()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Demo-Daten")
		return
	}

	if tenant == nil {
		respondError(w, http.StatusNotFound, "Demo-Tenant nicht gefunden")
		return
	}

	// Get credentials
	credentials, err := h.demoStateRepo.GetCredentials(tenant.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Zugangsdaten")
		return
	}

	if credentials == nil {
		respondError(w, http.StatusNotFound, "Demo-Zugangsdaten nicht gefunden")
		return
	}

	// Override admin email with config-based email (database may have old hardcoded value)
	credentials.AdminEmail = h.cfg.DemoAdminEmail()

	// Build response with demo users
	response := struct {
		AdminCredentials models.DemoCredentials `json:"admin"`
		DemoUsers        []models.DemoUser      `json:"demo_users"`
		DemoDogs         []models.DemoDog       `json:"demo_dogs"`
	}{
		AdminCredentials: *credentials,
		DemoUsers: []models.DemoUser{
			{
				Name:     "Anna Gruen",
				Email:    h.cfg.DemoUserEmail("anna"),
				Password: services.DemoUserPassword,
				Level:    "green",
				LevelDE:  "Anfaenger",
			},
			{
				Name:     "Bernd Orange",
				Email:    h.cfg.DemoUserEmail("bernd"),
				Password: services.DemoUserPassword,
				Level:    "orange",
				LevelDE:  "Fortgeschritten",
			},
			{
				Name:     "Clara Blau",
				Email:    h.cfg.DemoUserEmail("clara"),
				Password: services.DemoUserPassword,
				Level:    "blue",
				LevelDE:  "Experte",
			},
		},
		DemoDogs: []models.DemoDog{
			{Name: "Bella", Breed: "Labrador Retriever", Category: "green"},
			{Name: "Max", Breed: "Golden Retriever", Category: "green"},
			{Name: "Luna", Breed: "Border Collie", Category: "orange"},
			{Name: "Rocky", Breed: "Deutscher Schaeferhund", Category: "orange"},
			{Name: "Duke", Breed: "Rottweiler", Category: "blue"},
		},
	}

	respondJSON(w, http.StatusOK, response)
}

// GetStatus returns the demo tenant status
// GET /api/demo/status
func (h *DemoHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	// Get demo tenant
	tenant, err := h.tenantRepo.GetDemoTenant()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden der Demo-Daten")
		return
	}

	if tenant == nil {
		// Return false if no demo tenant
		respondJSON(w, http.StatusOK, models.DemoStatus{IsDemo: false})
		return
	}

	// Get state for next reset time
	state, err := h.demoStateRepo.GetState(tenant.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Laden des Demo-Status")
		return
	}

	status := models.DemoStatus{
		IsDemo: true,
	}

	if state != nil && state.NextResetAt != nil {
		status.NextResetAt = state.NextResetAt.Format("02.01.2006 15:04")
	}

	respondJSON(w, http.StatusOK, status)
}
