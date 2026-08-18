package models

import "time"

// Zone represents a geographic zone (colonia, barrio).
type Zone struct {
	ID        uint      `json:"id"`
	Nombre    string    `json:"nombre"`
	Ciudad    string    `json:"ciudad"`
	Estado    string    `json:"estado"`
	CreatedAt time.Time `json:"created_at"`
}

// ZoneRequest is the expected body for creating a zone.
type ZoneRequest struct {
	Nombre string `json:"nombre" binding:"required,max=100"`
	Ciudad string `json:"ciudad" binding:"required,max=100"`
	Estado string `json:"estado" binding:"required,max=100"`
}
