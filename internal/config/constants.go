package config

import "time"

// Public aliases and cross-project compatibility constants.
// Keep values minimal and self-contained so they can be referenced from
// other files without introducing cyclic dependencies.

// AppName is the exported alias of appName used by compatibility helpers.
const AppName = appName

// CurrentConfigFilename is the filename for the current app's JSON config.
const CurrentConfigFilename = AppName + ".json"

// LegacyAppName and LegacyConfigFilename allow reading legacy config/cache
// directories from a previous app name.
//
// In this repository we prioritize the current app (crush) and fall back to
// the legacy app (lash) when present, enabling a smoother migration.
const (
	LegacyAppName        = "lash"
	LegacyConfigFilename = LegacyAppName + ".json"
)

// ProvidersCacheFilename is the filename used to cache provider metadata.
const ProvidersCacheFilename = "providers.json"

// ProvidersCacheTTL defines how long the providers cache is considered fresh.
const ProvidersCacheTTL = 24 * time.Hour

// Anthropic defaults and header keys
const (
	DefaultAnthropicBaseURL = "https://api.anthropic.com/v1"
	HeaderAnthropicVersion  = "anthropic-version"
	DefaultAnthropicAPIVer  = "2023-06-01"
)

// Tool defaults
const (
	// Default timeout for bash tool when not specified (30 minutes)
	DefaultBashTimeoutMs = 30 * 60 * 1000
	// Maximum allowed timeout value that can be requested by the user (10 minutes)
	MaxBashTimeoutMs = 10 * 60 * 1000
	// Output will be truncated when exceeding this number of characters
	MaxBashOutputChars = 50000
	// Default number of lines to show/tail for logs and listings
	DefaultTailLines = 1000
)

// HTTP and token formatting constants used by providers
const (
	// Prefix used when an OAuth access token is provided instead of a raw API key
	BearerPrefix = "Bearer "
	// Standard HTTP header for server-provided retry backoff
	HeaderRetryAfter = "Retry-After"
)

// Token safety knobs used to keep completions within provider limits
const (
	// Reserve a small buffer to account for tool/system messages and tokenization variance
	ContextLimitBufferTokens = 500
	// Ensure we always allow at least this many tokens for the model to respond
	MinSafeMaxTokens = 256
)

// UI constants for spinner/verification UX
const (
    VerificationMinSpinnerDuration = 500 * time.Millisecond
)

// Data directory naming (for user home fallbacks)
const (
    DefaultDataDirectoryName = ".crush"
)

// Onboarding UX timings
const (
    OnboardingAPIKeySubmitDelay = 300 * time.Millisecond
)
