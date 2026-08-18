package services

import "math"

// ScoreWeights defines the weight of each metric in the overall score.
var ScoreWeights = struct {
	Download float64
	Upload   float64
	Ping     float64
	// Stability is derived from ping consistency (lower = more stable)
}{
	Download: 0.50,
	Upload:   0.20,
	Ping:     0.20,
	// Remaining 0.10 is for stability
}

// CalculateScore computes a quality score from 0 to 100 for a speed test.
//
// Formula:
//
//	50% download (normalized to 0-100 based on 1000 Mbps max)
//	20% upload   (normalized to 0-100 based on 500 Mbps max)
//	20% ping     (inverted: lower ping = higher score, 0ms=100, 200ms+=0)
//	10% stability (derived from ping — lower ping implies more stability)
func CalculateScore(downloadMbps, uploadMbps, pingMs float64) float64 {
	// Normalize download: 0 Mbps = 0, 1000+ Mbps = 100
	downloadScore := math.Min(downloadMbps/1000.0*100.0, 100.0)

	// Normalize upload: 0 Mbps = 0, 500+ Mbps = 100
	uploadScore := math.Min(uploadMbps/500.0*100.0, 100.0)

	// Normalize ping (inverted): 0 ms = 100, 200+ ms = 0
	pingScore := math.Max(0, 100.0-pingMs/2.0)

	// Stability estimate from ping (very low ping = very stable)
	stabilityScore := math.Max(0, 100.0-pingMs/1.5)

	score := downloadScore*0.50 +
		uploadScore*0.20 +
		pingScore*0.20 +
		stabilityScore*0.10

	// Clamp to 0–100
	return math.Round(math.Max(0, math.Min(100, score))*100) / 100
}

// QualityFromMetrics returns a stricter quality label based on the combined score.
// It uses download/upload/ping through CalculateScore instead of only one metric.
func QualityFromMetrics(downloadMbps, uploadMbps, pingMs float64) string {
	score := CalculateScore(downloadMbps, uploadMbps, pingMs)

	switch {
	case score >= 80:
		return "excelente"
	case score >= 60:
		return "bueno"
	default:
		return "malo"
	}
}
