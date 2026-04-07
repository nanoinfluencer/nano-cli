package country

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

//go:embed country.json
var countryJSON []byte

type countryRecord struct {
	Alpha2      string `json:"alpha-2"`
	CountryCode string `json:"country-code"`
}

var alpha2ToCode map[string]int

func init() {
	var records []countryRecord
	if err := json.Unmarshal(countryJSON, &records); err != nil {
		panic(err)
	}

	alpha2ToCode = make(map[string]int, len(records))
	for _, record := range records {
		code, err := strconv.Atoi(record.CountryCode)
		if err != nil {
			continue
		}
		alpha2 := strings.ToUpper(strings.TrimSpace(record.Alpha2))
		if alpha2 == "" {
			continue
		}
		alpha2ToCode[alpha2] = code
	}
}

func ResolveAlpha2List(values []string) ([]int, error) {
	if len(values) == 0 {
		return nil, nil
	}

	out := make([]int, 0, len(values))
	seen := map[int]bool{}
	for _, value := range values {
		alpha2 := strings.ToUpper(strings.TrimSpace(value))
		if alpha2 == "" {
			continue
		}
		code, ok := alpha2ToCode[alpha2]
		if !ok {
			return nil, fmt.Errorf("unsupported country code: %s", value)
		}
		if seen[code] {
			continue
		}
		seen[code] = true
		out = append(out, code)
	}
	return out, nil
}
