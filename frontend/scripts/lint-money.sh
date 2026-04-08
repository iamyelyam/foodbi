#!/bin/bash
# Money lint: catches common monetary formatting bugs in frontend code.
# Run via: npm run lint:money

set -e
ERRORS=0
SRC="frontend/src"

# If run from frontend/ dir, adjust path
if [ -d "src" ] && [ ! -d "frontend" ]; then
  SRC="src"
fi

echo "=== Money Lint ==="

# 1. No .toFixed(2) on monetary values (KZT has no subunits)
MATCHES=$(grep -rn '\.toFixed(2)' "$SRC" --include='*.tsx' --include='*.ts' || true)
if [ -n "$MATCHES" ]; then
  echo "FAIL: .toFixed(2) found (KZT has no decimals, use toLocaleString):"
  echo "$MATCHES"
  ERRORS=$((ERRORS + 1))
fi

# 2. No toLocaleString('en' in pages/components (wrong locale for KZT)
MATCHES=$(grep -rn "toLocaleString('en'" "$SRC/pages" "$SRC/components" --include='*.tsx' --include='*.ts' 2>/dev/null || true)
if [ -n "$MATCHES" ]; then
  echo "FAIL: toLocaleString('en') found (use 'ru-KZ' for KZT):"
  echo "$MATCHES"
  ERRORS=$((ERRORS + 1))
fi

# 3. No hardcoded non-KZT currency symbols
MATCHES=""
for SYM in '€' '₽' '£' '¥'; do
  FOUND=$(grep -rn --fixed-strings "$SYM" "$SRC" --include='*.tsx' --include='*.ts' || true)
  if [ -n "$FOUND" ]; then
    MATCHES="${MATCHES}${FOUND}\n"
  fi
done
if [ -n "$MATCHES" ]; then
  echo "FAIL: Hardcoded currency symbols found (use useCurrency() hook):"
  echo -e "$MATCHES"
  ERRORS=$((ERRORS + 1))
fi

# 4. No division by 100 on monetary values
MATCHES=$(grep -rn '/ 100\b\|/100\b\|\* 0\.01\b\|\*0\.01\b' "$SRC" --include='*.tsx' --include='*.ts' || true)
if [ -n "$MATCHES" ]; then
  echo "FAIL: Division/multiplication on monetary values found:"
  echo "$MATCHES"
  ERRORS=$((ERRORS + 1))
fi

if [ "$ERRORS" -gt 0 ]; then
  echo ""
  echo "=== $ERRORS money lint violation(s) found ==="
  exit 1
fi

echo "All money lint checks passed."
