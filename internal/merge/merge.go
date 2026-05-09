package merge

import (
	"fmt"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"sigs.k8s.io/yaml"
)

// Values merges multiple YAML files in order.
// files[0] is the base, files[1+] are overrides.
func Values(files ...[]byte) ([]byte, error) {
	if len(files) == 0 {
		return nil, nil
	}

	// Start with the first file as the base
	var baseMap map[string]interface{}
	if err := yaml.Unmarshal(files[0], &baseMap); err != nil {
		return nil, err
	}

	currentMerged := baseMap

	// Iteratively merge each subsequent file into the result
	for i := 1; i < len(files); i++ {
		var overrideMap map[string]interface{}
		if err := yaml.Unmarshal(files[i], &overrideMap); err != nil {
			// If failed unmarshal, return the error (avoid double prefix)
			return nil, err
		}

		// Check for null (nil) values at the top level of the override
		for k, v := range overrideMap {
			if v == nil {
				return []byte(fmt.Sprintf("A null value was detected at path: %s\n", k)), nil
			}
		}

		// Create a dummy chart for the current state
		ch := &chart.Chart{
			Metadata: &chart.Metadata{Name: "tester"},
			Values:   currentMerged,
		}

		// Coalesce the next layer
		next, err := chartutil.CoalesceValues(ch, overrideMap)
		if err != nil {
			return nil, err
		}
		currentMerged = next
	}

	return yaml.Marshal(currentMerged)
}
