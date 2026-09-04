package handlers

import (
	"database/sql"
	"log"
	"math"
	"net/http"
	"strconv"

	"apisql/database"
	"apisql/models"
	"apisql/services"
	"apisql/utils"

	"github.com/gin-gonic/gin"
)

// ProviderHandler holds dependencies for provider endpoints.
type ProviderHandler struct{}

// NewProviderHandler creates a new ProviderHandler.
func NewProviderHandler() *ProviderHandler {
	return &ProviderHandler{}
}

// List returns all known ISP providers.
// GET /api/v1/providers
func (h *ProviderHandler) List(c *gin.Context) {
	rows, err := database.DB.Query(
		`SELECT p.id, p.nombre, p.created_at,
		        COUNT(s.id) AS total_tests,
		        COALESCE(AVG(s.download_mbps), 0) AS avg_download,
		        COALESCE(AVG(s.upload_mbps), 0) AS avg_upload,
		        COALESCE(AVG(s.ping_ms), 0) AS avg_ping
		 FROM providers p
		 LEFT JOIN speedtests s ON p.id = s.provider_id
		 GROUP BY p.id, p.nombre, p.created_at
		 ORDER BY total_tests DESC`,
	)
	if err != nil {
		log.Printf("Error querying providers: %v", err)
		utils.InternalError(c, "")
		return
	}
	defer rows.Close()

	results := []models.ProviderStats{}
	for rows.Next() {
		var p models.ProviderStats
		var createdAt sql.NullTime
		if err := rows.Scan(
			&p.ID, &p.Nombre, &createdAt,
			&p.TotalTests, &p.AvgDownload, &p.AvgUpload, &p.AvgPing,
		); err != nil {
			log.Printf("Error scanning provider row: %v", err)
			continue
		}
		p.Score = services.CalculateScore(p.AvgDownload, p.AvgUpload, p.AvgPing)
		results = append(results, p)
	}

	utils.Success(c, http.StatusOK, "", results)
}

// Nearby returns providers inferred from speedtests near a location or within a bounding box.
// GET /api/v1/map/providers?lat=20.67&lng=-103.35&radius=2&limit=20
// GET /api/v1/map/providers?min_lat=20.45&max_lat=20.50&min_lng=-103.60&max_lng=-103.50&limit=20
func (h *ProviderHandler) Nearby(c *gin.Context) {
	limit := 20
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsed, err := strconv.Atoi(limitParam); err == nil {
			limit = parsed
		}
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	minTests := 1
	if minTestsParam := c.Query("min_tests"); minTestsParam != "" {
		if parsed, err := strconv.Atoi(minTestsParam); err == nil {
			minTests = parsed
		}
	}
	if minTests < 1 {
		minTests = 1
	}

	minLatStr := c.Query("min_lat")
	maxLatStr := c.Query("max_lat")
	minLngStr := c.Query("min_lng")
	maxLngStr := c.Query("max_lng")

	isBoundingBox := minLatStr != "" && maxLatStr != "" && minLngStr != "" && maxLngStr != ""

	var (
		lat, lng float64
		minLat, maxLat, minLng, maxLng float64
		radius float64
		hasRadius bool
	)

	if isBoundingBox {
		var err error
		minLat, err = strconv.ParseFloat(minLatStr, 64)
		if err != nil {
			utils.ValidationError(c, "Parámetro 'min_lat' debe ser numérico")
			return
		}
		maxLat, err = strconv.ParseFloat(maxLatStr, 64)
		if err != nil {
			utils.ValidationError(c, "Parámetro 'max_lat' debe ser numérico")
			return
		}
		minLng, err = strconv.ParseFloat(minLngStr, 64)
		if err != nil {
			utils.ValidationError(c, "Parámetro 'min_lng' debe ser numérico")
			return
		}
		maxLng, err = strconv.ParseFloat(maxLngStr, 64)
		if err != nil {
			utils.ValidationError(c, "Parámetro 'max_lng' debe ser numérico")
			return
		}

		if err := utils.ValidateCoordinates(minLat, minLng); err != nil {
			utils.ValidationError(c, "Coordenadas mínimas inválidas: "+err.Error())
			return
		}
		if err := utils.ValidateCoordinates(maxLat, maxLng); err != nil {
			utils.ValidationError(c, "Coordenadas máximas inválidas: "+err.Error())
			return
		}

		if minLat > maxLat {
			minLat, maxLat = maxLat, minLat
		}
		if minLng > maxLng {
			minLng, maxLng = maxLng, minLng
		}

		lat = (minLat + maxLat) / 2.0
		lng = (minLng + maxLng) / 2.0
	} else {
		var err error
		lat, err = strconv.ParseFloat(c.Query("lat"), 64)
		if err != nil {
			utils.ValidationError(c, "Parámetro 'lat' requerido y debe ser numérico (o indicar min_lat, max_lat, min_lng, max_lng)")
			return
		}

		lng, err = strconv.ParseFloat(c.Query("lng"), 64)
		if err != nil {
			utils.ValidationError(c, "Parámetro 'lng' requerido y debe ser numérico (o indicar min_lat, max_lat, min_lng, max_lng)")
			return
		}

		if err := utils.ValidateCoordinates(lat, lng); err != nil {
			utils.ValidationError(c, err.Error())
			return
		}

		radius = utils.DefaultRadiusKm
		if radiusParam := c.Query("radius"); radiusParam != "" {
			parsed, err := strconv.ParseFloat(radiusParam, 64)
			if err != nil {
				utils.ValidationError(c, "Parámetro 'radius' debe ser numérico")
				return
			}
			radius = utils.ValidateRadius(parsed)
		}
		hasRadius = true

		deltaLat := radius / 111.32
		cosLat := math.Cos(lat * math.Pi / 180)
		deltaLng := 180.0
		if math.Abs(cosLat) > 0.000001 {
			deltaLng = radius / (111.32 * math.Abs(cosLat))
		}

		minLat = math.Max(-90, lat-deltaLat)
		maxLat = math.Min(90, lat+deltaLat)
		minLng = math.Max(-180, lng-deltaLng)
		maxLng = math.Min(180, lng+deltaLng)
	}

	var query string
	var args []interface{}

	if isBoundingBox {
		query = `
			SELECT
			    p.id,
			    p.nombre,
			    COUNT(*) AS total_tests,
			    AVG(nearby.download_mbps) AS avg_download,
			    AVG(nearby.upload_mbps) AS avg_upload,
			    AVG(nearby.ping_ms) AS avg_ping,
			    AVG(nearby.latitude) AS avg_latitude,
			    AVG(nearby.longitude) AS avg_longitude,
			    MIN(nearby.latitude) AS min_latitude,
			    MAX(nearby.latitude) AS max_latitude,
			    MIN(nearby.longitude) AS min_longitude,
			    MAX(nearby.longitude) AS max_longitude,
			    MIN(nearby.distance_km) AS min_distance,
			    AVG(nearby.distance_km) AS avg_distance,
			    MAX(nearby.created_at) AS last_test_at
			FROM (
			    SELECT
			        s.provider_id,
			        s.download_mbps,
			        s.upload_mbps,
			        s.ping_ms,
			        s.latitude,
			        s.longitude,
			        s.created_at,
			        (6371 * ACOS(
			            LEAST(1.0, GREATEST(-1.0,
			                COS(RADIANS(?)) * COS(RADIANS(s.latitude))
			                * COS(RADIANS(s.longitude) - RADIANS(?))
			                + SIN(RADIANS(?)) * SIN(RADIANS(s.latitude))
			            ))
			        )) AS distance_km
			    FROM speedtests s
			    WHERE s.latitude BETWEEN ? AND ?
			      AND s.longitude BETWEEN ? AND ?
			) nearby
			JOIN providers p ON p.id = nearby.provider_id
			GROUP BY p.id, p.nombre
			HAVING COUNT(*) >= ?
			ORDER BY total_tests DESC, avg_download DESC
			LIMIT ?`

		args = []interface{}{
			lat, lng, lat,
			minLat, maxLat, minLng, maxLng,
			minTests, limit,
		}
	} else {
		query = `
			SELECT
			    p.id,
			    p.nombre,
			    COUNT(*) AS total_tests,
			    AVG(nearby.download_mbps) AS avg_download,
			    AVG(nearby.upload_mbps) AS avg_upload,
			    AVG(nearby.ping_ms) AS avg_ping,
			    AVG(nearby.latitude) AS avg_latitude,
			    AVG(nearby.longitude) AS avg_longitude,
			    MIN(nearby.latitude) AS min_latitude,
			    MAX(nearby.latitude) AS max_latitude,
			    MIN(nearby.longitude) AS min_longitude,
			    MAX(nearby.longitude) AS max_longitude,
			    MIN(nearby.distance_km) AS min_distance,
			    AVG(nearby.distance_km) AS avg_distance,
			    MAX(nearby.created_at) AS last_test_at
			FROM (
			    SELECT
			        s.provider_id,
			        s.download_mbps,
			        s.upload_mbps,
			        s.ping_ms,
			        s.latitude,
			        s.longitude,
			        s.created_at,
			        (6371 * ACOS(
			            LEAST(1.0, GREATEST(-1.0,
			                COS(RADIANS(?)) * COS(RADIANS(s.latitude))
			                * COS(RADIANS(s.longitude) - RADIANS(?))
			                + SIN(RADIANS(?)) * SIN(RADIANS(s.latitude))
			            ))
			        )) AS distance_km
			    FROM speedtests s
			    WHERE s.latitude BETWEEN ? AND ?
			      AND s.longitude BETWEEN ? AND ?
			) nearby
			JOIN providers p ON p.id = nearby.provider_id
			WHERE nearby.distance_km <= ?
			GROUP BY p.id, p.nombre
			HAVING COUNT(*) >= ?
			ORDER BY total_tests DESC, avg_download DESC
			LIMIT ?`

		args = []interface{}{
			lat, lng, lat,
			minLat, maxLat, minLng, maxLng,
			radius, minTests, limit,
		}
	}

	rows, err := database.DB.Query(database.Rebind(query), args...)
	if err != nil {
		log.Printf("Error querying nearby providers: %v", err)
		utils.InternalError(c, "")
		return
	}
	defer rows.Close()

	results := []models.NearbyProviderStats{}
	for rows.Next() {
		var p models.NearbyProviderStats
		if err := rows.Scan(
			&p.ProviderID, &p.Nombre, &p.TotalTests,
			&p.AvgDownload, &p.AvgUpload, &p.AvgPing,
			&p.AvgLatitude, &p.AvgLongitude,
			&p.MinLatitude, &p.MaxLatitude, &p.MinLongitude, &p.MaxLongitude,
			&p.MinDistanceKm, &p.AvgDistanceKm, &p.LastTestAt,
		); err != nil {
			log.Printf("Error scanning nearby provider row: %v", err)
			continue
		}
		p.Score = services.CalculateScore(p.AvgDownload, p.AvgUpload, p.AvgPing)
		p.Confidence = confidenceFromTestsAndScore(p.TotalTests, p.Score)
		results = append(results, p)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error iterating nearby providers: %v", err)
		utils.InternalError(c, "")
		return
	}

	responseData := gin.H{
		"center": gin.H{
			"lat": lat,
			"lng": lng,
		},
		"count":     len(results),
		"providers": results,
	}

	if hasRadius {
		responseData["radius_km"] = radius
	}

	if isBoundingBox {
		responseData["bounding_box"] = gin.H{
			"min_lat": minLat,
			"max_lat": maxLat,
			"min_lng": minLng,
			"max_lng": maxLng,
		}
	}

	utils.Success(c, http.StatusOK, "", responseData)
}

// GetByID returns detailed statistics for a single provider.
// GET /api/v1/providers/:id
func (h *ProviderHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.ValidationError(c, "ID inválido")
		return
	}

	var p models.ProviderStats
	err = database.DB.QueryRow(
		database.Rebind(`SELECT p.id, p.nombre,
		        COUNT(s.id) AS total_tests,
		        COALESCE(AVG(s.download_mbps), 0) AS avg_download,
		        COALESCE(AVG(s.upload_mbps), 0) AS avg_upload,
		        COALESCE(AVG(s.ping_ms), 0) AS avg_ping,
		        COUNT(DISTINCT s.zone_id) AS zones_covered
		 FROM providers p
		 LEFT JOIN speedtests s ON p.id = s.provider_id
		 WHERE p.id = ?
		 GROUP BY p.id, p.nombre`),
		id,
	).Scan(&p.ID, &p.Nombre, &p.TotalTests, &p.AvgDownload, &p.AvgUpload, &p.AvgPing, &p.ZonesCovered)

	if err == sql.ErrNoRows {
		utils.NotFound(c, "Proveedor no encontrado")
		return
	}
	if err != nil {
		log.Printf("Error querying provider: %v", err)
		utils.InternalError(c, "")
		return
	}

	p.Score = services.CalculateScore(p.AvgDownload, p.AvgUpload, p.AvgPing)

	utils.Success(c, http.StatusOK, "", p)
}

func confidenceFromTestsAndScore(totalTests int, score float64) string {
	switch {
	case totalTests >= 12 && score >= 70:
		return "alta"
	case totalTests >= 6 && score >= 55:
		return "media"
	default:
		return "baja"
	}
}
