package handlers

import (
	"log"
	"net/http"
	"strconv"

	"apisql/database"
	"apisql/models"
	"apisql/services"
	"apisql/utils"

	"github.com/gin-gonic/gin"
)

// ZoneHandler holds dependencies for zone endpoints.
type ZoneHandler struct{}

// NewZoneHandler creates a new ZoneHandler.
func NewZoneHandler() *ZoneHandler {
	return &ZoneHandler{}
}

// List returns all zones.
// GET /api/v1/zones
func (h *ZoneHandler) List(c *gin.Context) {
	ciudad := c.Query("ciudad")
	estado := c.Query("estado")

	query := `SELECT id, nombre, ciudad, estado, created_at FROM zones WHERE 1=1`
	args := []interface{}{}

	if ciudad != "" {
		query += " AND ciudad = ?"
		args = append(args, ciudad)
	}
	if estado != "" {
		query += " AND estado = ?"
		args = append(args, estado)
	}

	query += " ORDER BY estado, ciudad, nombre"

	rows, err := database.DB.Query(database.Rebind(query), args...)
	if err != nil {
		log.Printf("Error querying zones: %v", err)
		utils.InternalError(c, "")
		return
	}
	defer rows.Close()

	results := []models.Zone{}
	for rows.Next() {
		var z models.Zone
		if err := rows.Scan(&z.ID, &z.Nombre, &z.Ciudad, &z.Estado, &z.CreatedAt); err != nil {
			log.Printf("Error scanning zone row: %v", err)
			continue
		}
		results = append(results, z)
	}

	utils.Success(c, http.StatusOK, "", results)
}

// Create creates a new zone.
// POST /api/v1/zones
func (h *ZoneHandler) Create(c *gin.Context) {
	var req models.ZoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, err.Error())
		return
	}

	// Sanitize inputs
	req.Nombre = utils.SanitizeString(req.Nombre)
	req.Ciudad = utils.SanitizeString(req.Ciudad)
	req.Estado = utils.SanitizeString(req.Estado)

	var zoneID int64
	dbErr := database.DB.QueryRow(
		database.Rebind("INSERT INTO zones (nombre, ciudad, estado) VALUES (?, ?, ?) RETURNING id"),
		req.Nombre, req.Ciudad, req.Estado,
	).Scan(&zoneID)
	if dbErr != nil {
		log.Printf("Error inserting zone: %v", dbErr)
		utils.InternalError(c, "")
		return
	}

	utils.Success(c, http.StatusCreated, "Zona creada exitosamente", models.Zone{
		ID:     uint(zoneID),
		Nombre: req.Nombre,
		Ciudad: req.Ciudad,
		Estado: req.Estado,
	})
}

// Ranking returns the provider ranking for a specific zone.
// GET /api/v1/zones/:id/ranking
func (h *ZoneHandler) Ranking(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.ValidationError(c, "ID de zona inválido")
		return
	}

	// Verify zone exists
	var zoneName string
	err = database.DB.QueryRow(database.Rebind("SELECT nombre FROM zones WHERE id = ?"), id).Scan(&zoneName)
	if err != nil {
		utils.NotFound(c, "Zona no encontrada")
		return
	}

	// Get rankings grouped by provider
	rows, err := database.DB.Query(
		database.Rebind(`SELECT p.id, p.nombre,
		        AVG(s.download_mbps) AS avg_download,
		        AVG(s.upload_mbps) AS avg_upload,
		        AVG(s.ping_ms) AS avg_ping,
		        COUNT(s.id) AS total_tests
		 FROM speedtests s
		 JOIN providers p ON s.provider_id = p.id
		 WHERE s.zone_id = ?
		 GROUP BY p.id, p.nombre
		 HAVING total_tests >= 1
		 ORDER BY avg_download DESC`),
		id,
	)
	if err != nil {
		log.Printf("Error querying zone ranking: %v", err)
		utils.InternalError(c, "")
		return
	}
	defer rows.Close()

	rankings := []models.ProviderRanking{}
	for rows.Next() {
		var r models.ProviderRanking
		if err := rows.Scan(
			&r.ProviderID, &r.Nombre,
			&r.AvgDownload, &r.AvgUpload, &r.AvgPing, &r.TotalTests,
		); err != nil {
			log.Printf("Error scanning ranking row: %v", err)
			continue
		}
		r.Score = services.CalculateScore(r.AvgDownload, r.AvgUpload, r.AvgPing)
		rankings = append(rankings, r)
	}

	utils.Success(c, http.StatusOK, "", gin.H{
		"zone":     zoneName,
		"zone_id":  id,
		"rankings": rankings,
	})
}
