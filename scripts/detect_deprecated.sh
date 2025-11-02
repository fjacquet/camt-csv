#!/bin/bash
# detect_deprecated.sh - Detects usage of deprecated APIs in CAMT-CSV codebase

set -e

echo "🔍 Scanning for deprecated API usage in CAMT-CSV..."
echo "=================================================="

FOUND_ISSUES=0

# Function to check for patterns and report findings
check_pattern() {
    local pattern="$1"
    local description="$2"
    local files
    
    files=$(grep -r "$pattern" . --include="*.go" --exclude-dir=vendor --exclude-dir=.git 2>/dev/null || true)
    
    if [ -n "$files" ]; then
        echo "⚠️  Found $description:"
        echo "$files" | sed 's/^/   /'
        echo ""
        FOUND_ISSUES=1
    fi
}

# Check for deprecated transaction methods
echo "Checking for deprecated transaction methods..."
check_pattern "\.GetAmountAsFloat()" "deprecated GetAmountAsFloat() calls"
check_pattern "\.GetDebitAsFloat()" "deprecated GetDebitAsFloat() calls"
check_pattern "\.GetCreditAsFloat()" "deprecated GetCreditAsFloat() calls"
check_pattern "\.GetFeesAsFloat()" "deprecated GetFeesAsFloat() calls"

# Check for deprecated transaction construction methods
echo "Checking for deprecated transaction construction..."
check_pattern "\.SetPayerInfo(" "deprecated SetPayerInfo() calls"
check_pattern "\.SetPayeeInfo(" "deprecated SetPayeeInfo() calls"
check_pattern "\.SetAmountFromFloat(" "deprecated SetAmountFromFloat() calls"
check_pattern "\.ToBuilder()" "deprecated ToBuilder() calls"

# Check for removed global functions (should not exist in v2.0+)
echo "Checking for removed global functions..."
check_pattern "GetDefaultCategorizer()" "removed GetDefaultCategorizer() calls"
check_pattern "GetGlobalConfig()" "removed GetGlobalConfig() calls"
check_pattern "categorizer\.CategorizeTransaction(" "removed global CategorizeTransaction() calls"

# Check for deprecated internal methods
echo "Checking for deprecated internal methods..."
check_pattern "\.categorizeWithGemini(" "deprecated categorizeWithGemini() calls"
check_pattern "ProcessFileLegacy(" "deprecated ProcessFileLegacy() calls"
check_pattern "SaveMappings(" "deprecated SaveMappings() calls"

# Check for old debitor naming (should be debtor)
echo "Checking for old naming conventions..."
check_pattern "debitor" "old 'debitor' naming (should be 'debtor')"

# Check for direct Transaction struct construction (should use builder)
echo "Checking for direct Transaction construction..."
check_pattern "models\.Transaction{" "direct Transaction struct construction (consider using TransactionBuilder)"

# Summary
echo "=================================================="
if [ $FOUND_ISSUES -eq 0 ]; then
    echo "✅ No deprecated API usage found!"
    echo ""
    echo "Your code is ready for future CAMT-CSV versions."
else
    echo "❌ Found deprecated API usage!"
    echo ""
    echo "📖 Migration guidance:"
    echo "   • See docs/DEPRECATION_TIMELINE.md for complete migration guide"
    echo "   • See docs/MIGRATION_GUIDE_V2.md for detailed examples"
    echo "   • Use TransactionBuilder pattern for new transaction construction"
    echo "   • Use container.NewContainer() for dependency injection"
    echo "   • Replace float64 methods with decimal.Decimal operations"
    echo ""
    echo "⏰ Timeline:"
    echo "   • Current deprecated methods will be removed in v3.0.0"
    echo "   • Global singleton functions were already removed in v2.0.0"
fi

exit $FOUND_ISSUES