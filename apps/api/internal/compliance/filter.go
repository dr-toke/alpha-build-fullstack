// Package compliance is the hard-block/review-tier filter —
// 01-ARCHITECTURE.md §6, 04-PIPELINE.md §5. Per ADR-019, this package was a
// placeholder until beta; ADR-020 carved out the first narrow exception
// (service_listing, built when a real live scrape of cbdstore.in surfaced a
// doctor-consultation booking sitting in the product catalogue). This pass
// adds a second: hard_block, built after a full-catalog audit (recorded in
// this session's own findings — see the git log around this file) surfaced
// products making implicit medical claims that nothing was catching. The
// hard_block pattern itself was already fully harvested and specified
// (harvest/rules/compliance.json) — no new pattern invented here, just wired
// up, same evidence-based, narrow-exception discipline as service_listing.
//
// terminology_review, price_anomaly, and unknown_brand are still NOT
// implemented here. See M2-DECISIONS.md.
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

// RuleSet holds only what this slice of compliance needs — service_listing
// and hard_block. Not terminology_review/price_anomaly/unknown_brand,
// which compliance.json also describes; loading those now, unused, would
// misrepresent how much of compliance is actually built.
type RuleSet struct {
	ServiceListing *regexp.Regexp
	HardBlock      *regexp.Regexp
}

type rawCompliance struct {
	ServiceListing struct {
		Pattern string `json:"pattern"`
	} `json:"service_listing"`
	HardBlock struct {
		Pattern string `json:"pattern"`
	} `json:"hard_block"`
}

// LoadRuleSet reads harvest/rules/compliance.json and compiles the
// service_listing and hard_block patterns — same fail-fast-on-missing-key
// discipline as resolve.LoadRuleSet (M1 recheck): a broken or missing
// pattern must error here, at startup, not panic later the first time
// Evaluate runs.
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
	if raw.HardBlock.Pattern == "" {
		return nil, fmt.Errorf("compliance.LoadRuleSet: %s has no hard_block.pattern", path)
	}
	serviceListing, err := regexp.Compile(raw.ServiceListing.Pattern)
	if err != nil {
		return nil, fmt.Errorf("compliance.LoadRuleSet: service_listing pattern: %w", err)
	}
	hardBlock, err := regexp.Compile(raw.HardBlock.Pattern)
	if err != nil {
		return nil, fmt.Errorf("compliance.LoadRuleSet: hard_block pattern: %w", err)
	}
	return &RuleSet{ServiceListing: serviceListing, HardBlock: hardBlock}, nil
}

// Evaluate checks a listing against hard_block (name + description) and
// service_listing (name + category), in that order — hard_block first
// since a prohibited medical claim is the more severe finding of the two
// and short-circuiting avoids a less-informative service_listing detail
// string winning when both would technically match.
//
// hard_block checks name AND description — compliance.json doesn't
// document a `matched_against` scope for this tier the way service_listing
// does ("product NAME only"), and the pattern's own examples ("cures
// cancer", "miracle cure") are exactly the kind of claim that lives in
// marketing description copy, not a product title. Checking both is the
// literal, direct reading of an unscoped pattern, not a widening decision
// the way service_listing's category check was.
//
// service_listing keeps its original name+category scope
// (harvest/rules/compliance.json says "product NAME only" — extended to
// category too per ADR-020's evidence-based finding; see that comment's
// history for the real "Dr. Harshal Sawarkar" case that motivated it).
//
// Word-boundary matching throughout (00-CONSTITUTION.md §4 / the harvested
// patterns themselves), never substring.
//
// Does NOT check price or brand — terminology_review, price_anomaly, and
// unknown_brand are all unevaluated here and always pass. A caller must not
// read a Pass=true result as "compliant" in the full sense the eventual M2
// build will mean — only as "not a prohibited claim or a service/event
// listing."
func Evaluate(rs *RuleSet, name, description, category string) Result {
	combined := name + " " + description
	if m := rs.HardBlock.FindString(combined); m != "" {
		return Result{
			Pass:   false,
			Reason: domain.ReviewComplianceUncertain,
			Detail: "prohibited claim: " + m,
		}
	}
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
