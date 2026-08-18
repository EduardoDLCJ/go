package models

import "time"

// Provider represents an ISP (Internet Service Provider).
type Provider struct {
	ID        uint      `json:"id"`
	Nombre    string    `json:"nombre"`
	CreatedAt time.Time `json:"created_at"`
}

// ProviderStats holds aggregated statistics for a provider.
type ProviderStats struct {
	ID           uint    `json:"id"`
	Nombre       string  `json:"nombre"`
	AvgDownload  float64 `json:"avg_download_mbps"`
	AvgUpload    float64 `json:"avg_upload_mbps"`
	AvgPing      float64 `json:"avg_ping_ms"`
	TotalTests   int     `json:"total_tests"`
	Score        float64 `json:"score"`
	ZonesCovered int     `json:"zones_covered"`
}

// ProviderRanking is used in zone rankings.
type ProviderRanking struct {
	ProviderID  uint    `json:"provider_id"`
	Nombre      string  `json:"nombre"`
	AvgDownload float64 `json:"avg_download_mbps"`
	AvgUpload   float64 `json:"avg_upload_mbps"`
	AvgPing     float64 `json:"avg_ping_ms"`
	TotalTests  int     `json:"total_tests"`
	Score       float64 `json:"score"`
}

// NearbyProviderStats holds provider quality inferred from nearby speedtests.
type NearbyProviderStats struct {
	ProviderID    uint      `json:"provider_id"`
	Nombre        string    `json:"nombre"`
	AvgDownload   float64   `json:"avg_download_mbps"`
	AvgUpload     float64   `json:"avg_upload_mbps"`
	AvgPing       float64   `json:"avg_ping_ms"`
	TotalTests    int       `json:"total_tests"`
	Score         float64   `json:"score"`
	Confidence    string    `json:"confidence"`
	MinDistanceKm float64   `json:"min_distance_km"`
	AvgDistanceKm float64   `json:"avg_distance_km"`
	LastTestAt    time.Time `json:"last_test_at"`
}
