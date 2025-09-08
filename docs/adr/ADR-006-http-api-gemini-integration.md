# ADR-006: Simplified Jetify AI SDK Integration

## Status

**PROPOSED** - Awaiting implementation

## Context

The current implementation uses `github.com/google/generative-ai-go` v0.7.0 (pinned due to protobuf compatibility issues with newer versions). This creates technical debt, security risks, and blocks ecosystem updates. The AI categorization feature is non-critical (3rd tier fallback) but provides value for edge cases.

## Decision

Replace the `github.com/google/generative-ai-go` SDK dependency with **direct usage** of Jetify AI SDK's Google provider (`go.jetify.com/ai/provider/google`). Apply KISS and DRY principles by avoiding unnecessary abstractions and multi-provider complexity.

## Rationale

### Problems with Current Approach

- **Technical Debt**: Pinned to outdated version (v0.7.0 from June 2024)
- **Security Risk**: Missing 6+ months of security updates
- **Dependency Conflicts**: Protobuf version incompatibilities
- **Maintenance Burden**: Blocks Go ecosystem updates
- **Vendor Lock-in**: Tied to Google's SDK implementation

### Benefits of Simplified Jetify AI SDK Approach

- **Zero Dependency Conflicts**: No protobuf or version issues
- **Minimal Complexity**: Direct SDK usage, no custom abstractions
- **Go-Idiomatic**: Built specifically for Go (not auto-generated)
- **Production-Ready**: Built-in retries, rate limiting, error handling
- **KISS Principle**: Simple solution for simple problem
- **DRY Principle**: Single configuration source, no duplication
- **Active Development**: Apache 2.0 license, backed by Jetify
- **Focused Scope**: Google Gemini only (YAGNI - You Aren't Gonna Need It)

## Technical Specification

### 1. Direct Jetify AI SDK Usage (KISS Principle)

```go
import (
    "context"
    "os"
    "strings"
    "go.jetify.com/ai"
    "go.jetify.com/ai/provider/google"
)

// No custom wrapper - use Jetify SDK directly
func (c *Categorizer) categorizeWithAI(ctx context.Context, prompt string) (string, error) {
    apiKey := os.Getenv("GEMINI_API_KEY")
    if apiKey == "" {
        return "", fmt.Errorf("GEMINI_API_KEY not set")
    }
    
    model := google.NewLanguageModel("gemini-2.0-flash").WithAPIKey(apiKey)
    
    response, err := ai.GenerateTextStr(
        ctx,
        prompt,
        ai.WithModel(model),
        ai.WithMaxOutputTokens(50),
        ai.WithTemperature(0.1),
    )
    
    if err != nil {
        return "", fmt.Errorf("AI categorization failed: %w", err)
    }
    
    return strings.TrimSpace(response), nil
}
```

### 2. Simplified Configuration (DRY Principle)

#### Viper Configuration Updates

```yaml
ai:
  enabled: false  # Default disabled
  # API key from GEMINI_API_KEY environment variable only
```

#### Configuration Struct Updates

```go
type AIConfig struct {
    Enabled bool `mapstructure:"enabled"`
    // API key loaded from GEMINI_API_KEY environment variable
    // No need for additional fields - use SDK defaults
}
```

### 3. Integration Points

#### Categorizer Updates

```go
// Remove custom abstractions - use existing patterns
type Categorizer struct {
    // ... existing fields ...
    // Remove: aiClient, rateLimiter (Jetify SDK handles rate limiting)
}

// Simple initialization following existing patterns
func (c *Categorizer) initializeAI() error {
    cfg := config.GetGlobalConfig()
    if !cfg.AI.Enabled {
        c.logger.Debug("AI categorization disabled")
        return nil
    }
    
    apiKey := os.Getenv("GEMINI_API_KEY")
    if apiKey == "" {
        c.logger.Warn("AI categorization disabled: GEMINI_API_KEY not set")
        return nil
    }
    
    c.logger.Info("AI categorization enabled with Gemini")
    return nil
}
```

## Implementation Plan

### Phase 1: Simple Integration (2-3 hours)

1. Add `go.jetify.com/ai` dependency
2. Update `internal/categorizer/categorizer.go`:
   - Replace `genai.Client` imports with Jetify SDK
   - Update `categorizeWithGemini()` method to use direct SDK calls
   - Remove custom rate limiting (SDK handles this)
3. Update configuration to simple boolean enable/disable
4. Test with existing `GEMINI_API_KEY`

### Phase 2: Cleanup (30 minutes)

1. Remove `github.com/google/generative-ai-go` dependency
2. Update `go.mod` and `go.sum`
3. Verify all tests pass

**Total Effort**: **2-3 hours** (vs original 2-3 days)

## Testing Strategy

### Unit Tests

- Direct Jetify SDK integration
- Error handling for missing API key
- Configuration validation
- Backward compatibility

### Integration Tests

- Real API calls with Gemini
- Categorization accuracy comparison
- Performance validation

### Performance Tests

- Response time comparison (old SDK vs Jetify)
- Memory usage analysis
- Concurrent request handling

## Backward Compatibility

### API Compatibility

- Same categorization results
- Identical error handling behavior
- Simplified configuration (backward compatible)
- Same performance characteristics

### Migration Path

- Zero configuration changes required
- Existing `GEMINI_API_KEY` continues to work
- Same CLI flags and environment variables
- Graceful fallback for missing configuration
- Drop-in replacement

## Risks and Mitigations

### Risk: SDK Dependency

- **Mitigation**: Jetify AI SDK is actively maintained, Apache 2.0 licensed
- **Monitoring**: SDK version updates, community activity

### Risk: Performance Regression

- **Mitigation**: Performance testing, built-in SDK optimizations
- **Monitoring**: Response time metrics, success rate tracking

### Risk: Over-Simplification

- **Mitigation**: KISS principle appropriate for 3rd-tier fallback feature
- **Monitoring**: Feature usage metrics, categorization accuracy

## Success Criteria

1. **Functionality**: 100% feature parity with current implementation
2. **Performance**: Response times within 10% of current SDK
3. **Reliability**: 99.9% success rate for valid requests
4. **Maintainability**: Minimal code, no custom abstractions
5. **Security**: No API key exposure, secure handling
6. **Simplicity**: KISS and DRY principles applied

## Dependencies Changed

```go
// go.mod changes
- github.com/google/generative-ai-go v0.7.0
+ go.jetify.com/ai v0.3.2

// Indirect dependencies removed:
- cloud.google.com/go/ai v0.13.0
- Multiple protobuf-related dependencies

// New minimal dependencies:
+ Jetify AI SDK (Google provider only)
```

## Configuration Examples

### Minimal Configuration (KISS Principle)

```yaml
# ~/.camt-csv/config.yaml
ai:
  enabled: false  # Default disabled
  # API key from GEMINI_API_KEY environment variable
```

### Environment Variables

```bash
# Single API key (backward compatible)
export GEMINI_API_KEY="your-gemini-key-here"

# Optional: Enable via environment
export CAMT_AI_ENABLED=true
```

## Monitoring and Observability

### Logging

- Structured logging for AI categorization calls
- Request/response timing
- Error categorization and frequency
- API key validation events

### Metrics

- API response times (p50, p95, p99)
- Success/failure rates
- Categorization accuracy
- Feature usage frequency

## Future Enhancements

1. **Caching**: Response caching for repeated transactions
2. **Batch Processing**: Multiple transactions per API call
3. **Local LLM Integration**: Offline categorization if needed
4. **Additional Providers**: Only if business case emerges (YAGNI)

**Note**: Keep enhancements minimal - this is a 3rd-tier fallback feature

---

**Author**: Development Team  
**Date**: 2025-01-07  
**Reviewers**: Product Manager, Tech Lead  
**Implementation Target**: Sprint 2025.1
