package resolve

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// RuleSet bundles every compiled pattern M1 needs. Loaded once at process
// start (00-CONSTITUTION.md §6: "no classification pattern... is ever
// hardcoded in Go. They load from harvest/") and passed into
// ExtractCannabinoids / ResolveForm / etc. as a parameter — never a package
// global — so tests can load a fixture ruleset independent of the real
// harvest/ directory and so a rule change never requires a recompile.
//
// Scoped to what internal/resolve actually consumes: cannabinoids.json and
// categories.json. compliance.json is NOT loaded here — internal/compliance
// (M2) is a separate package per 01-ARCHITECTURE.md §6's package boundaries,
// and it loads its own rules rather than depending on internal/resolve for
// them. harvest/rules/dedup.md isn't JSON and belongs to internal/ingest's
// dedup logic (M4), not this loader.
type RuleSet struct {
	Cannabinoids CannabinoidRuleSet
	Categories   CategoryRuleSet
}

// CannabinoidRuleSet is every named regex from harvest/rules/cannabinoids.json's
// "patterns" object, compiled. See that file's "notes" object (not loaded
// here — it's harvest-file documentation, not runtime data) for why each one
// exists; internal/resolve/cannabinoids.go cites the pattern names directly.
type CannabinoidRuleSet struct {
	Patterns map[string]*regexp.Regexp
}

// CategoryRuleSet is harvest/rules/categories.json, compiled. Word lists are
// compiled into word-boundary alternations at load time (once), not
// recompiled per match — mirrors the harvested source's package-var-init-time
// wordSet() calls.
type CategoryRuleSet struct {
	// FormWordLists is keyed by form name (e.g. "edible_solid", "topical",
	// "vapeable" — see categories.json's form_word_lists keys).
	FormWordLists map[string]*regexp.Regexp

	PetWordsName        *regexp.Regexp
	PetStrongDescPhrase *regexp.Regexp
	PetWarningStrip     *regexp.Regexp
	ApparelWords        *regexp.Regexp
	NutritionWords      *regexp.Regexp
	CannabinoidContext  *regexp.Regexp
	ConcentrateMarkers  *regexp.Regexp
	NegationPrimary     *regexp.Regexp
	NegationFormFree    *regexp.Regexp

	CoherenceMatrix CoherenceMatrix

	// FormPriority orders detected forms most-specific-first; the first
	// present becomes the primary. categories.json: form_priority_primary_selection.
	FormPriority []string

	// Exclusive facet values that can never combine with anything else
	// (pet, apparel). categories.json: exclusive.
	Exclusive []string

	// SecondaryImplications: a facet value that always also implies a
	// legacy secondary category — e.g. extract -> [edible].
	// categories.json: secondary_implications.
	SecondaryImplications map[string][]string
}

// CoherenceMatrix is categories.json's coherence_matrix, compiled into plain
// string sets for fast membership checks in facets.go.
type CoherenceMatrix struct {
	IfTopicalDelete                 map[string]bool
	IfEdibleSolidDelete             map[string]bool
	IfTinctureDelete                map[string]bool
	IfVapeableAndNameSaysVapeDelete map[string]bool
}

// ── On-disk JSON shapes (unexported — this file is the only thing that
// needs to know cannabinoids.json / categories.json's literal structure) ──

type rawCannabinoids struct {
	Patterns map[string]string `json:"patterns"`
}

type rawCategories struct {
	FormWordLists              map[string][]string `json:"form_word_lists"`
	PetWordsNameLevel          []string            `json:"pet_words_name_level"`
	PetStrongDescriptionPhrase string              `json:"pet_strong_description_phrase"`
	PetWarningSentenceStrip    string              `json:"pet_warning_sentence_strip"`
	ApparelWords               []string            `json:"apparel_words"`
	NutritionWords             []string            `json:"nutrition_words"`
	CannabinoidContextWords    []string            `json:"cannabinoid_context_words"`
	ConcentrateMarkers         string              `json:"concentrate_markers"`
	NegationStripPattern       string              `json:"negation_strip_pattern"`
	NegationStripPattern2      string              `json:"negation_strip_pattern_2"`
	CoherenceMatrix            struct {
		IfTopicalDelete                 []string `json:"if_topical_delete"`
		IfEdibleSolidDelete             []string `json:"if_edible_solid_delete"`
		IfTinctureDelete                []string `json:"if_tincture_delete"`
		IfVapeableAndNameSaysVapeDelete []string `json:"if_vapeable_and_name_says_vape_delete"`
	} `json:"coherence_matrix"`
	FormPriorityPrimarySelection []string            `json:"form_priority_primary_selection"`
	Exclusive                    []string            `json:"exclusive"`
	SecondaryImplications        map[string][]string `json:"secondary_implications"`
}

// LoadRuleSet reads cannabinoids.json and categories.json from dir
// (harvest/rules/) and compiles every pattern. Returns an error naming the
// file and pattern on any bad regex — a broken harvested pattern must fail
// loudly at startup, never silently match nothing.
func LoadRuleSet(dir string) (*RuleSet, error) {
	cb, err := loadCannabinoids(filepath.Join(dir, "cannabinoids.json"))
	if err != nil {
		return nil, fmt.Errorf("resolve.LoadRuleSet: %w", err)
	}
	cat, err := loadCategories(filepath.Join(dir, "categories.json"))
	if err != nil {
		return nil, fmt.Errorf("resolve.LoadRuleSet: %w", err)
	}
	return &RuleSet{Cannabinoids: *cb, Categories: *cat}, nil
}

func loadCannabinoids(path string) (*CannabinoidRuleSet, error) {
	var raw rawCannabinoids
	if err := readJSON(path, &raw); err != nil {
		return nil, err
	}
	patterns := make(map[string]*regexp.Regexp, len(raw.Patterns))
	for name, pat := range raw.Patterns {
		re, err := regexp.Compile(pat)
		if err != nil {
			return nil, fmt.Errorf("%s: pattern %q: %w", path, name, err)
		}
		patterns[name] = re
	}
	for _, req := range requiredCannabinoidPatterns {
		if _, ok := patterns[req]; !ok {
			return nil, fmt.Errorf("%s: missing required pattern %q", path, req)
		}
	}
	return &CannabinoidRuleSet{Patterns: patterns}, nil
}

// requiredCannabinoidPatterns and requiredFormWordLists exist so a
// harvest/rules/*.json edit that drops a key cannabinoids.go/facets.go
// depends on fails LOUDLY at LoadRuleSet time — a fmt.Errorf naming exactly
// which key is missing — instead of failing SILENTLY later as a nil
// *regexp.Regexp panic the first time classify() or ExtractCannabinoids
// happens to reach that specific code path. Found during a recheck pass:
// rs.Patterns["x"] and rs.FormWordLists["x"] are indexed directly
// throughout cannabinoids.go and facets.go with no per-call nil check —
// correct and appropriately terse AS LONG AS the loader guarantees every
// key those two files reference actually exists. This is that guarantee,
// enforced once, at the one place a missing key can still be caught cleanly.
var requiredCannabinoidPatterns = []string{
	"cbd_label_first", "cbd_num_first", "thc_label_first", "thc_num_first",
	"pct_cbd", "pct_thc", "volume_ml", "weight_g", "generic_mg", "hemp_seed",
	"ratio_num_first", "ratio_label_first", "thc_dominant_wording", "ratio_bare",
	"chem_name_cannabidiol", "chem_name_thc", "thc_free", "trace_thc", "cannabinoid_pair",
}

var requiredFormWordLists = []string{
	"edible_solid", "edible", "topical", "smokable", "vapeable", "tincture", "beverage", "extract",
}

func loadCategories(path string) (*CategoryRuleSet, error) {
	var raw rawCategories
	if err := readJSON(path, &raw); err != nil {
		return nil, err
	}

	formLists := make(map[string]*regexp.Regexp, len(raw.FormWordLists))
	for form, words := range raw.FormWordLists {
		formLists[form] = wordSet(words)
	}

	petPhrase, err := regexp.Compile(raw.PetStrongDescriptionPhrase)
	if err != nil {
		return nil, fmt.Errorf("%s: pet_strong_description_phrase: %w", path, err)
	}
	petWarning, err := regexp.Compile(raw.PetWarningSentenceStrip)
	if err != nil {
		return nil, fmt.Errorf("%s: pet_warning_sentence_strip: %w", path, err)
	}
	concentrate, err := regexp.Compile(raw.ConcentrateMarkers)
	if err != nil {
		return nil, fmt.Errorf("%s: concentrate_markers: %w", path, err)
	}
	negation1, err := regexp.Compile(raw.NegationStripPattern)
	if err != nil {
		return nil, fmt.Errorf("%s: negation_strip_pattern: %w", path, err)
	}
	negation2, err := regexp.Compile(raw.NegationStripPattern2)
	if err != nil {
		return nil, fmt.Errorf("%s: negation_strip_pattern_2: %w", path, err)
	}
	for _, req := range requiredFormWordLists {
		if _, ok := formLists[req]; !ok {
			return nil, fmt.Errorf("%s: missing required form_word_lists entry %q", path, req)
		}
	}

	return &CategoryRuleSet{
		FormWordLists:       formLists,
		PetWordsName:        wordSet(raw.PetWordsNameLevel),
		PetStrongDescPhrase: petPhrase,
		PetWarningStrip:     petWarning,
		ApparelWords:        wordSet(raw.ApparelWords),
		NutritionWords:      wordSet(raw.NutritionWords),
		CannabinoidContext:  wordSet(raw.CannabinoidContextWords),
		ConcentrateMarkers:  concentrate,
		NegationPrimary:     negation1,
		NegationFormFree:    negation2,
		CoherenceMatrix: CoherenceMatrix{
			IfTopicalDelete:                 toSet(raw.CoherenceMatrix.IfTopicalDelete),
			IfEdibleSolidDelete:             toSet(raw.CoherenceMatrix.IfEdibleSolidDelete),
			IfTinctureDelete:                toSet(raw.CoherenceMatrix.IfTinctureDelete),
			IfVapeableAndNameSaysVapeDelete: toSet(raw.CoherenceMatrix.IfVapeableAndNameSaysVapeDelete),
		},
		FormPriority:          raw.FormPriorityPrimarySelection,
		Exclusive:             raw.Exclusive,
		SecondaryImplications: raw.SecondaryImplications,
	}, nil
}

// wordSet compiles a case-insensitive, word-boundary alternation — identical
// in shape to the harvested source's own wordSet() helper
// (dr-toke-init/apps/api/internal/catalog/normaliser/categories.go), so a
// harvested word list behaves exactly as it did there.
func wordSet(words []string) *regexp.Regexp {
	quoted := make([]string, len(words))
	for i, w := range words {
		quoted[i] = regexp.QuoteMeta(w)
	}
	return regexp.MustCompile(`(?i)\b(?:` + strings.Join(quoted, "|") + `)\b`)
}

func toSet(items []string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, it := range items {
		m[it] = true
	}
	return m
}

func readJSON(path string, v any) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(v); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}
