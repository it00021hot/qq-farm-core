package activitycenter

import (
	_ "embed"
	"encoding/json"
)

//go:embed catalog/constellation-2026072701.json
var constellationCatalogRaw []byte

type constellationCatalog struct {
	CatalogVersion int    `json:"catalogVersion"`
	ActivityID     string `json:"activityId"`
	DisplayName    string `json:"displayName"`
	ServerName     string `json:"serverName"`
	Groups         []struct {
		ID       string `json:"id"`
		Order    int    `json:"order"`
		NodeID   string `json:"nodeId"`
		Name     string `json:"name"`
		Category string `json:"category"`
		Rewards  []map[string]any `json:"rewards"`
	} `json:"groups"`
}

var cachedCatalog *constellationCatalog

// LoadConstellationCatalog returns the embedded constellation catalog.
func LoadConstellationCatalog() *constellationCatalog {
	if cachedCatalog != nil {
		return cachedCatalog
	}
	var c constellationCatalog
	if err := json.Unmarshal(constellationCatalogRaw, &c); err != nil {
		return nil
	}
	cachedCatalog = &c
	return cachedCatalog
}
