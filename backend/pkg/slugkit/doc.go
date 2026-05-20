// Package slugkit is the single source of truth for AgentsMesh identifier
// fields (slug / username / pod_key / handle). All UNIQUE string columns
// used as URL params, mention keys, or lookup identifiers MUST conform to
// the slugkit rule: lowercase letters/digits with hyphens between segments,
// length 2-100, and not in the Reserved set.
//
// # Layers of defense
//
//  1. DB CHECK constraint enforces the rule at the database (Phase 2/4 migrations).
//  2. GORM BeforeSave hook (slugkit.ValidateIdentifier) catches direct ORM writes.
//  3. Per-table *Registry service helpers (e.g. user.EnsureUniqueUsername) are
//     the ONLY sanctioned write path; they wrap GenerateUnique + collision retry.
//  4. Slug newtype protects new identifier fields at compile time.
//  5. CI lint blocks raw string assignments to identifier columns.
//
// # How to add a new identifier field (checklist)
//
//  1. In migrations, declare the column as VARCHAR(100) NOT NULL with a CHECK
//     constraint matching slugkit's pattern. Use the two-step pattern
//     (NOT VALID → backfill → VALIDATE) if the table has existing rows.
//
//  2. In the domain model, type the field as `slugkit.Slug` (preferred for
//     new fields) or `string` (acceptable when interoperating with legacy
//     callers). Add a `BeforeSave` hook calling slugkit.ValidateIdentifier.
//
//  3. Add a service-layer `*Registry` helper that wraps
//     `slugkit.GenerateUnique` with a uniqueness check against the table.
//     All external-data ingress paths MUST funnel through this helper.
//
//  4. Add a regression test feeding non-compliant input (dots, uppercase,
//     unicode) and asserting the field stored passes slugkit.Validate.
package slugkit
