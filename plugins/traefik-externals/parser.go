package traefikexternals

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/coredns/coredns/plugin/pkg/log"
)

// maxFileSize limits file reads to prevent DoS attacks (1MB)
const maxFileSize = 1 << 20

// Parser extracts Host() rules from Traefik YAML config files.
type Parser struct {
	// hostPattern matches Host(...) patterns, capturing the contents
	hostPattern *regexp.Regexp

	// hostSNIPattern matches HostSNI(...) patterns for TCP/TLS services
	hostSNIPattern *regexp.Regexp

	// hostRegexpPattern detects HostRegexp() usage (not supported, but we warn)
	hostRegexpPattern *regexp.Regexp

	// backtickPattern extracts content from backtick-quoted strings
	backtickPattern *regexp.Regexp

	// envPattern matches {{env "VAR"}} Go template patterns
	envPattern *regexp.Regexp

	// middlewareOnlyFiles are files that only define middlewares, not routers
	middlewareOnlyFiles map[string]bool

	// envVars caches environment variables
	envVars map[string]string

	// hostRegexpWarned tracks if we've already warned about HostRegexp
	hostRegexpWarned bool
}

// NewParser creates a new Traefik config parser.
// Environment variables are captured at creation time and are immutable.
func NewParser() *Parser {
	return NewParserWithEnv(loadEnvVars())
}

// NewParserWithEnv creates a parser with custom environment variables.
// This is primarily useful for testing. The provided map is copied to prevent
// external mutation.
func NewParserWithEnv(envVars map[string]string) *Parser {
	// Copy the map to ensure immutability
	vars := make(map[string]string, len(envVars))
	for k, v := range envVars {
		vars[k] = v
	}

	return &Parser{
		// Matches: Host(...) - captures everything inside parentheses
		// Supports both single and multi-host: Host(`a.com`) or Host(`a.com`, `b.com`)
		hostPattern: regexp.MustCompile(`Host\(([^)]+)\)`),

		// Matches: HostSNI(...) for TCP/TLS services
		hostSNIPattern: regexp.MustCompile(`HostSNI\(([^)]+)\)`),

		// Detects HostRegexp() usage - we can't enumerate regex matches
		hostRegexpPattern: regexp.MustCompile(`HostRegexp\(`),

		// Extracts content from backtick-quoted strings
		backtickPattern: regexp.MustCompile("`([^`]+)`"),

		// Matches: {{env "VAR_NAME"}}
		envPattern: regexp.MustCompile(`\{\{env\s+"([^"]+)"\}\}`),

		middlewareOnlyFiles: map[string]bool{
			"middleware.yml":           true,
			"authentik_middleware.yml": true,
			"authelia_middlewares.yml": true,
			"crowdsec-bouncer.yml":     true,
		},

		envVars: vars,
	}
}

// loadEnvVars loads all environment variables into a map.
func loadEnvVars() map[string]string {
	vars := make(map[string]string)
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			vars[parts[0]] = parts[1]
		}
	}
	return vars
}

// IsMiddlewareOnly returns true if the file is a middleware-only config.
func (p *Parser) IsMiddlewareOnly(filename string) bool {
	return p.middlewareOnlyFiles[filename]
}

// ParseFile reads a Traefik YAML file and extracts resolved hostnames.
// Files larger than 1MB are rejected to prevent DoS attacks.
func (p *Parser) ParseFile(filePath string) ([]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Limit read to maxFileSize to prevent DoS
	content, err := io.ReadAll(io.LimitReader(f, maxFileSize+1))
	if err != nil {
		return nil, err
	}

	if len(content) > maxFileSize {
		return nil, fmt.Errorf("file exceeds maximum size of %d bytes", maxFileSize)
	}

	return p.ParseContent(string(content)), nil
}

// ParseContent extracts resolved hostnames from Traefik config content.
// Supports Host(), HostSNI(), and multi-host syntax like Host(`a.com`, `b.com`).
// Warns about HostRegexp() usage (not supported as we can't enumerate regex matches).
// YAML comments (lines starting with #) are stripped before parsing.
func (p *Parser) ParseContent(content string) []string {
	var hosts []string
	seen := make(map[string]bool)

	// Strip YAML comments before parsing to avoid matching commented-out rules
	content = p.stripComments(content)

	// Warn about HostRegexp() usage (once per parser instance)
	if !p.hostRegexpWarned && p.hostRegexpPattern.MatchString(content) {
		log.Warning("traefik-externals: HostRegexp() rules found but not supported (cannot enumerate regex matches)")
		p.hostRegexpWarned = true
	}

	// Extract hosts from Host() patterns
	p.extractHostsFromMatches(p.hostPattern.FindAllStringSubmatch(content, -1), seen, &hosts)

	// Extract hosts from HostSNI() patterns (TCP/TLS services)
	p.extractHostsFromMatches(p.hostSNIPattern.FindAllStringSubmatch(content, -1), seen, &hosts)

	return hosts
}

// stripComments removes YAML comments from content.
// Handles both full-line comments and inline comments.
func (p *Parser) stripComments(content string) string {
	lines := strings.Split(content, "\n")
	var cleanLines []string

	for _, line := range lines {
		// Find comment marker (outside of quoted strings)
		// Simple approach: find first # that's not inside backticks
		inBacktick := false
		commentIdx := -1

		for i, ch := range line {
			if ch == '`' {
				inBacktick = !inBacktick
			} else if ch == '#' && !inBacktick {
				commentIdx = i
				break
			}
		}

		if commentIdx >= 0 {
			line = line[:commentIdx]
		}
		cleanLines = append(cleanLines, line)
	}

	return strings.Join(cleanLines, "\n")
}

// extractHostsFromMatches processes regex matches and extracts hostnames.
func (p *Parser) extractHostsFromMatches(matches [][]string, seen map[string]bool, hosts *[]string) {
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		// match[1] contains everything inside Host() or HostSNI()
		// e.g., "`a.com`" or "`a.com`, `b.com`"
		innerContent := match[1]

		// Extract all backtick-quoted strings
		backtickMatches := p.backtickPattern.FindAllStringSubmatch(innerContent, -1)
		for _, btMatch := range backtickMatches {
			if len(btMatch) < 2 {
				continue
			}

			hostTemplate := btMatch[1]

			// Resolve environment variable templates
			resolved := p.resolveEnvTemplate(hostTemplate)
			if resolved == "" {
				// Template resolution failed (missing env var)
				continue
			}

			// Normalize to lowercase
			resolved = strings.ToLower(resolved)

			if !seen[resolved] {
				seen[resolved] = true
				*hosts = append(*hosts, resolved)
			}
		}
	}
}

// resolveEnvTemplate resolves {{env "VAR"}} patterns in a string.
// Returns empty string if any required variable is missing or empty.
func (p *Parser) resolveEnvTemplate(template string) string {
	result := template

	// Find all env references
	matches := p.envPattern.FindAllStringSubmatch(template, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		varName := match[0]  // Full match: {{env "VAR"}}
		envKey := match[1]   // Captured group: VAR

		envValue, exists := p.envVars[envKey]
		if !exists || envValue == "" {
			log.Debugf("traefik-externals: env var %s not set, skipping host", envKey)
			return ""
		}

		result = strings.Replace(result, varName, envValue, 1)
	}

	return result
}

// GetEnvVar returns an environment variable from the parser's cache.
// The returned value is read-only as the envVars map is immutable after creation.
func (p *Parser) GetEnvVar(key string) (string, bool) {
	val, ok := p.envVars[key]
	return val, ok
}
