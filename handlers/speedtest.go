package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"

	"apisql/database"
	"apisql/models"
	"apisql/services"
	"apisql/utils"

	"github.com/gin-gonic/gin"
)

// SpeedtestHandler holds dependencies for speedtest endpoints.
type SpeedtestHandler struct{}

// NewSpeedtestHandler creates a new SpeedtestHandler.
func NewSpeedtestHandler() *SpeedtestHandler {
	return &SpeedtestHandler{}
}

// Create saves a new speed test result.
// POST /api/v1/speedtests
// This endpoint is anonymous — no authentication required.
func (h *SpeedtestHandler) Create(c *gin.Context) {
	var req models.SpeedtestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, err.Error())
		return
	}

	// Validate coordinates
	if err := utils.ValidateCoordinates(req.Latitude, req.Longitude); err != nil {
		utils.ValidationError(c, err.Error())
		return
	}

	// Get client IP and hash it (never store plaintext)
	clientIP := c.ClientIP()
	ipHash := services.HashIP(clientIP)

	// Detect ISP automatically from client IP
	ispName := services.DetectISP(clientIP)

	// Find or create the provider
	providerID, err := findOrCreateProvider(ispName)
	if err != nil {
		log.Printf("Error finding/creating provider: %v", err)
		utils.InternalError(c, "")
		return
	}

	// Sanitize visitor_id
	visitorID := utils.SanitizeString(req.VisitorID)

	var testID int64
	err = database.DB.QueryRow(
		database.Rebind(`INSERT INTO speedtests (provider_id, zone_id, download_mbps, upload_mbps, ping_ms, latitude, longitude, ip_hash, visitor_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`),
		providerID, req.ZoneID, req.DownloadMbps, req.UploadMbps, req.PingMs,
		req.Latitude, req.Longitude, ipHash, nilIfEmpty(visitorID),
	).Scan(&testID)
	if err != nil {
		log.Printf("Error inserting speedtest: %v", err)
		utils.InternalError(c, "")
		return
	}

	// Calculate quality and score
	quality := services.QualityFromMetrics(req.DownloadMbps, req.UploadMbps, req.PingMs)
	score := services.CalculateScore(req.DownloadMbps, req.UploadMbps, req.PingMs)

	response := models.SpeedtestResponse{
		ID:           uint64(testID),
		Provider:     ispName,
		DownloadMbps: req.DownloadMbps,
		UploadMbps:   req.UploadMbps,
		PingMs:       req.PingMs,
		Latitude:     req.Latitude,
		Longitude:    req.Longitude,
		Quality:      quality,
		Score:        score,
	}

	utils.Success(c, http.StatusCreated, "Speedtest guardado exitosamente", response)
}

// List returns speedtests with optional filters and pagination.
// GET /api/v1/speedtests
func (h *SpeedtestHandler) List(c *gin.Context) {
	var filter models.SpeedtestFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		utils.ValidationError(c, err.Error())
		return
	}

	page, pageSize := utils.ValidatePagination(filter.Page, filter.PageSize)
	offset := utils.PaginationOffset(page, pageSize)

	// Build dynamic query
	query := `
		SELECT s.id, s.provider_id, p.nombre, s.download_mbps, s.upload_mbps, 
		       s.ping_ms, s.latitude, s.longitude, s.created_at
		FROM speedtests s
		JOIN providers p ON s.provider_id = p.id
		WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM speedtests s WHERE 1=1`
	args := []interface{}{}
	countArgs := []interface{}{}

	if filter.ProviderID != nil {
		query += " AND s.provider_id = ?"
		countQuery += " AND s.provider_id = ?"
		args = append(args, *filter.ProviderID)
		countArgs = append(countArgs, *filter.ProviderID)
	}
	if filter.ZoneID != nil {
		query += " AND s.zone_id = ?"
		countQuery += " AND s.zone_id = ?"
		args = append(args, *filter.ZoneID)
		countArgs = append(countArgs, *filter.ZoneID)
	}

	// Count total
	var total int
	if err := database.DB.QueryRow(database.Rebind(countQuery), countArgs...).Scan(&total); err != nil {
		log.Printf("Error counting speedtests: %v", err)
		utils.InternalError(c, "")
		return
	}

	query += " ORDER BY s.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, pageSize, offset)

	rows, err := database.DB.Query(database.Rebind(query), args...)
	if err != nil {
		log.Printf("Error querying speedtests: %v", err)
		utils.InternalError(c, "")
		return
	}
	defer rows.Close()

	results := []models.SpeedtestResponse{}
	for rows.Next() {
		var s models.SpeedtestResponse
		var providerID uint
		if err := rows.Scan(
			&s.ID, &providerID, &s.Provider, &s.DownloadMbps, &s.UploadMbps,
			&s.PingMs, &s.Latitude, &s.Longitude, &s.CreatedAt,
		); err != nil {
			log.Printf("Error scanning speedtest row: %v", err)
			continue
		}
		s.Quality = services.QualityFromMetrics(s.DownloadMbps, s.UploadMbps, s.PingMs)
		s.Score = services.CalculateScore(s.DownloadMbps, s.UploadMbps, s.PingMs)
		results = append(results, s)
	}

	utils.SuccessWithMeta(c, results, &utils.Meta{
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	})
}

// GetByID returns a single speedtest by ID.
// GET /api/v1/speedtests/:id
func (h *SpeedtestHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.ValidationError(c, "ID inválido")
		return
	}

	var s models.SpeedtestResponse
	var providerID uint
	err = database.DB.QueryRow(
		database.Rebind(`SELECT s.id, s.provider_id, p.nombre, s.download_mbps, s.upload_mbps, 
		        s.ping_ms, s.latitude, s.longitude, s.created_at
		 FROM speedtests s
		 JOIN providers p ON s.provider_id = p.id
		 WHERE s.id = ?`), id,
	).Scan(&s.ID, &providerID, &s.Provider, &s.DownloadMbps, &s.UploadMbps,
		&s.PingMs, &s.Latitude, &s.Longitude, &s.CreatedAt)

	if err == sql.ErrNoRows {
		utils.NotFound(c, "Speedtest no encontrado")
		return
	}
	if err != nil {
		log.Printf("Error querying speedtest: %v", err)
		utils.InternalError(c, "")
		return
	}

	s.Quality = services.QualityFromMetrics(s.DownloadMbps, s.UploadMbps, s.PingMs)
	s.Score = services.CalculateScore(s.DownloadMbps, s.UploadMbps, s.PingMs)

	utils.Success(c, http.StatusOK, "", s)
}

// findOrCreateProvider looks up a provider by name or creates it.
func findOrCreateProvider(nombre string) (uint, error) {
	var id uint
	err := database.DB.QueryRow(database.Rebind("SELECT id FROM providers WHERE nombre = ?"), nombre).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}

	// Provider doesn't exist — create it
	err = database.DB.QueryRow(database.Rebind("INSERT INTO providers (nombre) VALUES (?) RETURNING id"), nombre).Scan(&id)
	if err != nil {
		// Might be a race condition — try to read again
		err2 := database.DB.QueryRow(database.Rebind("SELECT id FROM providers WHERE nombre = ?"), nombre).Scan(&id)
		if err2 == nil {
			return id, nil
		}
		return 0, err
	}

	return id, nil
}

// nilIfEmpty returns a nil pointer if the string is empty, otherwise a pointer to the string.
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
