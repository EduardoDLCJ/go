package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ispCache caches ISP lookups by IP hash to avoid redundant external calls.
var (
	ispCache    = make(map[string]ispCacheEntry)
	ispCacheMu  sync.RWMutex
	cacheTTL    = 24 * time.Hour
)

type ispCacheEntry struct {
	ISP       string
	ExpiresAt time.Time
}

// ipAPIResponse represents the response from ip-api.com.
type ipAPIResponse struct {
	Status  string `json:"status"`
	ISP     string `json:"isp"`
	Org     string `json:"org"`
	AS      string `json:"as"`
	Message string `json:"message"`
}

// DetectISP queries ip-api.com to identify the ISP for a given IP address.
// Results are cached for 24 hours to reduce external API calls.
// Returns the ISP name or "Desconocido" if detection fails.
func DetectISP(ip string) string {
	// Strip port if present
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		// Check if this is IPv6 — if it contains multiple colons, don't strip
		if strings.Count(ip, ":") == 1 {
			ip = ip[:idx]
		}
	}

	// Skip private/local IPs
	if isPrivateIP(ip) {
		return "Local/Privada"
	}

	// Check cache
	ipHash := HashIP(ip)
	ispCacheMu.RLock()
	if entry, ok := ispCache[ipHash]; ok && time.Now().Before(entry.ExpiresAt) {
		ispCacheMu.RUnlock()
		return entry.ISP
	}
	ispCacheMu.RUnlock()

	// Query external API
	isp := queryISPAPI(ip)

	// Cache the result
	ispCacheMu.Lock()
	ispCache[ipHash] = ispCacheEntry{
		ISP:       isp,
		ExpiresAt: time.Now().Add(cacheTTL),
	}
	ispCacheMu.Unlock()

	return isp
}

// queryISPAPI makes the actual HTTP call to ip-api.com.
func queryISPAPI(ip string) string {
	client := &http.Client{Timeout: 5 * time.Second}

	url := fmt.Sprintf("http://ip-api.com/json/%s?fields=status,isp,org,as,message", ip)
	resp, err := client.Get(url)
	if err != nil {
		return "Desconocido"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "Desconocido"
	}

	var result ipAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "Desconocido"
	}

	if result.Status != "success" || result.ISP == "" {
		return "Desconocido"
	}

	return normalizeISPName(result.ISP)
}

// normalizeISPName cleans up common ISP name variations for consistency.
func normalizeISPName(isp string) string {
	isp = strings.TrimSpace(isp)

	// Map of known ISP name variations to canonical names
	knownISPs := map[string]string{
		"telefonos de mexico":           "Telmex",
		"telmex":                        "Telmex",
		"telnor":                        "Telmex",
		"uninet":                        "Telmex",
		"total play telecomunicaciones": "Totalplay",
		"totalplay":                     "Totalplay",
		"megacable":                     "Megacable",
		"megacable comunicaciones":      "Megacable",
		"axtel":                         "Axtel",
		"izzi telecom":                  "Izzi",
		"izzi":                          "Izzi",
		"cablevision":                   "Izzi",
	}

	lower := strings.ToLower(isp)
	for key, canonical := range knownISPs {
		if strings.Contains(lower, key) {
			return canonical
		}
	}

	return isp
}

// isPrivateIP checks if an IP belongs to a private/reserved range.
func isPrivateIP(ip string) bool {
	privateRanges := []string{
		"127.", "10.", "192.168.", "172.16.", "172.17.", "172.18.",
		"172.19.", "172.20.", "172.21.", "172.22.", "172.23.",
		"172.24.", "172.25.", "172.26.", "172.27.", "172.28.",
		"172.29.", "172.30.", "172.31.", "::1", "fe80:",
	}
	for _, prefix := range privateRanges {
		if strings.HasPrefix(ip, prefix) {
			return true
		}
	}
	return false
}
