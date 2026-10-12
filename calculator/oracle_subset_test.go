package calculator

import (
	"fmt"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/internal/oraclebin"
)

func oracleRecordsForBuilds(want ...string) (map[string]oracleRecord, error) {
	path, err := oracleBlobForVersion(raw.LatestVersion)
	if err != nil {
		return nil, err
	}
	recs, err := oraclebin.LoadSubset(path, want)
	if err != nil {
		return nil, fmt.Errorf("loading %v from the oracle blob: %w", want, err)
	}
	return recs, nil
}
