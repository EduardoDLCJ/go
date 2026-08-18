package models

import "time"

// Speedtest represents a single speed test result.
type Speedtest struct {
	ID           uint64    `json:"id"`
	ProviderID   uint      `json:"provider_id"`
	ZoneID       *uint     `json:"zone_id,omitempty"` // nullable
	DownloadMbps float64   `json:"download_mbps"`
	UploadMbps   float64   `json:"upload_mbps"`
	PingMs       float64   `json:"ping_ms"`
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
	IPHash       *string   `json:"ip_hash,omitempty"`
	VisitorID    *string   `json:"visitor_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// SpeedtestRequest is the expected body for submitting a speed test.
// The ISP is detected automatically — the user only provides measurements and location.
type SpeedtestRequest struct {
	DownloadMbps float64 `json:"download_mbps" binding:"required,gt=0,lte=10000"`
	UploadMbps   float64 `json:"upload_mbps" binding:"required,gt=0,lte=10000"`
	PingMs       float64 `json:"ping_ms" binding:"required,gt=0,lte=5000"`
	Latitude     float64 `json:"latitude" binding:"required,min=-90,max=90"`
	Longitude    float64 `json:"longitude" binding:"required,min=-180,max=180"`
	VisitorID    string  `json:"visitor_id" binding:"omitempty,max=64"`
	ZoneID       *uint   `json:"zone_id" binding:"omitempty"`
}

// SpeedtestResponse is the public representation returned after saving a test.
type SpeedtestResponse struct {
	ID           uint64    `json:"id"`
	Provider     string    `json:"provider"`
	DownloadMbps float64   `json:"download_mbps"`
	UploadMbps   float64   `json:"upload_mbps"`
	PingMs       float64   `json:"ping_ms"`
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
	Quality      string    `json:"quality"` // excelente, bueno, malo
	Score        float64   `json:"score"`
	CreatedAt    time.Time `json:"created_at"`
}

// SpeedtestFilter holds query parameters for filtering speedtests.
type SpeedtestFilter struct {
	ProviderID *uint   `form:"provider_id"`
	ZoneID     *uint   `form:"zone_id"`
	MinQuality string  `form:"min_quality"`
	Page       int     `form:"page,default=1"`
	PageSize   int     `form:"page_size,default=20"`
}

// QualityFromDownload returns a quality label based on download speed.
func QualityFromDownload(downloadMbps float64) string {
	switch {
	case downloadMbps >= 300:
		return "excelente"
	case downloadMbps >= 100:
		return "bueno"
	default:
		return "malo"
	}
}
