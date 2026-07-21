#!/usr/bin/env bash
set -euo pipefail

# Go module name — must match go.mod
MODULE="github.com/code4ward/google-admanager-api-go"

# ---------- Configuration ----------
# Add or remove versions / services here.

VERSIONS=(
  "v202602"
  "v202505"
)

SERVICES=(
  "NetworkService"
  "OrderService"
  "LineItemService"
  "CompanyService"
  "CreativeService"
  "CreativeSetService"
  "InventoryService"
  "ReportService"
  "UserService"
  "ForecastService"
  "CustomTargetingService"
  "LineItemCreativeAssociationService"
  "PlacementService"
  "PublisherQueryLanguageService"
  "NativeStyleService"
)

# ---------- Helpers ----------

# Convert CamelCase to snake_case.
to_snake_case() {
  echo "$1" \
    | sed -E 's/([A-Z])/_\1/g' \
    | sed 's/^_//' \
    | tr '[:upper:]' '[:lower:]'
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
TEMPLATE_DIR="${SCRIPT_DIR}/templates"

# Fix structs where an embedded type name collides with a field name.
# gowsdl generates e.g.:
#   type BooleanValue struct {
#       *Value
#       Value bool `xml:"value,omitempty"...`
#   }
# Go rejects this because *Value and field Value share the same name.
# The base Value type is always an empty struct, so dropping the embed is safe.
fix_embedded_conflicts() {
  local file="$1"
  # perl -i (not `sed -i`) for portability: BSD sed requires an extension arg
  # after -i and does not treat \t as a tab, so `sed -i '/^\t\*Value$/d'` fails
  # on macOS. perl's -i and \t behave identically on macOS and Linux.
  perl -i -ne 'print unless /^\t\*Value$/' "$file"
  echo "  Fixed embedded *Value conflicts in $(basename "$file")"
}

# Retype GAM's DateTime/Date fields from the flat scalars gowsdl emits
# (soap.XSDDateTime / soap.XSDDate) to the generated element-only complex
# structs (*DateTime / *Date). GAM's WSDL declares these as complexTypes, but
# gowsdl maps them to xsd scalars; the live SOAP endpoint then rejects the flat
# wire shape with a "cvc-complex-type.2.3 ... element-only" fault. The complex
# DateTime/Date structs ARE generated (just never referenced as field types),
# so the retype is safe. Guarded on the struct existing in the file, so a
# package that lacks the complex type is left untouched.
# Verified against the live test network; regression-guarded by the sdktest
# package (see /gam-p0-findings.md #1). Uses perl -i for the same BSD/GNU
# portability reason as fix_embedded_conflicts above.
fix_datetime_types() {
  local file="$1"
  grep -q '^type DateTime struct' "$file" && perl -i -pe 's/\bsoap\.XSDDateTime\b/*DateTime/g' "$file"
  grep -q '^type Date struct' "$file" && perl -i -pe 's/\bsoap\.XSDDate\b/*Date/g' "$file"
  gofmt -w "$file"
  echo "  Retyped DateTime/Date fields to complex types in $(basename "$file")"
}

# render invokes the Go template renderer (scripts/gen) from the module root so
# that go run resolves the ./scripts/gen package. generate.sh emits no Go source
# itself; all generated Go comes from scripts/templates/*.tmpl via this helper.
render() {
  ( cd "${ROOT_DIR}" && go run ./scripts/gen "$@" )
}

# derive_child_types prints the comma-separated concrete leaf subtypes of the
# abstract CustomCriteriaNode in the given generated file: the structs that embed
# *CustomCriteriaLeaf (CustomCriteria / CmsMetadataCriteria / AudienceSegmentCriteria).
# Derived per package rather than hardcoded, so a schema change (a new or removed
# criterion type) is picked up automatically instead of silently diverging.
# CustomCriteriaSet is the set type itself and is handled by the template, so it
# is intentionally not in this list.
derive_child_types() {
  awk '/^type .* struct/{name=$2} /^\t\*CustomCriteriaLeaf$/{print name}' "$1" | paste -sd, -
}

# Fix GAM's custom-targeting tree (CustomCriteriaSet.children). The children are
# an abstract CustomCriteriaNode in the WSDL: every child must name its concrete
# subtype (CustomCriteria / AudienceSegmentCriteria / nested CustomCriteriaSet)
# via an xsi:type attribute. gowsdl emits `Children []*CustomCriteriaNode` (an
# empty base struct) with no xsi:type, so a concrete node can neither be stored
# in the slice nor serialized in the shape GAM accepts. This retypes Children to
# the CustomCriteriaChild interface (a patch, since it mutates a generated field)
# and renders the xsi:type-emitting marshalers into a sibling custom_criteria_ext.go
# — the same class of hand-authored fix as actions.go's xsiTypedAction for abstract
# *Action args.
#
# Applied to EVERY package that declares CustomCriteriaSet, not just line_item:
# request-side Targeting.CustomTargeting reaches CustomCriteriaSet through several
# services (e.g. forecast_service via GetAvailabilityForecast -> ProspectiveLineItem
# -> LineItem -> Targeting.CustomTargeting), so any of them can need to serialize
# concrete criteria. The template makes the extra packages free to cover, so there
# is no reason to leave a latent gap. Verified against the live test network;
# regression-guarded by the sdktest package (see /gam-p0-findings.md).
fix_custom_criteria() {
  local file="$1" pkg="$2" dir="$3"
  grep -q '^type CustomCriteriaSet struct' "$file" || return 0
  local child_types
  child_types=$(derive_child_types "$file")
  if [[ -z "${child_types}" ]]; then
    echo "  ERROR: ${pkg} has CustomCriteriaSet but no *CustomCriteriaLeaf subtypes" >&2
    return 1
  fi
  perl -i -pe 's/Children \[\]\*CustomCriteriaNode/Children []CustomCriteriaChild/' "$file"
  gofmt -w "$file"
  render \
    -template "${TEMPLATE_DIR}/custom_criteria_ext.go.tmpl" \
    -out "${dir}/custom_criteria_ext.go" \
    -package "${pkg}" \
    -child-types "${child_types}"
  echo "  Fixed CustomCriteria xsi:type polymorphism in ${pkg} (retype + custom_criteria_ext.go)"
}

# ---------- Generate ----------

for version in "${VERSIONS[@]}"; do
  version_dir="${ROOT_DIR}/services/${version}"
  mkdir -p "${version_dir}"

  # --- Generate gowsdl service clients ---
  for service in "${SERVICES[@]}"; do
    pkg=$(to_snake_case "${service}")

    echo "Generating ${service} for ${version} (package: ${pkg})..."
    go tool gowsdl \
      -d "${version_dir}" \
      -o "${pkg}.go" \
      -p "${pkg}" \
      "https://ads.google.com/apis/ads/publisher/${version}/${service}?wsdl"

    generated_file="${version_dir}/${pkg}/${pkg}.go"
    if [[ -f "${generated_file}" ]]; then
      fix_embedded_conflicts "${generated_file}"
      fix_datetime_types "${generated_file}"
      fix_custom_criteria "${generated_file}" "${pkg}" "${version_dir}/${pkg}"
    fi
  done

  # --- Generate per-version client.go with service wrappers ---
  services_csv=$(IFS=,; echo "${SERVICES[*]}")
  render \
    -template "${TEMPLATE_DIR}/client.go.tmpl" \
    -out "${version_dir}/client.go" \
    -module "${MODULE}" \
    -version "${version}" \
    -services "${services_csv}"

  echo "Generated ${version_dir}/client.go"
done

echo ""
echo "Generation complete. Run 'go mod tidy' to update dependencies."
