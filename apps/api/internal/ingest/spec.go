package ingest

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ScraperSpec is one harvest/scrapers/*.yaml file, parsed. Only the fields
// an adapter actually needs at runtime are typed here — the yaml files also
// carry api/fields_from_shopify_json/quirks/http/notes blocks that are
// harvest-file documentation for a human (or a future adapter author), the
// same way harvest/rules/cannabinoids.json's "notes" object isn't loaded
// into resolve.CannabinoidRuleSet. Untyped fields are silently ignored by
// yaml.Unmarshal, not an error — see LoadScraperSpec's doc comment for why
// that's deliberate here and not a validation gap like ruleset.go's was.
type ScraperSpec struct {
	Slug              string            `yaml:"slug"`
	Name              string            `yaml:"name"`
	BaseURL           string            `yaml:"base_url"`
	Platform          string            `yaml:"platform"`
	TrustedAggregator bool              `yaml:"trusted_aggregator"`
	Role              string            `yaml:"role"`
	VendorMap         map[string]string `yaml:"vendor_map"`
}

// LoadScraperSpec reads every harvest/scrapers/*.yaml in dir, keyed by slug.
// Unlike resolve.LoadRuleSet, an unrecognised or missing spec is not a
// startup-fatal error here — harvest/NOTES.md's DEFERRED STORES list is 13
// stores with no *.yaml file yet, by design (PoC scope), and that must stay
// a normal, expected state rather than something that fails the whole
// loader. A caller asking for a specific slug that doesn't have a file gets
// a clear "no such spec" error at the point of use, not at load time.
func LoadScraperSpec(dir string) (map[string]*ScraperSpec, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("ingest.LoadScraperSpec: %w", err)
	}

	specs := make(map[string]*ScraperSpec)
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".yaml" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("ingest.LoadScraperSpec: reading %s: %w", path, err)
		}
		var spec ScraperSpec
		if err := yaml.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("ingest.LoadScraperSpec: parsing %s: %w", path, err)
		}
		if spec.Slug == "" {
			return nil, fmt.Errorf("ingest.LoadScraperSpec: %s has no slug", path)
		}
		specs[spec.Slug] = &spec
	}
	return specs, nil
}
