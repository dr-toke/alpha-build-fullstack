-- +goose Up
--
-- M0.5 — 03-DOMAIN-MODEL.md §6 (brands) and §7 (self-checking reference
-- content: states, ROA, aggregators). The self-correction columns
-- (link_status/link_failures/verify_interval_days, `stale` computed at query
-- time) are not re-derived here — they're copied from the prior alpha's own
-- migration 026_reference_content.sql, which 03-DOMAIN-MODEL.md §7
-- retroactively describes ("self-checking") without repeating the schema.
-- Harvested verbatim into harvest/reference/{states,roa,aggregators}.json —
-- this migration's seed data below matches those files exactly.

-- ── Brands ───────────────────────────────────────────────────────────────────

CREATE TABLE brands (
    id                 uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    slug               text        NOT NULL UNIQUE,
    name               text        NOT NULL,
    full_name          text,
    founded            text,                 -- source data is a bare year string ("2013"), not a date
    city               text,
    state              text,
    url                text,
    description        text,
    categories         text[]      NOT NULL DEFAULT '{}',
    verified           boolean     NOT NULL DEFAULT false,   -- drives the pending-verification badge (00-CONSTITUTION.md §3)
    ayush              boolean     NOT NULL DEFAULT false,
    fssai              boolean     NOT NULL DEFAULT false,
    ayush_reg_number   text,
    fssai_licence      text,
    coa_available      boolean     NOT NULL DEFAULT false,
    affiliate_url      text,
    highlight          text,
    last_verified      date        NOT NULL DEFAULT CURRENT_DATE,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE product_clusters
    ADD CONSTRAINT product_clusters_brand_fk
    FOREIGN KEY (brand_id) REFERENCES brands(id);

INSERT INTO brands (slug, name, full_name, founded, city, state, url, description, categories, ayush, fssai, coa_available, highlight, last_verified) VALUES
    ('boheco', 'BOHECO', 'Bombay Hemp Company', '2013', 'Mumbai', 'Maharashtra', 'https://boheco.com', 'India''s pioneering hemp brand, founded 2013 and backed by Ratan Tata. 39+ products across pain, sleep, stress, and nutrition. Ayush-approved by the Madhya Pradesh state department. Retail stores in Delhi-NCR, Bengaluru, and Hyderabad.', ARRAY['oils','nutrition','topicals'], true, true, true, 'Ratan Tata invested', DATE '2025-05-01'),
    ('cannazo-india', 'Cannazo India', 'Cannazo India Pvt. Ltd.', '2020', 'Mumbai', 'Maharashtra', 'https://cannazoindia.com', 'First Indian medical cannabis company to appear on Shark Tank India. Uses nanotechnology for enhanced cannabinoid bioavailability. Government-approved, third-party tested formulations. Prescription-based Vijaya products.', ARRAY['oils','topicals','vijaya'], true, false, true, 'Shark Tank India', DATE '2025-05-01'),
    ('cure-by-design', 'Cure By Design', 'Cure By Design', '2020', 'Bengaluru', 'Karnataka', 'https://curebydesign.in', 'Founded by Daanish Matheen. One of India''s most comprehensive hemp wellness portfolios: CBD oils, gummies, hemp nutrition, Ayurvedic cannabis, and India''s most extensive pet CBD line. Ministry of Ayush licensed.', ARRAY['oils','edibles','topicals','pet'], true, true, true, 'Largest pet CBD range', DATE '2025-05-01'),
    ('indie-extracts', 'Indie Extracts', 'Indie Extracts', '2022', 'Mumbai', 'Maharashtra', 'https://indieextracts.com', 'Founded by Sahil Shivdasani. Built on 40+ years of aromatherapy heritage. Full-spectrum Vijaya leaf extract skincare and wellness products. GMP-certified, third-party tested. Cannabis oils require prescription.', ARRAY['oils','skincare','topicals'], true, false, true, 'Skincare-first approach', DATE '2025-05-01'),
    ('ananta-hemp-works', 'Ananta Hemp Works', 'Ananta Hemp Works', '2019', 'Haridwar', 'Uttarakhand', 'https://itshemp.in/brand/ananta-hemp-works/', 'Uttarakhand-based licensed hemp cultivator. Hemp seeds, protein powder, hemp hearts, and seed oil. One of India''s largest licensed hemp growers. Focused on nutrition and food-grade hemp products.', ARRAY['nutrition','seeds'], false, true, true, 'Licensed farm, Uttarakhand', DATE '2025-05-01'),
    ('awshad', 'Awshad', 'Awshad', '2020', 'Mumbai', 'Maharashtra', 'https://itshemp.in/brand/awshad/', 'Ayurvedic cannabis formulations with clinical backing. CBD oils, topicals, and wellness products. Doctor consultation available on platform. Listed on ItsHemp and CBDStore India.', ARRAY['oils','topicals','vijaya'], true, false, true, 'Doctor consultation available', DATE '2025-05-01'),
    ('health-horizons', 'Health Horizons', 'Health Horizons', '2018', 'Delhi', 'Delhi', 'https://itshemp.in/brand/health-horizons/', 'Wide range of hemp-based nutrition and wellness. Hemp seed oil, protein powder, and pain relief balms. Available across major Indian e-commerce platforms. One of the older FSSAI-compliant hemp nutrition brands.', ARRAY['nutrition','oils','topicals'], false, true, false, 'Wide e-commerce availability', DATE '2025-05-01'),
    ('the-trost', 'The Trost', 'The Trost', '2021', 'Delhi', 'Delhi', 'https://thetrost.com', 'Cannabis-based Ayurvedic formulations focused on education alongside product sales. Strong editorial blog covering legal guides and harm reduction. Prescription Vijaya products available.', ARRAY['oils','vijaya'], true, false, true, 'Strong educational content', DATE '2025-05-01'),
    ('hemp-tribe', 'Hemp Tribe', 'Hemp Tribe', '2019', 'Dharamsala', 'Himachal Pradesh', 'https://itshemp.in/brand/hemp-tribe/', 'Himachal Pradesh-sourced hemp nutrition. Seeds, hemp hearts, protein, and oil. Community-driven brand working directly with mountain farming communities. Transparent farm-to-product sourcing.', ARRAY['nutrition','seeds'], false, true, true, 'Direct farmer sourcing', DATE '2025-05-01'),
    ('noigra', 'Noigra', 'Noigra', '2020', 'Bengaluru', 'Karnataka', 'https://itshemp.in/brand/noigra/', 'Hemp-derived CBD and wellness products. Tinctures, topicals, and gummies. Third-party lab tested. COA available on request. Ayush-registered formulations.', ARRAY['oils','edibles','topicals'], true, false, true, 'COA on request', DATE '2025-05-01'),
    ('wholeleaf', 'Wholeleaf', 'Wholeleaf', '2021', 'Delhi', 'Delhi', 'https://itshemp.in/brand/wholeleaf/', 'Full-spectrum and broad-spectrum Vijaya formulations. Focused on clinical conditions with doctor-guided protocols. Prescription model. One of the few brands offering high-potency Vijaya extracts.', ARRAY['oils','vijaya'], true, false, true, 'Doctor-guided protocols', DATE '2025-05-01'),
    ('hempbuti', 'Hempbuti', 'Hempbuti', '2020', 'Dehradun', 'Uttarakhand', 'https://itshemp.in/brand/hempbuti/', 'Uttarakhand hemp nutrition brand. Affordable hemp seeds, seed oil, and protein powder. Good entry-level choice for consumers new to hemp nutrition. FSSAI compliant.', ARRAY['nutrition','seeds'], false, true, false, 'Budget-accessible', DATE '2025-05-01');

-- ── States ───────────────────────────────────────────────────────────────────

CREATE TABLE states (
    id                  uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    slug                text        UNIQUE NOT NULL,
    name                text        NOT NULL,
    status              text        NOT NULL CHECK (status IN ('legal','tolerated','grey','limited','illegal')),
    bhang_shops         text        NOT NULL DEFAULT '0',
    detail              text        NOT NULL,
    excise_url          text,
    notes               text,
    featured            boolean     NOT NULL DEFAULT false,
    display_order       int         NOT NULL DEFAULT 100,
    last_verified       date        NOT NULL DEFAULT CURRENT_DATE,
    verify_interval_days int        NOT NULL DEFAULT 180,
    link_status         text        NOT NULL DEFAULT 'unknown'
                          CHECK (link_status IN ('unknown','ok','redirect','broken','no_url')),
    link_checked_at     timestamptz,
    link_failures       int         NOT NULL DEFAULT 0,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);

INSERT INTO states
    (slug, name, status, bhang_shops, detail, excise_url, notes, featured, display_order, last_verified)
VALUES
    ('delhi-ncr', 'Delhi NCR', 'legal', 'Limited',
     'Bhang licensed under Delhi Excise. Government-authorized shops in older neighborhoods near major temples. Year-round availability. Hemp wellness products sold separately via Ayush brand stockists.',
     'https://excise.delhi.gov.in', NULL, true, 0, DATE '2025-05-01'),
    ('uttar-pradesh', 'Uttar Pradesh', 'legal', '100+',
     'Government bhang shops across Varanasi, Lucknow, Allahabad, and all major pilgrimage towns. Highest density of legal bhang retail in India. Excise-licensed vendors source from government godowns only.',
     'https://excise.up.gov.in', NULL, false, 10, DATE '2025-05-01'),
    ('rajasthan', 'Rajasthan', 'legal', '80+',
     'Widely available in Jaipur, Jaisalmer, Pushkar, and Jodhpur. Government-licensed bhang lassi shops integrated with tourism culture. Holi season sees peak availability and consumption.',
     NULL, NULL, false, 20, DATE '2025-05-01'),
    ('madhya-pradesh', 'Madhya Pradesh', 'legal', '50+',
     'Legal under government regulation. Licensed bhang shops in multiple cities. Also the state that approved BOHECO''s Ayush registration — most advanced state for hemp Ayush licensing.',
     NULL, NULL, false, 30, DATE '2025-05-01'),
    ('uttarakhand', 'Uttarakhand', 'legal', 'Pilot',
     'First state to license industrial hemp cultivation commercially (2017). Below 0.3% THC threshold. Most progressive state for hemp innovation — multiple brands grow and manufacture here. Growing wellness retail.',
     NULL, NULL, false, 40, DATE '2025-05-01'),
    ('odisha', 'Odisha', 'legal', '30+',
     'Long tradition of legal cannabis consumption including bhang. Government-regulated shops across multiple cities. State excise licenses both production and sale.',
     NULL, NULL, false, 50, DATE '2025-05-01'),
    ('himachal-pradesh', 'Himachal Pradesh', 'tolerated', 'Traditional',
     'Traditional consumption tolerated in mountain communities with deep cultural heritage. Hemp cultivation being explored for industrial and medicinal use. Not formally regulated like UP or Rajasthan.',
     NULL, NULL, false, 60, DATE '2025-05-01'),
    ('maharashtra', 'Maharashtra', 'grey', '0',
     'Hemp CBD products in regulatory grey zone. No licensed bhang retail. Industrial hemp import allowed. CBD wellness brands operate via Ayush framework. NDPS enforcement varies significantly by district.',
     NULL,
     'Several major brands (BOHECO, Cannazo, Cure By Design, Indie Extracts) are headquartered here despite the grey zone.',
     false, 70, DATE '2025-05-01'),
    ('karnataka', 'Karnataka', 'grey', '0',
     'Growing wellness retail presence in Bengaluru via Ayush-registered brands. No bhang licensing. Enforcement inconsistent. Multiple hemp brands headquartered in Bengaluru.',
     NULL, NULL, false, 80, DATE '2025-05-01'),
    ('punjab', 'Punjab', 'limited', 'Festival only',
     'Bhang thandai sold in some licensed stores during Holi and Maha Shivaratri only. Not available year-round. Stricter NDPS enforcement outside festival periods.',
     NULL, NULL, false, 90, DATE '2025-05-01');

UPDATE states SET link_status = 'no_url' WHERE excise_url IS NULL;
CREATE INDEX states_order_idx ON states(display_order, name);

-- ── Routes of administration ────────────────────────────────────────────────

CREATE TABLE roa_methods (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    slug            text        UNIQUE NOT NULL,
    method          text        NOT NULL,
    onset           text        NOT NULL,
    duration        text        NOT NULL,
    bioavailability text        NOT NULL,
    pros            text[]      NOT NULL DEFAULT '{}',
    cons            text[]      NOT NULL DEFAULT '{}',
    best_for        text[]      NOT NULL DEFAULT '{}',
    warning_note    text,
    display_order   int         NOT NULL DEFAULT 100,
    last_verified   date        NOT NULL DEFAULT CURRENT_DATE,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

INSERT INTO roa_methods
    (slug, method, onset, duration, bioavailability, pros, cons, best_for, warning_note, display_order)
VALUES
    ('smoking-vaping', 'Smoking / Vaping', '1–5 min', '1–3 hrs', '~30%',
     ARRAY['Fastest onset of any method','Real-time dose titration','Immediate relief assessment'],
     ARRAY['Lung irritation over time','Short duration requires redosing','Social stigma in India','Smell'],
     ARRAY['Acute pain episodes','Experienced users','Situations needing rapid onset'],
     'Combustion creates byproducts. Vaping is lower harm than smoking but not zero harm.', 10),
    ('edibles-capsules', 'Edibles / Capsules', '30–120 min', '4–8 hrs', '~6–20%',
     ARRAY['Longest duration of any method','Discreet, no smell','No lung involvement','Precise dosing with capsules'],
     ARRAY['Very slow onset — #1 cause of accidental overconsumption','Variable absorption affected by food intake','Liver-processed, producing 11-OH-THC (stronger effect)','Hard to reverse if too much taken'],
     ARRAY['Chronic pain management','Sleep disorders','Long-duration sustained relief','Daytime microdosing'],
     'GOLDEN RULE: Start at 2.5–5mg. Wait 2 full hours before considering more. Never double-dose because ''nothing happened''.', 20),
    ('sublingual-oil', 'Sublingual Oil / Tincture', '15–45 min', '2–6 hrs', '~20–35%',
     ARRAY['Faster than edibles','Drop-by-drop dose control','No combustion','Consistent absorption'],
     ARRAY['Taste can be unpleasant (carrier oil dependent)','Requires 60–90 seconds held under tongue','Dose adjustment takes 30+ minutes to assess'],
     ARRAY['Daily wellness regimen','Anxiety management','Pain management','Clinical use with physician oversight'],
     NULL, 30),
    ('topical', 'Topical (Balm / Cream)', '15–60 min', '2–4 hrs', 'Local only',
     ARRAY['Zero psychoactive effect','Targeted local relief','Completely beginner-safe','No systemic risk'],
     ARRAY['Does not enter bloodstream','Limited to application site only','Ineffective for systemic conditions'],
     ARRAY['Joint and muscle pain','Post-workout recovery','Skin conditions','Arthritis','Anyone wanting zero psychoactivity'],
     NULL, 40),
    ('beverages', 'Beverages (Bhang / Thandai)', '45–120 min', '4–10 hrs', '~6–20%',
     ARRAY['Deeply embedded in Indian cultural tradition','Social consumption context','Accessible at legal government shops','Often fat-infused (higher absorption)'],
     ARRAY['Hardest to dose accurately of all methods','Potency varies wildly between vendors and batches','Festival context encourages overconsumption','Very long duration can catch people off-guard'],
     ARRAY['Cultural and festival context','Experienced users in social settings'],
     'Half a glass maximum for beginners. Bhang potency varies enormously — government shop bhang is typically weaker than private preparations. Always eat beforehand.', 50);

CREATE INDEX roa_methods_order_idx ON roa_methods(display_order);

-- ── Aggregators ──────────────────────────────────────────────────────────────

CREATE TABLE aggregators (
    id                   uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    slug                 text        UNIQUE NOT NULL,
    name                 text        NOT NULL,
    url                  text        NOT NULL,
    description          text        NOT NULL,
    source_slug          text        REFERENCES scrape_sources(slug),
    brand_count_label    text,
    product_count_label  text,
    derived_brand_count   int,
    derived_product_count int,
    featured             boolean     NOT NULL DEFAULT false,
    display_order        int         NOT NULL DEFAULT 100,
    active               boolean     NOT NULL DEFAULT true,
    last_verified        date        NOT NULL DEFAULT CURRENT_DATE,
    verify_interval_days int         NOT NULL DEFAULT 180,
    link_status          text        NOT NULL DEFAULT 'unknown'
                          CHECK (link_status IN ('unknown','ok','redirect','broken','no_url')),
    link_checked_at      timestamptz,
    link_failures        int         NOT NULL DEFAULT 0,
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now()
);

-- source_slug FKs to scrape_sources — only cbdstore exists this pass (PoC
-- scope, harvest/NOTES.md). itshemp/cannameds rows are seeded with a NULL
-- source_slug for now and should be re-pointed once those scrapers are built;
-- NULL is valid (source_slug is nullable) so this doesn't block the seed.
INSERT INTO aggregators
    (slug, name, url, description, source_slug, brand_count_label, product_count_label, display_order)
VALUES
    ('itshemp', 'ItsHemp', 'https://itshemp.in',
     'India''s largest hemp and cannabis marketplace, spanning oils, edibles, topicals, and wellness.',
     NULL, '80+', '3500+', 10),
    ('cbdstore-india', 'CBD Store India', 'https://cbdstore.in',
     'Medical cannabis focus. Prescription-based product sales. Wide brand selection including Cannabryl, Magiccann, Cannazo.',
     'cbdstore', NULL, NULL, 20),
    ('cannameds-india', 'CannaMeds India', 'https://cannameds.in',
     'Medical cannabis products with doctor consultation services integrated.',
     NULL, NULL, NULL, 30);

CREATE INDEX aggregators_order_idx ON aggregators(display_order) WHERE active = true;

-- +goose Down
DROP TABLE aggregators;
DROP TABLE roa_methods;
DROP TABLE states;
ALTER TABLE product_clusters DROP CONSTRAINT product_clusters_brand_fk;
DROP TABLE brands;
