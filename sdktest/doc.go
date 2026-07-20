// Package sdktest is the SDK's serialization test suite: coverage upstream
// (gowsdl-generated google-admanager-api-go) entirely lacks, grounded in wire
// shapes confirmed against a live GAM test network.
//
// It guards two things only:
//
//  1. Known gowsdl codegen defects that scripts/generate.sh patches after
//     generation (e.g. embedded-field name collisions, DateTime/Date fields
//     mistyped as flat scalars). If a future `make generate` regenerates
//     services/ without carrying a fix forward, these tests fail to compile
//     or fail to pass — a loud, immediate signal that the corresponding
//     generate.sh step regressed.
//  2. The serialization contract for the SOAP fragments this fork's callers
//     actually use: constructing a type and xml.Marshaling it must produce
//     the shape the live GAM SOAP endpoint accepts, as confirmed by a probe
//     that made real calls against a GAM test network.
//
// This package lives outside services/ deliberately: `make clean` deletes
// services/ and `make generate` regenerates it from scratch, so any test
// living inside would be wiped on every regen. Living outside services/ also
// means these tests reference the fixed generated types directly, so a
// regeneration that drops a fix (e.g. reintroducing the embedded *Value
// conflict) makes this package fail to compile — and avoids the import
// cycle a test package inside the root admanager package would hit when
// importing the generated service packages.
//
// Evidence behind the fixes and shapes asserted here traces back to the adcp
// repo's gam-p0-findings.md (the live probe run and the faults it uncovered).
package sdktest
