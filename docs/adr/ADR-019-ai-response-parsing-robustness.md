# ADR-019: Robust AI Response Parsing in cleanCategory

## Status

**ACCEPTED** - Implemented as hotfix post-v1.6

## Context

After deploying OpenRouter with `mistralai/mistral-small-2603`, the `cleanCategory` function in both `OpenRouterClient` and `GeminiClient` received verbose, multi-line responses instead of a bare category name. For example:

```
Given the description "L ESTANCOT" (a known French restaurant chain), the category for this transaction is:

**Restaurants**
```

The existing `cleanCategory` only stripped `Category:` prefixes and trimmed whitespace. It could not handle:
1. Multi-line explanatory text before the answer
2. Markdown bold formatting (`**Category**`)

As a result, the full AI explanation was stored verbatim in staging YAML files and used as the category — which never matched any known category, leaving transactions as `Uncategorized`.

## Decision

Extend `cleanCategory` in both `OpenRouterClient` and `GeminiClient` with two pre-processing steps applied before existing logic:

1. **Multi-line extraction**: If the response contains newlines, walk lines from the end and take the last non-empty line as the candidate. This handles models that explain reasoning before giving the answer.

2. **Markdown bold stripping**: Strip leading/trailing `*` characters to handle `**Category**` formatting.

Both clients share identical logic; no shared utility function is introduced (Rule of Three — only two call sites).

## Rationale

### Why last line, not first?

Models tend to state their conclusion last ("... therefore the category is: **Restaurants**"). Taking the last non-empty line is correct for this pattern. First-line extraction would capture the explanation prefix.

### Why not fix the prompt instead?

The prompt already ends with `Category:` to elicit a bare answer. Some models ignore this instruction and produce verbose output regardless. Defensive parsing in `cleanCategory` is more robust than relying on model compliance.

### Why not a shared helper?

Two call sites. A shared function would add indirection without eliminating meaningful duplication. If a third client is added, extract then.

## Consequences

### Positive

- Verbose AI responses are correctly parsed regardless of model chattiness
- Staging YAML files receive clean category names, not AI explanations
- Backward compatible — single-line bare responses still work

### Negative

- If a model returns a multi-line response where the last line is not the category (e.g., a disclaimer), parsing will fail silently and return `Uncategorized`
- Two copies of the same logic to maintain across `OpenRouterClient` and `GeminiClient`
