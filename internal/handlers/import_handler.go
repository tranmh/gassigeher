package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tranmh/gassigeher/internal/database"
	"github.com/tranmh/gassigeher/internal/middleware"
	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/repository"
)

// ImportHandler handles data import requests
type ImportHandler struct {
	db           *database.DB
	dogRepo      *repository.DogRepository
	colorRepo    *repository.ColorCategoryRepository
	settingsRepo *repository.SettingsRepository
}

// NewImportHandler creates a new import handler
func NewImportHandler(db *database.DB) *ImportHandler {
	return &ImportHandler{
		db:           db,
		settingsRepo: repository.NewSettingsRepository(db),
		dogRepo:      repository.NewDogRepository(db),
		colorRepo: repository.NewColorCategoryRepository(db),
	}
}

// ImportPreview represents a preview of data to import
type ImportPreview struct {
	Headers     []string          `json:"headers"`
	SampleRows  [][]string        `json:"sample_rows"` // First 5 rows
	TotalRows   int               `json:"total_rows"`
	Mapping     map[string]string `json:"mapping,omitempty"` // column index -> field name
	Suggestions map[string]string `json:"suggestions"`       // auto-detected mappings
}

// ImportResult represents the result of an import
type ImportResult struct {
	TotalRows int      `json:"total_rows"`
	Imported  int      `json:"imported"`
	Skipped   int      `json:"skipped"`
	Errors    []string `json:"errors"`
}

// FieldMapping maps CSV columns to dog fields
type FieldMapping struct {
	Name                string `json:"name"`                           // Column index for name
	Breed               string `json:"breed,omitempty"`                // Column index for breed
	Age                 string `json:"age,omitempty"`                  // Column index for age
	Size                string `json:"size,omitempty"`                 // Column index for size (small/medium/large)
	Color               string `json:"color,omitempty"`                // Column index for color category
	SpecialInstructions string `json:"special_instructions,omitempty"` // Column index for special instructions
	PickupLocation      string `json:"pickup_location,omitempty"`      // Column index for pickup location
	IsAvailable         string `json:"is_available,omitempty"`         // Column index for availability
}

// PreviewImport parses an uploaded CSV/Excel file and returns a preview
// POST /api/v1/admin/import/dogs/preview
func (h *ImportHandler) PreviewImport(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	// SECURITY: Limit request body size to prevent DoS attacks
	const maxSize = 10 << 20 // 10MB
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)

	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(maxSize); err != nil {
		respondError(w, http.StatusBadRequest, "Datei zu gross (max 10MB)")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "Keine Datei hochgeladen")
		return
	}
	defer file.Close()

	// Check file extension
	filename := strings.ToLower(header.Filename)
	if !strings.HasSuffix(filename, ".csv") {
		respondError(w, http.StatusBadRequest, "Nur CSV-Dateien werden unterstützt (.csv)")
		return
	}

	// Parse CSV
	reader := csv.NewReader(file)
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	// Try to detect separator (comma, semicolon, tab)
	firstLine, err := reader.Read()
	if err != nil {
		// Try with semicolon
		if _, seekErr := file.Seek(0, 0); seekErr != nil {
			respondError(w, http.StatusInternalServerError, "Fehler beim Lesen der Datei")
			return
		}
		reader = csv.NewReader(file)
		reader.Comma = ';'
		firstLine, err = reader.Read()
		if err != nil {
			respondError(w, http.StatusBadRequest, "CSV konnte nicht gelesen werden")
			return
		}
	}

	// If only one column, try semicolon
	if len(firstLine) == 1 && strings.Contains(firstLine[0], ";") {
		if _, seekErr := file.Seek(0, 0); seekErr != nil {
			respondError(w, http.StatusInternalServerError, "Fehler beim Lesen der Datei")
			return
		}
		reader = csv.NewReader(file)
		reader.Comma = ';'
		firstLine, _ = reader.Read()
	}

	headers := firstLine

	// Read sample rows
	var sampleRows [][]string
	totalRows := 0
	for i := 0; i < 100; i++ { // Max 100 rows for preview
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		totalRows++
		if len(sampleRows) < 5 {
			sampleRows = append(sampleRows, row)
		}
	}

	// Count remaining rows
	for {
		_, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err == nil {
			totalRows++
		}
	}

	// Auto-detect field mappings
	suggestions := h.suggestMappings(headers, tenantID)

	respondJSON(w, http.StatusOK, ImportPreview{
		Headers:     headers,
		SampleRows:  sampleRows,
		TotalRows:   totalRows,
		Suggestions: suggestions,
	})
}

// suggestMappings auto-detects field mappings based on column headers
// First match wins - later matches do not overwrite earlier suggestions
func (h *ImportHandler) suggestMappings(headers []string, tenantID int) map[string]string {
	suggestions := make(map[string]string)

	// Common column name variations (German and English)
	namePatterns := []string{"name", "bezeichnung", "hundename", "tier"}
	breedPatterns := []string{"rasse", "breed", "art"}
	agePatterns := []string{"alter", "age", "geboren", "birth"}
	sizePatterns := []string{"groesse", "größe", "size", "gewicht"}
	colorPatterns := []string{"farbe", "color", "kategorie", "category"}
	instructionPatterns := []string{"hinweis", "special", "anmerkung", "notiz", "note"}
	locationPatterns := []string{"standort", "location", "ort", "platz", "abhol"}
	availablePatterns := []string{"verfügbar", "available", "status", "aktiv"}

	for i, header := range headers {
		headerLower := strings.ToLower(strings.TrimSpace(header))
		idx := strconv.Itoa(i)

		// Check each pattern - first match wins (don't overwrite existing suggestions)
		if _, exists := suggestions["name"]; !exists {
			for _, p := range namePatterns {
				if strings.Contains(headerLower, p) {
					suggestions["name"] = idx
					break
				}
			}
		}
		if _, exists := suggestions["breed"]; !exists {
			for _, p := range breedPatterns {
				if strings.Contains(headerLower, p) {
					suggestions["breed"] = idx
					break
				}
			}
		}
		if _, exists := suggestions["age"]; !exists {
			for _, p := range agePatterns {
				if strings.Contains(headerLower, p) {
					suggestions["age"] = idx
					break
				}
			}
		}
		if _, exists := suggestions["size"]; !exists {
			for _, p := range sizePatterns {
				if strings.Contains(headerLower, p) {
					suggestions["size"] = idx
					break
				}
			}
		}
		if _, exists := suggestions["color"]; !exists {
			for _, p := range colorPatterns {
				if strings.Contains(headerLower, p) {
					suggestions["color"] = idx
					break
				}
			}
		}
		if _, exists := suggestions["special_instructions"]; !exists {
			for _, p := range instructionPatterns {
				if strings.Contains(headerLower, p) {
					suggestions["special_instructions"] = idx
					break
				}
			}
		}
		if _, exists := suggestions["pickup_location"]; !exists {
			for _, p := range locationPatterns {
				if strings.Contains(headerLower, p) {
					suggestions["pickup_location"] = idx
					break
				}
			}
		}
		if _, exists := suggestions["is_available"]; !exists {
			for _, p := range availablePatterns {
				if strings.Contains(headerLower, p) {
					suggestions["is_available"] = idx
					break
				}
			}
		}
	}

	return suggestions
}

// ExecuteImport imports dogs from CSV with the provided field mapping
// POST /api/v1/admin/import/dogs
func (h *ImportHandler) ExecuteImport(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := r.Context().Value(middleware.TenantIDKey).(int)

	// SECURITY: Limit request body size to prevent DoS attacks
	const maxSize = 10 << 20 // 10MB
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)

	// Parse multipart form
	if err := r.ParseMultipartForm(maxSize); err != nil {
		respondError(w, http.StatusBadRequest, "Datei zu gross (max 10MB)")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "Keine Datei hochgeladen")
		return
	}
	defer file.Close()

	// Get field mapping from form
	mappingJSON := r.FormValue("mapping")
	var mapping FieldMapping
	if err := json.Unmarshal([]byte(mappingJSON), &mapping); err != nil {
		respondError(w, http.StatusBadRequest, "Ungültiges Mapping")
		return
	}

	// Validate required mapping
	if mapping.Name == "" {
		respondError(w, http.StatusBadRequest, "Name-Spalte muss angegeben werden")
		return
	}

	// Get default color category (configurable via system_settings)
	defaultID := 1 // Fallback: color category ID 1
	if colorSetting, err := h.settingsRepo.Get(tenantID, "default_color_for_new_users"); err == nil && colorSetting != nil && colorSetting.Value != "" {
		if parsed, err := strconv.Atoi(colorSetting.Value); err == nil && parsed > 0 {
			defaultID = parsed
		}
	}
	var defaultColorID *int
	defaultColorID = &defaultID
	colors, _ := h.colorRepo.FindAll(tenantID)
	colorMap := make(map[string]int)
	for _, c := range colors {
		colorMap[strings.ToLower(c.Name)] = c.ID
	}

	// Parse CSV
	reader := csv.NewReader(file)
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	// Try to detect separator
	firstLine, err := reader.Read()
	if err != nil {
		respondError(w, http.StatusBadRequest, "CSV konnte nicht gelesen werden")
		return
	}

	// If only one column, try semicolon
	if len(firstLine) == 1 && strings.Contains(firstLine[0], ";") {
		if _, seekErr := file.Seek(0, 0); seekErr != nil {
			respondError(w, http.StatusInternalServerError, "Fehler beim Lesen der Datei")
			return
		}
		reader = csv.NewReader(file)
		reader.Comma = ';'
		reader.Read() // Skip header
	}

	// Start transaction for atomic import
	tx, err := h.db.Begin()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Datenbankfehler beim Starten der Transaktion")
		return
	}
	defer tx.Rollback() // Will be no-op if commit succeeds

	// Create a transaction-aware dog repository
	dogRepoTx := repository.NewDogRepositoryWithTx(h.db, tx)

	// Import rows
	result := ImportResult{
		Errors: []string{},
	}

	rowNum := 1 // Start after header
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		rowNum++
		result.TotalRows++

		if err != nil {
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("Zeile %d: Lesefehler", rowNum))
			continue
		}

		// Extract values
		name := getColumnValue(row, mapping.Name)
		if name == "" {
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("Zeile %d: Name fehlt", rowNum))
			continue
		}

		dog := &models.Dog{
			TenantID:    tenantID,
			Name:        name,
			Breed:       getColumnValue(row, mapping.Breed),
			ColorID:     defaultColorID,
			IsAvailable: true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// Parse optional fields
		if mapping.Age != "" {
			if ageStr := getColumnValue(row, mapping.Age); ageStr != "" {
				if age, err := parseAge(ageStr); err == nil {
					dog.Age = age
				}
			}
		}

		if mapping.Size != "" {
			if size := parseSize(getColumnValue(row, mapping.Size)); size != "" {
				dog.Size = size
			}
		}

		if mapping.Color != "" {
			if colorName := getColumnValue(row, mapping.Color); colorName != "" {
				if colorID, ok := colorMap[strings.ToLower(colorName)]; ok {
					dog.ColorID = &colorID
				}
			}
		}

		if mapping.SpecialInstructions != "" {
			if instr := getColumnValue(row, mapping.SpecialInstructions); instr != "" {
				dog.SpecialInstructions = &instr
			}
		}

		if mapping.PickupLocation != "" {
			if loc := getColumnValue(row, mapping.PickupLocation); loc != "" {
				dog.PickupLocation = &loc
			}
		}

		if mapping.IsAvailable != "" {
			dog.IsAvailable = parseAvailable(getColumnValue(row, mapping.IsAvailable))
		}

		// Create dog within transaction
		if err := dogRepoTx.CreateTx(tx, dog); err != nil {
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("Zeile %d: %s", rowNum, err.Error()))
			continue
		}

		result.Imported++
	}

	// Commit the transaction if any dogs were imported
	if result.Imported > 0 {
		if err := tx.Commit(); err != nil {
			respondError(w, http.StatusInternalServerError, "Fehler beim Speichern der importierten Hunde")
			return
		}
	}

	respondJSON(w, http.StatusOK, result)
}

// Helper functions

func getColumnValue(row []string, colIndex string) string {
	if colIndex == "" {
		return ""
	}
	idx, err := strconv.Atoi(colIndex)
	if err != nil || idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func parseSize(size string) string {
	s := strings.ToLower(strings.TrimSpace(size))
	switch {
	case strings.Contains(s, "klein") || strings.Contains(s, "small") || s == "s":
		return "small"
	case strings.Contains(s, "mittel") || strings.Contains(s, "medium") || s == "m":
		return "medium"
	case strings.Contains(s, "gross") || strings.Contains(s, "groß") || strings.Contains(s, "large") || s == "l":
		return "large"
	}
	return ""
}

func parseAge(ageStr string) (int, error) {
	s := strings.TrimSpace(ageStr)
	if s == "" {
		return 0, fmt.Errorf("leeres Alter")
	}

	// Must start with a digit (rejects negative ages like "-3" and garbage like "a1b2")
	if len(s) == 0 || s[0] < '0' || s[0] > '9' {
		return 0, fmt.Errorf("ungültiges Alter: muss mit einer Zahl beginnen")
	}

	// Extract numbers from strings like "3 Jahre" or "5 years"
	numStr := ""
	for _, r := range s {
		if r >= '0' && r <= '9' {
			numStr += string(r)
		} else if numStr != "" {
			break // Stop at first non-digit after digits
		}
	}

	if numStr == "" {
		return 0, fmt.Errorf("keine Zahl gefunden")
	}

	age, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, err
	}

	// Validate reasonable age for a dog (max 30 years)
	if age > 30 {
		return 0, fmt.Errorf("unrealistisches Alter: %d Jahre (max 30)", age)
	}

	return age, nil
}

func parseAvailable(val string) bool {
	v := strings.ToLower(strings.TrimSpace(val))
	switch v {
	case "ja", "yes", "true", "1", "aktiv", "active", "verfügbar", "available":
		return true
	case "nein", "no", "false", "0", "inaktiv", "inactive":
		return false
	}
	return true // Default to available
}

// GetImportTemplate returns a sample CSV template
// GET /api/v1/admin/import/dogs/template
func (h *ImportHandler) GetImportTemplate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=hunde_import_vorlage.csv")

	writer := csv.NewWriter(w)
	writer.Comma = ';'

	// Header
	writer.Write([]string{"Name", "Rasse", "Alter", "Größe", "Farbe", "Spezialhinweise", "Abholort", "Verfügbar"})

	// Sample rows
	writer.Write([]string{"Bella", "Labrador", "3 Jahre", "groß", "Grün", "Verträgt sich gut mit Kindern", "Hauptgebäude", "ja"})
	writer.Write([]string{"Max", "Schäferhund-Mix", "5 Jahre", "groß", "Orange", "Braucht erfahrenen Halter", "Nebengebäude", "ja"})
	writer.Write([]string{"Luna", "Dackel", "2 Jahre", "klein", "Grün", "", "Hauptgebäude", "ja"})

	writer.Flush()
}
