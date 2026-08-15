// Package compliance is the hard-block/review-tier filter —
// 01-ARCHITECTURE.md §6, 04-PIPELINE.md §5. Per ADR-019, this package was a
// placeholder until beta; ADR-020 carves out ONE narrow exception:
// service_listing detection, built now because a real live scrape of
// cbdstore.in surfaced a doctor-consultation booking sitting in the product
// catalogue, and staging that kind of thing even into a not-yet-live
// pipeline was unacceptable to leave unaddressed.
//
// Everything else compliance.json describes — hard_block, terminology_review,
// price_anomaly, unknown_brand — is NOT implemented here. Evaluate only
// checks service_listing. See M2-DECISIONS.md.
package compliance

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/dr-toke/api/internal/domain"
)

// Result is Evaluate's verdict. Reason uses the same domain.ReviewReason
// vocabulary the full compliance filter will use later
// (compliance_uncertain, per harvest/rules/compliance.json's service_listing
// action) — so this doesn't need a signature change when the rest of M2
// eventually gets built, just more populated branches.
type Result struct {
	Pass   bool
	Reason domain.ReviewReason
	Detail string
}

// RuleSet holds only what this narrow slice of compliance needs — the
// service_listing pattern. Not the four other tiers compliance.json also
// describes; loading those now, unused, would misrepresent how much of
// compliance is actually built.
type RuleSet struct {
	ServiceListing *regexp.Regexp
}

type rawCompliance struct {
	ServiceListing struct {
		Pattern string `json:"pattern"`
	} `json:"service_listing"`
}

// LoadRuleSet reads harvest/rules/compliance.json and compiles ONLY the
// service_listing pattern — same fail-fast-on-missing-key discipline as
// resolve.LoadRuleSet (M1 recheck): a broken or missing pattern must error
// here, at startup, not panic later the first time Evaluate runs.
func LoadRuleSet(path string) (*RuleSet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("compliance.LoadRuleSet: %w", err)
	}
	var raw rawCompliance
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("compliance.LoadRuleSet: parsing %s: %w", path, err)
	}
	if raw.ServiceListing.Pattern == "" {
		return nil, fmt.Errorf("compliance.LoadRuleSet: %s has no service_listing.pattern", path)
	}
	re, err := regexp.Compile(raw.ServiceListing.Pattern)
	if err != nil {
		return nil, fmt.Errorf("compliance.LoadRuleSet: service_listing pattern: %w", err)
	}
	return &RuleSet{ServiceListing: re}, nil
}

// Evaluate checks a listing's name AND category against the
// service_listing pattern. harvest/rules/compliance.json says
// "matched_against: product NAME only" — but the motivating real example
// for building this ("Dr. Harshal Sawarkar – BAMS Ayurvedic Physician...",
// a live cbdstore.in listing) proves that's not sufficient on its own: its
// TITLE contains none of the pattern's keywords at all. What DOES carry the
// signal, confirmed by fetching the real listing directly, is Shopify's own
// product_type field: "Doctors Consultation" — mapped through the ingest
// pipeline into RawListing.CategoryRaw / domain.RawProduct.CategoryRaw.
//
// So this checks category too, word-boundary, same pattern — a deliberate,
// evidence-based extension beyond what the harvested doc specifies, not a
// silent one. If checking name-only turns out to be the right call for
// other stores' data (some may not populate product_type usefully), that's
// a real design question for the full M2 build, not resolved here.
//
// Word-boundary matching throughout (00-CONSTITUTION.md §4 / the harvested
// pattern itself), never substring.
//
// Does NOT check description, price, or brand — hard_block,
// terminology_review, price_anomaly, and unknown_brand are all unevaluated
// here and always pass. A caller must not read a Pass=true result from this
// function as "compliant" in the full sense the eventual M2 build will
// mean — only as "not a service/event listing, by name or category."
func Evaluate(rs *RuleSet, name, category string) Result {
	if m := rs.ServiceListing.FindString(name); m != "" {
		return Result{
			Pass:   false,
			Reason: domain.ReviewComplianceUncertain,
			Detail: "service/event listing (name), not a product: " + m,
		}
	}
	if m := rs.ServiceListing.FindString(category); m != "" {
		return Result{
			Pass:   false,
			Reason: domain.ReviewComplianceUncertain,
			Detail: "service/event listing (category), not a product: " + m,
		}
	}
	return Result{Pass: true}
}
