package models

import "time"

// Report represents a connectivity issue report from a user.
type Report struct {
	ID          uint64    `json:"id"`
	ProviderID  uint      `json:"provider_id"`
	ZoneID      *uint     `json:"zone_id,omitempty"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	IssueType   string    `json:"issue_type"`
	Descripcion *string   `json:"descripcion,omitempty"`
	IPHash      *string   `json:"ip_hash,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// ReportRequest is the expected body for submitting a report.
type ReportRequest struct {
	ProviderID  uint    `json:"provider_id" binding:"required"`
	Latitude    float64 `json:"latitude" binding:"required,min=-90,max=90"`
	Longitude   float64 `json:"longitude" binding:"required,min=-180,max=180"`
	IssueType   string  `json:"issue_type" binding:"required,oneof=sin_internet lento intermitente"`
	Descripcion string  `json:"descripcion" binding:"omitempty,max=1000"`
	ZoneID      *uint   `json:"zone_id" binding:"omitempty"`
}

// ReportResponse is the public representation of a report.
type ReportResponse struct {
	ID          uint64    `json:"id"`
	Provider    string    `json:"provider"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	IssueType   string    `json:"issue_type"`
	Descripcion *string   `json:"descripcion,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// ReportFilter holds query parameters for filtering reports.
type ReportFilter struct {
	ProviderID *uint  `form:"provider_id"`
	ZoneID     *uint  `form:"zone_id"`
	IssueType  string `form:"issue_type"`
	Page       int    `form:"page,default=1"`
	PageSize   int    `form:"page_size,default=20"`
}

// Valid issue types for validation.
var ValidIssueTypes = []string{"sin_internet", "lento", "intermitente"}
