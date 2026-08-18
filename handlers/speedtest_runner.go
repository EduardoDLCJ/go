package handlers

import (
	"crypto/rand"
	"io"
	"net/http"
	"strconv"
	"time"

	"apisql/utils"

	"github.com/gin-gonic/gin"
)

// SpeedtestRunnerHandler provides endpoints for actual speed measurement.
// The client downloads/uploads data from these endpoints and measures timing.
type SpeedtestRunnerHandler struct{}

// NewSpeedtestRunnerHandler creates a new SpeedtestRunnerHandler.
func NewSpeedtestRunnerHandler() *SpeedtestRunnerHandler {
	return &SpeedtestRunnerHandler{}
}

// Ping returns a minimal response for latency measurement.
// GET /api/v1/speedtest/ping
// The client measures the round-trip time of this request.
func (h *SpeedtestRunnerHandler) Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"timestamp": time.Now().UnixMilli(),
	})
}

// Download streams random bytes to the client for download speed measurement.
// GET /api/v1/speedtest/download?size=25
// size = megabytes (default 25, max 100)
// The client measures how long it takes to receive all the data.
func (h *SpeedtestRunnerHandler) Download(c *gin.Context) {
	// Parse size in MB (default 25 MB)
	sizeMB := 25
	if s := c.Query("size"); s != "" {
		if parsed, err := strconv.Atoi(s); err == nil && parsed > 0 && parsed <= 100 {
			sizeMB = parsed
		}
	}

	totalBytes := int64(sizeMB) * 1024 * 1024

	// Set headers for binary streaming — disable caching
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", strconv.FormatInt(totalBytes, 10))
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("X-Speedtest-Size", strconv.Itoa(sizeMB))

	// Stream random data in 64 KB chunks
	chunkSize := 65536 // 64 KB
	chunk := make([]byte, chunkSize)
	remaining := totalBytes

	c.Status(http.StatusOK)

	for remaining > 0 {
		toWrite := int64(chunkSize)
		if remaining < toWrite {
			toWrite = remaining
			chunk = make([]byte, toWrite)
		}

		// Fill with random data to prevent compression from skewing results
		if _, err := rand.Read(chunk); err != nil {
			return
		}

		if _, err := c.Writer.Write(chunk); err != nil {
			return // Client disconnected
		}
		c.Writer.Flush()
		remaining -= toWrite
	}
}

// Upload accepts data from the client for upload speed measurement.
// POST /api/v1/speedtest/upload
// The client sends a blob of data and measures how long it takes.
func (h *SpeedtestRunnerHandler) Upload(c *gin.Context) {
	startTime := time.Now()

	// Read and discard all uploaded data, counting bytes
	// Limit to 100 MB to prevent abuse
	maxSize := int64(100 * 1024 * 1024)
	limitedReader := io.LimitReader(c.Request.Body, maxSize)

	bytesRead, err := io.Copy(io.Discard, limitedReader)
	if err != nil {
		utils.InternalError(c, "Error al recibir datos")
		return
	}

	elapsed := time.Since(startTime)

	c.JSON(http.StatusOK, gin.H{
		"bytes_received": bytesRead,
		"elapsed_ms":     elapsed.Milliseconds(),
		"timestamp":      time.Now().UnixMilli(),
	})
}
