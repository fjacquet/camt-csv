# ADR-002: Hybrid Categorization Approach

## Status

Accepted

## Context

Transaction categorization is critical for financial analysis, but presents several challenges:

1. **Accuracy**: Manual categorization is time-consuming but accurate
2. **Scale**: Large transaction volumes require automation
3. **Learning**: System should improve over time
4. **Cost**: AI services have usage costs
5. **Reliability**: External services may be unavailable
6. **Privacy**: Some users prefer local-only processing

## Decision

We will implement a three-tier hybrid categorization system:

1. **Direct Mapping** (Tier 1): Exact matches against known creditor/debtor mappings
2. **Keyword Matching** (Tier 2): Local keyword-based categorization using predefined rules
3. **AI Fallback** (Tier 3): Google Gemini AI for unknown transactions (optional)

### Categorization Flow

```
Transaction → Direct Mapping → Found? → Return Category
                     ↓ No
              Keyword Matching → Found? → Return Category
                     ↓ No
              AI Enabled? → No → Return "Uncategorized"
                     ↓ Yes
              AI Categorization → Success? → Save Mapping & Return Category
                     ↓ No
              Return "Uncategorized"
```

## Consequences

### Positive

- **Performance**: Fast local lookups for known transactions
- **Learning**: AI results automatically improve local mappings
- **Cost Control**: AI only used when necessary
- **Offline Capability**: Works without internet for known patterns
- **Privacy**: Sensitive transactions can be handled locally
- **Accuracy**: Combines human knowledge with AI intelligence

### Negative

- **Complexity**: Three different categorization mechanisms to maintain
- **Configuration**: Requires management of keyword rules and mappings
- **Dependencies**: Optional dependency on external AI service
- **Rate Limiting**: AI service has usage limits

### Mitigation Strategies

- Comprehensive configuration management for all three tiers
- Graceful degradation when AI service is unavailable
- Rate limiting implementation to prevent API quota exhaustion
- Clear documentation for configuring keyword rules

## Implementation Details

### Direct Mapping Storage

```yaml
# database/creditor_categories.yaml
"MIGROS": "Groceries"
"COOP": "Groceries"
"SBB": "Transportation"

# database/debitor_categories.yaml
"Salary Company": "Income"
"Insurance Provider": "Insurance"
```

### Keyword Matching Rules

```yaml
# database/categories.yaml
categories:
  Groceries:
    keywords: ["supermarket", "grocery", "food", "migros", "coop"]
  Transportation:
    keywords: ["sbb", "train", "bus", "transport", "mobility"]
```

### AI Integration

- Uses Google Gemini API with configurable model
- Rate limited to prevent quota exhaustion
- Structured prompts for consistent categorization
- Automatic learning from successful categorizations

## Configuration

```yaml
categorization:
  ai_enabled: false
  ai_model: "gemini-2.0-flash"
  rate_limit: 10  # requests per minute
  fallback_category: "Uncategorized"
```

## Related Decisions

- ADR-001: Parser interface standardization
- ADR-003: Functional programming adoption
- ADR-004: Configuration management strategy

## Date

2024-12-19

## Authors

- Development Team
