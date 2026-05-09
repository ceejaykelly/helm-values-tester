package merge

import (
	"fmt"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"sigs.k8s.io/yaml"
)

// Finds any null (nil) values in the map and returns the path to the first one found.
func findNullPath(m map[string]interface{}, prefix string) (string, bool) {
	for k, v := range m {
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		if v == nil {
			return path, true
		}
		switch vv := v.(type) {
		case map[string]interface{}:
			if p, found := findNullPath(vv, path); found {
				return p, true
			}
		case []interface{}:
			for i, elem := range vv {
				elemPath := fmt.Sprintf("%s[%d]", path, i)
				if elem == nil {
					return elemPath, true
				}
				if mm, ok := elem.(map[string]interface{}); ok {
					if p, found := findNullPath(mm, elemPath); found {
						return p, true
					}
				}
			}
		}
	}
	return "", false
}

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

	// Check for null (nil) values in the final merged result
	if path, found := findNullPath(currentMerged, ""); found {
		return []byte(fmt.Sprintf("A null value was detected at path: %s\n", path)), nil
	}

	return yaml.Marshal(currentMerged)
}
