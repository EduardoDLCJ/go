package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIResponse is the standard envelope for all API responses.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// Meta holds pagination metadata.
type Meta struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

// Success sends a successful JSON response.
func Success(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// SuccessWithMeta sends a successful JSON response with pagination metadata.
func SuccessWithMeta(c *gin.Context, data interface{}, meta *Meta) {
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// Error sends an error JSON response.
func Error(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, APIResponse{
		Success: false,
		Error:   message,
	})
}

// ValidationError sends a 422 response for validation failures.
func ValidationError(c *gin.Context, message string) {
	c.JSON(http.StatusUnprocessableEntity, APIResponse{
		Success: false,
		Error:   "Error de validación: " + message,
	})
}

// Unauthorized sends a 401 response.
func Unauthorized(c *gin.Context, message string) {
	if message == "" {
		message = "No autorizado"
	}
	c.JSON(http.StatusUnauthorized, APIResponse{
		Success: false,
		Error:   message,
	})
}

// Forbidden sends a 403 response.
func Forbidden(c *gin.Context, message string) {
	if message == "" {
		message = "Acceso denegado"
	}
	c.JSON(http.StatusForbidden, APIResponse{
		Success: false,
		Error:   message,
	})
}

// NotFound sends a 404 response.
func NotFound(c *gin.Context, message string) {
	if message == "" {
		message = "Recurso no encontrado"
	}
	c.JSON(http.StatusNotFound, APIResponse{
		Success: false,
		Error:   message,
	})
}

// InternalError sends a 500 response.
func InternalError(c *gin.Context, message string) {
	if message == "" {
		message = "Error interno del servidor"
	}
	c.JSON(http.StatusInternalServerError, APIResponse{
		Success: false,
		Error:   message,
	})
}

// TooManyRequests sends a 429 response.
func TooManyRequests(c *gin.Context) {
	c.JSON(http.StatusTooManyRequests, APIResponse{
		Success: false,
		Error:   "Demasiadas solicitudes. Intente nuevamente en un momento.",
	})
}
