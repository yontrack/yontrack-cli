// Package assistants recognises the assistants (agent kinds) which helped
// write commits, from their trailers, their author and their committer.
//
// It is a port of SCMCommitAssistants.kt of Yontrack
// (ontrack-extension-scm/.../scm/changelog/assistants/, yontrack/yontrack
// commit 6f0cb372c7d7c20d24a0bb8c094b5854e4f920b5), so that the assisted
// change the CI sets on a build is the one Yontrack would compute. Change
// both together.
package assistants

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// Names of the assistants behind the built-in conventions
const (
	ClaudeCode = "Claude Code"
	Codex      = "Codex"
	Copilot    = "Copilot"
	Devin      = "Devin"
)

const (
	trailerCoAuthoredBy  = "co-authored-by"
	trailerAssistedBy    = "assisted-by"
	trailerClaudeSession = "claude-session"
)

// Key of a trailer, as git accepts it
var trailerKeyRegex = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]*$`)

// A trailer line: its key, a colon, its value
var trailerLineRegex = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9-]*)\s*:\s*(.*)$`)

// Email in a `Name <email>` trailer value
var trailerEmailRegex = regexp.MustCompile(`<([^<>\s]+)>`)

// Commit is a commit, as git records it.
type Commit struct {
	Hash        string
	AuthorName  string
	AuthorEmail string
	// Name of the committer. As on Yontrack, it is matched against the logins.
	CommitterName string
	// Email of the committer. As on Yontrack, no rule reads it.
	CommitterEmail string
	// Full message of the commit
	Message string
}

// Marker is the kind of marker which names an assistant on a commit.
type Marker string

const (
	// MarkerCoAuthor is a Co-Authored-By trailer whose email is an agent's.
	MarkerCoAuthor Marker = "CO_AUTHOR"
	// MarkerAssistedBy is an Assisted-by trailer, naming the assistant.
	MarkerAssistedBy Marker = "ASSISTED_BY"
	// MarkerSessionTrailer is a session trailer (Claude-Session), carrying the session link.
	MarkerSessionTrailer Marker = "SESSION_TRAILER"
	// MarkerTrailer is a trailer whose key is named by an agent marker pattern.
	MarkerTrailer Marker = "TRAILER"
	// MarkerAuthor is the author of the commit: its email or its name.
	MarkerAuthor Marker = "AUTHOR"
	// MarkerCommitter is the committer of the commit: its name.
	MarkerCommitter Marker = "COMMITTER"
)

// Assistant is an assistant which helped write a commit.
type Assistant struct {
	Name string
	// Markers which named the assistant on the commit, in the order they were found
	Markers []Marker
	// Link to the agent session behind the commit, empty when no trailer carries one
	SessionLink string
}

// Trailer is a `Key: value` line of the trailer block of a commit message.
type Trailer struct {
	Key   string
	Value string
}

// Trailers of a commit message: the `Key: value` lines of its trailer block,
// which is the last paragraph of the message, provided there is more than
// one paragraph. A trailer anywhere else is prose.
//
// Windows line endings are tolerated. The trailers are in their order in the
// message, their values trimmed.
func Trailers(message string) []Trailer {
	var paragraphs [][]string
	var current []string
	for _, line := range splitLines(message) {
		if isBlank(line) {
			if len(current) > 0 {
				paragraphs = append(paragraphs, current)
				current = nil
			}
		} else {
			current = append(current, line)
		}
	}
	if len(current) > 0 {
		paragraphs = append(paragraphs, current)
	}
	if len(paragraphs) < 2 {
		return nil
	}
	var trailers []Trailer
	for _, line := range paragraphs[len(paragraphs)-1] {
		if match := trailerLineRegex.FindStringSubmatch(strings.TrimSpace(line)); match != nil {
			trailers = append(trailers, Trailer{Key: match[1], Value: strings.TrimSpace(match[2])})
		}
	}
	return trailers
}

// splitLines splits a message on any line ending: CRLF, LF or CR.
func splitLines(message string) []string {
	message = strings.ReplaceAll(message, "\r\n", "\n")
	message = strings.ReplaceAll(message, "\r", "\n")
	return strings.Split(message, "\n")
}

func isBlank(s string) bool {
	return strings.TrimSpace(s) == ""
}

// PatternType is what a Pattern is matched against.
type PatternType string

const (
	// PatternCoAuthorEmail is a regular expression matching the whole email of a Co-Authored-By trailer.
	PatternCoAuthorEmail PatternType = "CO_AUTHOR_EMAIL"
	// PatternTrailer is the key of a trailer, whatever its value.
	PatternTrailer PatternType = "TRAILER"
	// PatternAuthorEmail is a regular expression matching the whole email of the author.
	PatternAuthorEmail PatternType = "AUTHOR_EMAIL"
	// PatternLogin is the exact login - for the CLI, the name - of the author or of the committer.
	PatternLogin PatternType = "LOGIN"
)

// Pattern recognises an assistant on a commit.
type Pattern struct {
	// Name of the assistant to report
	Name string `json:"name"`
	// What the value is matched against
	Type PatternType `json:"type"`
	// Regular expression, trailer key or login, depending on the type
	Value string `json:"value"`
}

// Settings are the agent markers settings of Yontrack.
type Settings struct {
	// Whether the built-in conventions are recognised
	BuiltInConventions bool
	// Additional patterns
	Patterns []Pattern
}

// DefaultSettings are the settings of a Yontrack where nobody changed them:
// the built-in conventions only.
func DefaultSettings() Settings {
	return Settings{BuiltInConventions: true}
}

// ParseSettings reads the values of the agent-markers settings, as
// `settingsById(id: "agent-markers") { values }` returns them.
func ParseSettings(values []byte) (Settings, error) {
	var parsed struct {
		BuiltInConventions *bool     `json:"builtInConventions"`
		Patterns           []Pattern `json:"patterns"`
	}
	if err := json.Unmarshal(values, &parsed); err != nil {
		return Settings{}, fmt.Errorf("cannot read the agent markers settings: %w", err)
	}
	settings := DefaultSettings()
	if parsed.BuiltInConventions != nil {
		settings.BuiltInConventions = *parsed.BuiltInConventions
	}
	settings.Patterns = parsed.Patterns
	return settings, nil
}

type namedRegex struct {
	regex *regexp.Regexp
	name  string
}

// Rules are the effective rules recognising assistants on commits: the
// built-in conventions when they are on, plus the patterns of the settings.
type Rules struct {
	builtInConventions bool
	coAuthorEmails     []namedRegex
	trailers           map[string]string
	authorEmails       []namedRegex
	logins             map[string]string
}

// NewRules compiles the rules for some settings. An invalid pattern is
// skipped, and the warnings say which and why - like a regular expression
// which Java accepts, so Yontrack too, but which Go cannot compile.
func NewRules(settings Settings) (*Rules, []string) {
	rules := &Rules{
		builtInConventions: settings.BuiltInConventions,
		trailers:           map[string]string{},
		logins:             map[string]string{},
	}

	if settings.BuiltInConventions {
		rules.coAuthorEmails = append(rules.coAuthorEmails,
			exactEmail("noreply@anthropic.com", ClaudeCode),
			exactEmail("codex@openai.com", Codex),
			exactEmail("copilot@github.com", Copilot),
		)
		rules.authorEmails = append(rules.authorEmails, exactEmail("copilot@github.com", Copilot))
		rules.logins["copilot-swe-agent[bot]"] = Copilot
		rules.logins["devin-ai-integration[bot]"] = Devin
	}

	var warnings []string
	for index, pattern := range settings.Patterns {
		name := strings.TrimSpace(pattern.Name)
		value := strings.TrimSpace(pattern.Value)
		if problem := checkPattern(pattern); problem != "" {
			if name == "" {
				warnings = append(warnings, fmt.Sprintf("Pattern #%d: %s", index+1, problem))
			} else {
				warnings = append(warnings, fmt.Sprintf("Pattern #%d (%s): %s", index+1, name, problem))
			}
			continue
		}
		switch pattern.Type {
		case PatternCoAuthorEmail:
			rules.coAuthorEmails = append(rules.coAuthorEmails, namedRegex{wholeIgnoringCase(value), name})
		case PatternTrailer:
			putIfAbsent(rules.trailers, strings.ToLower(value), name)
		case PatternAuthorEmail:
			rules.authorEmails = append(rules.authorEmails, namedRegex{wholeIgnoringCase(value), name})
		case PatternLogin:
			putIfAbsent(rules.logins, strings.ToLower(value), name)
		}
	}

	return rules, warnings
}

// checkPattern is the problem of a pattern, empty when it is valid.
func checkPattern(pattern Pattern) string {
	value := strings.TrimSpace(pattern.Value)
	switch {
	case isBlank(pattern.Name):
		return "the name is required"
	case value == "":
		return "the value is required"
	}
	switch pattern.Type {
	case PatternCoAuthorEmail, PatternAuthorEmail:
		if _, err := regexp.Compile(value); err != nil {
			return fmt.Sprintf("invalid regular expression `%s`", value)
		}
	case PatternTrailer:
		if !trailerKeyRegex.MatchString(value) {
			return fmt.Sprintf("`%s` is not a trailer key", value)
		}
	case PatternLogin:
	default:
		return fmt.Sprintf("unknown type `%s`", pattern.Type)
	}
	return ""
}

func putIfAbsent(m map[string]string, key, value string) {
	if _, ok := m[key]; !ok {
		m[key] = value
	}
}

// wholeIgnoringCase compiles a valid regular expression to match a whole
// string, ignoring the case, as Kotlin's Regex.matches does.
func wholeIgnoringCase(expression string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)^(?:` + expression + `)$`)
}

func exactEmail(email, name string) namedRegex {
	return namedRegex{wholeIgnoringCase(regexp.QuoteMeta(email)), name}
}

// Assistants of a commit, once each, in the order they were found.
func (r *Rules) Assistants(commit Commit) []Assistant {
	var collector collector

	// Trailers
	for _, trailer := range Trailers(commit.Message) {
		key := strings.ToLower(trailer.Key)
		switch key {
		case trailerCoAuthoredBy:
			if email := trailerEmail(trailer.Value); email != "" {
				if name, ok := firstMatch(r.coAuthorEmails, email); ok {
					collector.add(name, MarkerCoAuthor, "")
				}
			}
		case trailerAssistedBy:
			if r.builtInConventions {
				// Kernel style is `Assisted-by: AGENT_NAME:MODEL_VERSION [TOOL...]`
				name, _, _ := strings.Cut(trailer.Value, ":")
				if name = strings.TrimSpace(name); name != "" {
					collector.add(name, MarkerAssistedBy, "")
				}
			}
		case trailerClaudeSession:
			if r.builtInConventions {
				collector.add(ClaudeCode, MarkerSessionTrailer, strings.TrimSpace(trailer.Value))
			}
		}
		if name, ok := r.trailers[key]; ok {
			collector.add(name, MarkerTrailer, "")
		}
	}

	// Author
	if email := strings.TrimSpace(commit.AuthorEmail); email != "" {
		if name, ok := firstMatch(r.authorEmails, email); ok {
			collector.add(name, MarkerAuthor, "")
		}
	}
	if name, ok := r.login(commit.AuthorName); ok {
		collector.add(name, MarkerAuthor, "")
	}

	// Committer
	if name, ok := r.login(commit.CommitterName); ok {
		collector.add(name, MarkerCommitter, "")
	}

	return collector.assistants
}

// login is the assistant named by a name as git records it: bots commit
// under their login. Yontrack prefers the SCM login, which the CLI does not
// have, and falls back on this.
func (r *Rules) login(name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}
	assistant, ok := r.logins[strings.ToLower(name)]
	return assistant, ok
}

func firstMatch(rules []namedRegex, value string) (string, bool) {
	for _, rule := range rules {
		if rule.regex.MatchString(value) {
			return rule.name, true
		}
	}
	return "", false
}

// trailerEmail is the email of a `Name <email>` trailer value, or the value
// itself when it is a bare email, or empty.
func trailerEmail(value string) string {
	if match := trailerEmailRegex.FindStringSubmatch(value); match != nil {
		return match[1]
	}
	value = strings.TrimSpace(value)
	if strings.Contains(value, "@") && !strings.ContainsFunc(value, unicode.IsSpace) {
		return value
	}
	return ""
}

// collector gathers the assistants of a commit, once each whatever the case
// of their name, with all their markers and the first session link found.
type collector struct {
	assistants []Assistant
}

func (c *collector) add(name string, marker Marker, sessionLink string) {
	for index := range c.assistants {
		entry := &c.assistants[index]
		if strings.EqualFold(entry.Name, name) {
			if !containsMarker(entry.Markers, marker) {
				entry.Markers = append(entry.Markers, marker)
			}
			if entry.SessionLink == "" {
				entry.SessionLink = sessionLink
			}
			return
		}
	}
	c.assistants = append(c.assistants, Assistant{Name: name, Markers: []Marker{marker}, SessionLink: sessionLink})
}

func containsMarker(markers []Marker, marker Marker) bool {
	for _, m := range markers {
		if m == marker {
			return true
		}
	}
	return false
}

// Change is the assisted change of a range of commits.
type Change struct {
	// Distinct names of the assistants, sorted
	Assistants []string
	// Number of commits written with an assistant
	AssistedCommits int
	// Number of commits
	TotalCommits int
	// Distinct links to the agent sessions behind the commits
	SessionLinks []string
}

// Summarize is the assisted change of commits, as Yontrack computes it from
// a change log: every commit counts, merges included.
func Summarize(commits []Commit, rules *Rules) Change {
	change := Change{
		Assistants:   []string{},
		TotalCommits: len(commits),
		SessionLinks: []string{},
	}
	names := map[string]bool{}
	links := map[string]bool{}
	for _, commit := range commits {
		found := rules.Assistants(commit)
		if len(found) > 0 {
			change.AssistedCommits++
		}
		for _, assistant := range found {
			if name := strings.TrimSpace(assistant.Name); !names[name] {
				names[name] = true
				change.Assistants = append(change.Assistants, name)
			}
			if link := assistant.SessionLink; link != "" && !links[link] {
				links[link] = true
				change.SessionLinks = append(change.SessionLinks, link)
			}
		}
	}
	sort.Strings(change.Assistants)
	return change
}
