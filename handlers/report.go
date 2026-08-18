package handlers

import (
	"log"
	"net/http"

	"apisql/database"
	"apisql/models"
	"apisql/services"
	"apisql/utils"

	"github.com/gin-gonic/gin"
)

// ReportHandler holds dependencies for report endpoints.
type ReportHandler struct{}

// NewReportHandler creates a new ReportHandler.
func NewReportHandler() *ReportHandler {
	return &ReportHandler{}
}

// Create saves a new connectivity issue report.
// POST /api/v1/reports
// This endpoint is anonymous.
func (h *ReportHandler) Create(c *gin.Context) {
	var req models.ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, err.Error())
		return
	}

	// Validate coordinates
	if err := utils.ValidateCoordinates(req.Latitude, req.Longitude); err != nil {
		utils.ValidationError(c, err.Error())
		return
	}

	// Verify provider exists
	var providerName string
	err := database.DB.QueryRow(database.Rebind("SELECT nombre FROM providers WHERE id = ?"), req.ProviderID).Scan(&providerName)
	if err != nil {
		utils.ValidationError(c, "Proveedor no encontrado")
		return
	}

	// Hash IP
	clientIP := c.ClientIP()
	ipHash := services.HashIP(clientIP)

	// Sanitize description
	var descripcion *string
	if req.Descripcion != "" {
		sanitized := utils.SanitizeString(req.Descripcion)
		descripcion = &sanitized
	}

	// Insert report
	var reportID int64
	err = database.DB.QueryRow(
		database.Rebind(`INSERT INTO reports (provider_id, zone_id, latitude, longitude, issue_type, descripcion, ip_hash)
		 VALUES (?, ?, ?, ?, ?, ?, ?) RETURNING id`),
		req.ProviderID, req.ZoneID, req.Latitude, req.Longitude,
		req.IssueType, descripcion, ipHash,
	).Scan(&reportID)
	if err != nil {
		log.Printf("Error inserting report: %v", err)
		utils.InternalError(c, "")
		return
	}

	response := models.ReportResponse{
		ID:          uint64(reportID),
		Provider:    providerName,
		Latitude:    req.Latitude,
		Longitude:   req.Longitude,
		IssueType:   req.IssueType,
		Descripcion: descripcion,
	}

	utils.Success(c, http.StatusCreated, "Reporte guardado exitosamente", response)
}

// List returns reports with optional filters and pagination.
// GET /api/v1/reports
func (h *ReportHandler) List(c *gin.Context) {
	var filter models.ReportFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		utils.ValidationError(c, err.Error())
		return
	}

	page, pageSize := utils.ValidatePagination(filter.Page, filter.PageSize)
	offset := utils.PaginationOffset(page, pageSize)

	// Build dynamic query
	query := `
		SELECT r.id, r.provider_id, p.nombre, r.latitude, r.longitude, 
		       r.issue_type, r.descripcion, r.created_at
		FROM reports r
		JOIN providers p ON r.provider_id = p.id
		WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM reports r WHERE 1=1`
	args := []interface{}{}
	countArgs := []interface{}{}

	if filter.ProviderID != nil {
		query += " AND r.provider_id = ?"
		countQuery += " AND r.provider_id = ?"
		args = append(args, *filter.ProviderID)
		countArgs = append(countArgs, *filter.ProviderID)
	}
	if filter.ZoneID != nil {
		query += " AND r.zone_id = ?"
		countQuery += " AND r.zone_id = ?"
		args = append(args, *filter.ZoneID)
		countArgs = append(countArgs, *filter.ZoneID)
	}
	if filter.IssueType != "" {
		query += " AND r.issue_type = ?"
		countQuery += " AND r.issue_type = ?"
		args = append(args, filter.IssueType)
		countArgs = append(countArgs, filter.IssueType)
	}

	// Count total
	var total int
	if err := database.DB.QueryRow(database.Rebind(countQuery), countArgs...).Scan(&total); err != nil {
		log.Printf("Error counting reports: %v", err)
		utils.InternalError(c, "")
		return
	}

	query += " ORDER BY r.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, pageSize, offset)

	rows, err := database.DB.Query(database.Rebind(query), args...)
	if err != nil {
		log.Printf("Error querying reports: %v", err)
		utils.InternalError(c, "")
		return
	}
	defer rows.Close()

	results := []models.ReportResponse{}
	for rows.Next() {
		var r models.ReportResponse
		var providerID uint
		if err := rows.Scan(
			&r.ID, &providerID, &r.Provider, &r.Latitude, &r.Longitude,
			&r.IssueType, &r.Descripcion, &r.CreatedAt,
		); err != nil {
			log.Printf("Error scanning report row: %v", err)
			continue
		}
		results = append(results, r)
	}

	utils.SuccessWithMeta(c, results, &utils.Meta{
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	})
}
