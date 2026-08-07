package activitycenter

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed catalog/constellation-2026072701.json
var constellationCatalogRaw []byte

// flexString unmarshals JSON string or number into a string (catalog nodeId is numeric).
type flexString string

func (s *flexString) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*s = ""
		return nil
	}
	var asString string
	if err := json.Unmarshal(b, &asString); err == nil {
		*s = flexString(asString)
		return nil
	}
	var asNumber json.Number
	if err := json.Unmarshal(b, &asNumber); err == nil {
		*s = flexString(asNumber.String())
		return nil
	}
	return fmt.Errorf("flexString: unsupported JSON %s", string(b))
}

func (s flexString) String() string { return string(s) }

type constellationCatalog struct {
	CatalogVersion int    `json:"catalogVersion"`
	ActivityID     string `json:"activityId"`
	DisplayName    string `json:"displayName"`
	ServerName     string `json:"serverName"`
	Groups         []struct {
		ID       string           `json:"id"`
		Order    int              `json:"order"`
		NodeID   flexString       `json:"nodeId"`
		Name     string           `json:"name"`
		Category string           `json:"category"`
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
