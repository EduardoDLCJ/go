package handlers

import (
	"log"
	"net/http"
	"strconv"

	"apisql/database"
	"apisql/services"
	"apisql/utils"

	"github.com/gin-gonic/gin"
)

// MapHandler holds dependencies for map-related endpoints.
type MapHandler struct{}

// NewMapHandler creates a new MapHandler.
func NewMapHandler() *MapHandler {
	return &MapHandler{}
}

// MapPoint represents a single point on the map.
type MapPoint struct {
	ID           uint64  `json:"id"`
	Provider     string  `json:"provider"`
	DownloadMbps float64 `json:"download_mbps"`
	UploadMbps   float64 `json:"upload_mbps"`
	PingMs       float64 `json:"ping_ms"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	Quality      string  `json:"quality"`
	Score        float64 `json:"score"`
	Distance     float64 `json:"distance_km"`
}

// HeatmapPoint represents an aggregated data point for heatmaps.
type HeatmapPoint struct {
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	AvgDownload float64 `json:"avg_download_mbps"`
	AvgUpload   float64 `json:"avg_upload_mbps"`
	AvgPing     float64 `json:"avg_ping_ms"`
	TestCount   int     `json:"test_count"`
	Quality     string  `json:"quality"`
	Intensity   float64 `json:"intensity"` // 0-1 for heatmap rendering
}

// Points returns speedtest results near a given location.
// GET /api/v1/map/points?lat=20.67&lng=-103.35&radius=2&provider_id=1
func (h *MapHandler) Points(c *gin.Context) {
	// Parse required location parameters
	lat, err := strconv.ParseFloat(c.Query("lat"), 64)
	if err != nil {
		utils.ValidationError(c, "Parámetro 'lat' requerido y debe ser numérico")
		return
	}
	lng, err := strconv.ParseFloat(c.Query("lng"), 64)
	if err != nil {
		utils.ValidationError(c, "Parámetro 'lng' requerido y debe ser numérico")
		return
	}

	if err := utils.ValidateCoordinates(lat, lng); err != nil {
		utils.ValidationError(c, err.Error())
		return
	}

	// Optional radius.
	radius := utils.DefaultRadiusKm
	if r := c.Query("radius"); r != "" {
		if parsed, err := strconv.ParseFloat(r, 64); err == nil {
			radius = utils.ValidateRadius(parsed)
		}
	}

	// Build query with Haversine formula for distance calculation
	query := `
		SELECT id, nombre, download_mbps, upload_mbps, ping_ms, latitude, longitude, distance
		FROM (
			SELECT s.id, p.nombre, s.download_mbps, s.upload_mbps, s.ping_ms,
			       s.latitude, s.longitude,
			       (6371 * acos(
			           LEAST(1.0, cos(radians(?)) * cos(radians(s.latitude)) 
			           * cos(radians(s.longitude) - radians(?)) 
			           + sin(radians(?)) * sin(radians(s.latitude)))
			       )) AS distance
			FROM speedtests s
			JOIN providers p ON s.provider_id = p.id
		) sub
		WHERE distance <= ?
		ORDER BY distance ASC
		LIMIT 500`

	args := []interface{}{lat, lng, lat, radius}

	// Optional provider filter
	if providerIDStr := c.Query("provider_id"); providerIDStr != "" {
		if providerID, err := strconv.ParseUint(providerIDStr, 10, 32); err == nil {
			query = `
				SELECT id, nombre, download_mbps, upload_mbps, ping_ms, latitude, longitude, distance
				FROM (
					SELECT s.id, p.nombre, s.download_mbps, s.upload_mbps, s.ping_ms,
					       s.latitude, s.longitude,
					       (6371 * acos(
					           LEAST(1.0, cos(radians(?)) * cos(radians(s.latitude)) 
					           * cos(radians(s.longitude) - radians(?)) 
					           + sin(radians(?)) * sin(radians(s.latitude)))
					       )) AS distance
					FROM speedtests s
					JOIN providers p ON s.provider_id = p.id
					WHERE s.provider_id = ?
				) sub
				WHERE distance <= ?
				ORDER BY distance ASC
				LIMIT 500`
			args = []interface{}{lat, lng, lat, providerID, radius}
		}
	}

	rows, err := database.DB.Query(database.Rebind(query), args...)
	if err != nil {
		log.Printf("Error querying map points: %v", err)
		utils.InternalError(c, "")
		return
	}
	defer rows.Close()

	points := []MapPoint{}
	for rows.Next() {
		var p MapPoint
		if err := rows.Scan(
			&p.ID, &p.Provider, &p.DownloadMbps, &p.UploadMbps, &p.PingMs,
			&p.Latitude, &p.Longitude, &p.Distance,
		); err != nil {
			log.Printf("Error scanning map point: %v", err)
			continue
		}
		p.Quality = services.QualityFromMetrics(p.DownloadMbps, p.UploadMbps, p.PingMs)
		p.Score = services.CalculateScore(p.DownloadMbps, p.UploadMbps, p.PingMs)
		points = append(points, p)
	}

	utils.Success(c, http.StatusOK, "", gin.H{
		"center": gin.H{
			"lat": lat,
			"lng": lng,
		},
		"radius_km": radius,
		"count":     len(points),
		"points":    points,
	})
}

// Heatmap returns aggregated data points for heatmap rendering.
// GET /api/v1/map/heatmap?lat=20.67&lng=-103.35&radius=10
// Points are aggregated into ~0.01 degree grid cells (~1.1 km).
func (h *MapHandler) Heatmap(c *gin.Context) {
	lat, err := strconv.ParseFloat(c.Query("lat"), 64)
	if err != nil {
		utils.ValidationError(c, "Parámetro 'lat' requerido y debe ser numérico")
		return
	}
	lng, err := strconv.ParseFloat(c.Query("lng"), 64)
	if err != nil {
		utils.ValidationError(c, "Parámetro 'lng' requerido y debe ser numérico")
		return
	}

	if err := utils.ValidateCoordinates(lat, lng); err != nil {
		utils.ValidationError(c, err.Error())
		return
	}

	radius := utils.MaxRadiusKm
	if r := c.Query("radius"); r != "" {
		if parsed, err := strconv.ParseFloat(r, 64); err == nil {
			radius = utils.ValidateRadius(parsed)
		}
	}

	// Aggregate into grid cells of ~0.01 degrees (~1.1 km)
	query := `
		SELECT 
		    ROUND(s.latitude, 2) AS grid_lat,
		    ROUND(s.longitude, 2) AS grid_lng,
		    AVG(s.download_mbps) AS avg_download,
		    AVG(s.upload_mbps) AS avg_upload,
		    AVG(s.ping_ms) AS avg_ping,
		    COUNT(*) AS test_count
		FROM speedtests s
		WHERE (6371 * acos(
		    LEAST(1.0, cos(radians(?)) * cos(radians(s.latitude)) 
		    * cos(radians(s.longitude) - radians(?)) 
		    + sin(radians(?)) * sin(radians(s.latitude)))
		)) <= ?
		GROUP BY grid_lat, grid_lng
		ORDER BY avg_download DESC`

	rows, err := database.DB.Query(database.Rebind(query), lat, lng, lat, radius)
	if err != nil {
		log.Printf("Error querying heatmap: %v", err)
		utils.InternalError(c, "")
		return
	}
	defer rows.Close()

	points := []HeatmapPoint{}
	var maxScore float64

	for rows.Next() {
		var p HeatmapPoint
		if err := rows.Scan(&p.Latitude, &p.Longitude, &p.AvgDownload, &p.AvgUpload, &p.AvgPing, &p.TestCount); err != nil {
			log.Printf("Error scanning heatmap point: %v", err)
			continue
		}
		pointScore := services.CalculateScore(p.AvgDownload, p.AvgUpload, p.AvgPing)
		p.Quality = services.QualityFromMetrics(p.AvgDownload, p.AvgUpload, p.AvgPing)
		if pointScore > maxScore {
			maxScore = pointScore
		}
		points = append(points, p)
	}

	// Calculate intensity (0-1) relative to best nearby combined quality score.
	if maxScore > 0 {
		for i := range points {
			pointScore := services.CalculateScore(points[i].AvgDownload, points[i].AvgUpload, points[i].AvgPing)
			points[i].Intensity = pointScore / maxScore
		}
	}

	utils.Success(c, http.StatusOK, "", gin.H{
		"center": gin.H{
			"lat": lat,
			"lng": lng,
		},
		"radius_km": radius,
		"count":     len(points),
		"points":    points,
	})
}
