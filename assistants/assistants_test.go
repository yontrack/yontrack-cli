package assistants

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The cases of SCMCommitAssistantsTest.kt, in
// ontrack-extension-scm/src/test/java/net/nemerosa/ontrack/extension/scm/changelog/assistants/
// of yontrack/yontrack, copied from commit 6f0cb372c7d7c20d24a0bb8c094b5854e4f920b5,
// so that the CLI and Yontrack give a build the same assisted change.
//
// The CLI has no SCM login: where a case of the server gives a login, it is
// given here as the git author or committer name, which is what the server
// falls back on, and what the CLI reads.

var builtIns, _ = NewRules(DefaultSettings())

// commit is the commit of the server's tests: the author is Damien,
// damien@example.com, and there is no committer.
func commit(message string, options ...func(*Commit)) Commit {
	c := Commit{
		Hash:        "0123456789abcdef",
		AuthorName:  "Damien",
		AuthorEmail: "damien@example.com",
		Message:     message,
	}
	for _, option := range options {
		option(&c)
	}
	return c
}

func author(name string) func(*Commit) {
	return func(c *Commit) { c.AuthorName = name }
}

func authorEmail(email string) func(*Commit) {
	return func(c *Commit) { c.AuthorEmail = email }
}

func committer(name string) func(*Commit) {
	return func(c *Commit) { c.CommitterName = name }
}

func committerEmail(email string) func(*Commit) {
	return func(c *Commit) { c.CommitterEmail = email }
}

func assistant(name string, markers ...Marker) Assistant {
	return Assistant{Name: name, Markers: markers}
}

func withSession(a Assistant, link string) Assistant {
	a.SessionLink = link
	return a
}

func rules(t *testing.T, settings Settings) *Rules {
	t.Helper()
	r, warnings := NewRules(settings)
	assert.Empty(t, warnings)
	return r
}

func TestTrailers(t *testing.T) {
	cases := []struct {
		name     string
		message  string
		expected []Trailer
	}{
		{
			"Trailers of the last paragraph",
			"Subject\n\nSome body.\n\nRefs: #12\nCo-Authored-By: Claude <noreply@anthropic.com>",
			[]Trailer{{"Refs", "#12"}, {"Co-Authored-By", "Claude <noreply@anthropic.com>"}},
		},
		{
			"No trailers in a subject alone",
			"Co-Authored-By: Claude <noreply@anthropic.com>",
			nil,
		},
		{
			"Trailers with Windows line endings",
			"Subject\r\n\r\nBody\r\n\r\nCo-Authored-By: Claude <noreply@anthropic.com>\r\n",
			[]Trailer{{"Co-Authored-By", "Claude <noreply@anthropic.com>"}},
		},
		{
			"Trailing blank lines are ignored",
			"Subject\n\nAssisted-by: Codex\n\n\n",
			[]Trailer{{"Assisted-by", "Codex"}},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, Trailers(c.message))
		})
	}
}

func TestAssistants(t *testing.T) {
	coAuthorPattern := Settings{BuiltInConventions: true, Patterns: []Pattern{
		{Name: "Acme Bot", Type: PatternCoAuthorEmail, Value: `.*-bot@acme\.com`},
	}}
	trailerPattern := Settings{BuiltInConventions: true, Patterns: []Pattern{
		{Name: "Acme Agent", Type: PatternTrailer, Value: "Acme-Agent-Run"},
	}}
	authorEmailPattern := Settings{BuiltInConventions: true, Patterns: []Pattern{
		{Name: "Acme Bot", Type: PatternAuthorEmail, Value: `bot\+.*@acme\.com`},
	}}
	loginPattern := Settings{BuiltInConventions: true, Patterns: []Pattern{
		{Name: "Acme Bot", Type: PatternLogin, Value: "acme-bot"},
	}}
	builtInsOff := Settings{BuiltInConventions: false, Patterns: []Pattern{
		{Name: "Acme Bot", Type: PatternLogin, Value: "acme-bot"},
	}}

	cases := []struct {
		name     string
		settings *Settings
		commit   Commit
		expected []Assistant
	}{
		// Plain commits
		{"No assistant on a plain commit", nil,
			commit("Some fix\n\nWith a body."),
			nil},

		// Co-Authored-By
		{"Claude Code from its co-author trailer", nil,
			commit("Some fix\n\nCo-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"),
			[]Assistant{assistant("Claude Code", MarkerCoAuthor)}},
		{"Codex from its co-author trailer", nil,
			commit("Some fix\n\nCo-authored-by: Codex <codex@openai.com>"),
			[]Assistant{assistant("Codex", MarkerCoAuthor)}},
		{"Copilot from its co-author trailer", nil,
			commit("Some fix\n\nCo-authored-by: Copilot <copilot@github.com>"),
			[]Assistant{assistant("Copilot", MarkerCoAuthor)}},
		{"Trailer keys are case-insensitive", nil,
			commit("Some fix\n\nco-authored-BY: Claude <NoReply@Anthropic.com>"),
			[]Assistant{assistant("Claude Code", MarkerCoAuthor)}},
		{"A human co-author is not an assistant", nil,
			commit("Some fix\n\nCo-Authored-By: Jane Doe <jane@example.com>"),
			nil},
		{"A lookalike email is not an assistant", nil,
			commit("Some fix\n\n" +
				"Co-Authored-By: Claude <noreply@anthropic.com.example.com>\n" +
				"Co-Authored-By: Claude <fake-noreply@anthropic.com>\n" +
				"Co-authored-by: Codex <codex@openai.co>"),
			nil},
		{"A trailer outside the last paragraph is not read", nil,
			commit("Some fix\n\nCo-Authored-By: Claude <noreply@anthropic.com>\n\nThe previous line is prose, not a trailer."),
			nil},
		{"A co-author trailer in a subject alone is not read", nil,
			commit("Co-Authored-By: Claude <noreply@anthropic.com>"),
			nil},
		{"Windows line endings", nil,
			commit("Some fix\r\n\r\nSome body\r\n\r\nCo-Authored-By: Claude <noreply@anthropic.com>\r\n"),
			[]Assistant{assistant("Claude Code", MarkerCoAuthor)}},

		// Assisted-by
		{"Assisted-by names the assistant", nil,
			commit("Some fix\n\nAssisted-by:   Gemini CLI  "),
			[]Assistant{assistant("Gemini CLI", MarkerAssistedBy)}},
		{"Kernel-style Assisted-by keeps the name before the model", nil,
			commit("Some fix\n\nAssisted-by: Claude:claude-opus-5-5 coccinelle\nSigned-off-by: Jane Doe <jane@example.com>"),
			[]Assistant{assistant("Claude", MarkerAssistedBy)}},
		{"An empty Assisted-by is ignored", nil,
			commit("Some fix\n\nAssisted-by:"),
			nil},

		// Claude-Session
		{"Claude-Session gives Claude Code and its session link", nil,
			commit("Some fix\n\nClaude-Session: https://claude.ai/code/session_123"),
			[]Assistant{withSession(assistant("Claude Code", MarkerSessionTrailer), "https://claude.ai/code/session_123")}},

		// Logins and author
		{"Copilot from its author email", nil,
			commit("Some fix", author("Copilot"), authorEmail("copilot@github.com")),
			[]Assistant{assistant("Copilot", MarkerAuthor)}},
		{"A lookalike author email is not an assistant", nil,
			commit("Some fix", author("Copilot"), authorEmail("copilot@github.com.example.com")),
			nil},
		// The server's case gives the login copilot-swe-agent[bot] to the author Jane
		{"Copilot from its author login", nil,
			commit("Some fix", author("copilot-swe-agent[bot]")),
			[]Assistant{assistant("Copilot", MarkerAuthor)}},
		{"Copilot from its author name as git records it", nil,
			commit("Some fix", author("copilot-swe-agent[bot]"), authorEmail("198982749+Copilot@users.noreply.github.com")),
			[]Assistant{assistant("Copilot", MarkerAuthor)}},
		// The server's case gives the login devin-ai-integration[bot] to the committer
		{"Devin from its committer login", nil,
			commit("Some fix", committer("devin-ai-integration[bot]")),
			[]Assistant{assistant("Devin", MarkerCommitter)}},
		{"Devin from its committer name", nil,
			commit("Some fix", committer("devin-ai-integration[bot]")),
			[]Assistant{assistant("Devin", MarkerCommitter)}},
		// The server's case gives the login devin-ai-integration to the author
		{"A lookalike login is not an assistant", nil,
			commit("Some fix", author("copilot-swe-agent"), committer("devin-ai-integration")),
			nil},

		// Merge
		{"One assistant per commit, its markers merged, the first session link wins", nil,
			commit("Some fix\n\n"+
				"Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>\n"+
				"Claude-Session: https://claude.ai/code/session_1\n"+
				"Co-Authored-By: Claude Sonnet <noreply@anthropic.com>\n"+
				"Claude-Session: https://claude.ai/code/session_2\n"+
				"Assisted-by: Claude Code\n"+
				"Co-authored-by: Copilot <copilot@github.com>",
				authorEmail("copilot@github.com"), committer("copilot-swe-agent[bot]")),
			[]Assistant{
				withSession(assistant("Claude Code", MarkerCoAuthor, MarkerSessionTrailer, MarkerAssistedBy), "https://claude.ai/code/session_1"),
				assistant("Copilot", MarkerCoAuthor, MarkerAuthor, MarkerCommitter),
			}},
		{"Assistants are merged on their name whatever its case", nil,
			commit("Some fix\n\nCo-authored-by: Codex <codex@openai.com>\nAssisted-by: codex:gpt-5"),
			[]Assistant{assistant("Codex", MarkerCoAuthor, MarkerAssistedBy)}},

		// Patterns
		{"Co-author email pattern", &coAuthorPattern,
			commit("Some fix\n\nCo-Authored-By: Review <review-bot@ACME.com>"),
			[]Assistant{assistant("Acme Bot", MarkerCoAuthor)}},
		{"Co-author email pattern, not matching", &coAuthorPattern,
			commit("Some fix\n\nCo-Authored-By: Jane <jane@acme.com>"),
			nil},
		{"Trailer pattern", &trailerPattern,
			commit("Some fix\n\nacme-agent-run: 1234"),
			[]Assistant{assistant("Acme Agent", MarkerTrailer)}},
		{"Trailer pattern, not matching", &trailerPattern,
			commit("Some fix\n\nAcme-Agent-Run-Id: 1234"),
			nil},
		{"Author email pattern", &authorEmailPattern,
			commit("Some fix", authorEmail("bot+fixer@acme.com")),
			[]Assistant{assistant("Acme Bot", MarkerAuthor)}},
		{"Author email pattern, not matching", &authorEmailPattern,
			commit("Some fix", authorEmail("jane@acme.com")),
			nil},
		// The server's case gives the login acme-bot to the author
		{"Login pattern", &loginPattern,
			commit("Some fix", author("acme-bot"), committer("acme-bot")),
			[]Assistant{assistant("Acme Bot", MarkerAuthor, MarkerCommitter)}},
		{"Login pattern, not matching", &loginPattern,
			commit("Some fix", author("acme-bot-2")),
			nil},
		{"Patterns add to the built-ins", &loginPattern,
			commit("Some fix\n\nCo-Authored-By: Claude <noreply@anthropic.com>", author("acme-bot")),
			[]Assistant{assistant("Claude Code", MarkerCoAuthor), assistant("Acme Bot", MarkerAuthor)}},
		{"Built-ins switched off", &builtInsOff,
			commit("Some fix\n\n"+
				"Co-Authored-By: Claude <noreply@anthropic.com>\n"+
				"Assisted-by: Codex\n"+
				"Claude-Session: https://claude.ai/code/session_1",
				author("acme-bot"), authorEmail("copilot@github.com"), committer("devin-ai-integration[bot]")),
			[]Assistant{assistant("Acme Bot", MarkerAuthor)}},

		// Specific to the CLI
		{"A login matches the name whatever its case", nil,
			commit("Some fix", author("Copilot-SWE-Agent[bot]")),
			[]Assistant{assistant("Copilot", MarkerAuthor)}},
		{"A GitHub noreply address is not recognised, as on the server", nil,
			commit("Some fix\n\nCo-authored-by: Copilot <198982749+Copilot@users.noreply.github.com>"),
			nil},
		{"The committer email is not read, as on the server", nil,
			commit("Some fix", committerEmail("copilot@github.com")),
			nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := builtIns
			if c.settings != nil {
				r = rules(t, *c.settings)
			}
			assert.Equal(t, c.expected, r.Assistants(c.commit))
		})
	}
}

// The validation of the patterns of the server, which the CLI turns into
// warnings: an invalid pattern is skipped.
func TestPatternWarnings(t *testing.T) {
	_, warnings := NewRules(Settings{BuiltInConventions: true, Patterns: []Pattern{
		{Name: "Acme Bot", Type: PatternCoAuthorEmail, Value: `.*@acme\.com`},
		{Name: "Acme Bot", Type: PatternTrailer, Value: "Acme-Agent"},
		{Name: "Acme Bot", Type: PatternAuthorEmail, Value: `bot@acme\.com`},
		{Name: "Acme Bot", Type: PatternLogin, Value: "acme-bot[bot]"},
	}})
	assert.Empty(t, warnings, "Valid patterns")

	_, warnings = NewRules(Settings{BuiltInConventions: true, Patterns: []Pattern{
		{Name: "Acme Bot", Type: PatternCoAuthorEmail, Value: "*@acme.com"},
		{Name: "Acme Bot", Type: PatternAuthorEmail, Value: "bot["},
		{Name: " ", Type: PatternLogin, Value: "acme-bot"},
		{Name: "Acme Bot", Type: PatternLogin, Value: ""},
		{Name: "Acme Bot", Type: PatternTrailer, Value: "Acme Agent"},
	}})
	assert.Equal(t, []string{
		"Pattern #1 (Acme Bot): invalid regular expression `*@acme.com`",
		"Pattern #2 (Acme Bot): invalid regular expression `bot[`",
		"Pattern #3: the name is required",
		"Pattern #4 (Acme Bot): the value is required",
		"Pattern #5 (Acme Bot): `Acme Agent` is not a trailer key",
	}, warnings, "Invalid patterns")
}

func TestInvalidPatternsAreSkipped(t *testing.T) {
	r, warnings := NewRules(Settings{BuiltInConventions: true, Patterns: []Pattern{
		{Name: "Broken", Type: PatternAuthorEmail, Value: "bot["},
		{Name: "Acme Bot", Type: PatternAuthorEmail, Value: `bot@acme\.com`},
	}})

	assert.Len(t, warnings, 1)
	assert.Equal(t, []Assistant{assistant("Acme Bot", MarkerAuthor)},
		r.Assistants(commit("Some fix", authorEmail("bot@acme.com"))))
}

// Yontrack compiles the patterns with Java: one that Go's RE2 cannot compile,
// like a lookaround, is skipped with a warning, and the others still apply.
func TestRE2IncompatiblePatternIsSkipped(t *testing.T) {
	r, warnings := NewRules(Settings{BuiltInConventions: true, Patterns: []Pattern{
		{Name: "Lookbehind", Type: PatternCoAuthorEmail, Value: `.*(?<!human)@acme\.com`},
		{Name: "Acme Bot", Type: PatternCoAuthorEmail, Value: `.*-bot@acme\.com`},
	}})

	assert.Equal(t, []string{"Pattern #1 (Lookbehind): invalid regular expression `.*(?<!human)@acme\\.com`"}, warnings)
	assert.Equal(t, []Assistant{assistant("Acme Bot", MarkerCoAuthor)},
		r.Assistants(commit("Some fix\n\nCo-Authored-By: Review <review-bot@acme.com>")))
}

// A type this CLI does not know - a newer Yontrack - is skipped too.
func TestUnknownPatternTypeIsSkipped(t *testing.T) {
	_, warnings := NewRules(Settings{BuiltInConventions: true, Patterns: []Pattern{
		{Name: "Acme Bot", Type: "COMMITTER_EMAIL", Value: "bot@acme.com"},
	}})

	assert.Equal(t, []string{"Pattern #1 (Acme Bot): unknown type `COMMITTER_EMAIL`"}, warnings)
}

// A regular expression matches the whole email, even with an alternation.
func TestPatternMatchesTheWholeEmail(t *testing.T) {
	r := rules(t, Settings{BuiltInConventions: false, Patterns: []Pattern{
		{Name: "Acme Bot", Type: PatternAuthorEmail, Value: `bot@acme\.com|robot@acme\.com`},
	}})

	assert.Nil(t, r.Assistants(commit("Some fix", authorEmail("bot@acme.com.example.com"))))
	assert.Nil(t, r.Assistants(commit("Some fix", authorEmail("a-robot@acme.com"))))
	assert.Equal(t, []Assistant{assistant("Acme Bot", MarkerAuthor)},
		r.Assistants(commit("Some fix", authorEmail("robot@acme.com"))))
}

// The values of the agent-markers settings, as settingsById returns them.
func TestParseSettings(t *testing.T) {
	settings, err := ParseSettings([]byte(`{"builtInConventions": false, "patterns": [{"name": "Acme Bot", "type": "LOGIN", "value": "acme-bot"}]}`))

	assert.NoError(t, err)
	assert.Equal(t, Settings{BuiltInConventions: false, Patterns: []Pattern{{Name: "Acme Bot", Type: PatternLogin, Value: "acme-bot"}}}, settings)
}

// The built-in conventions are on unless the settings say otherwise.
func TestParseSettingsDefaults(t *testing.T) {
	settings, err := ParseSettings([]byte(`{}`))

	assert.NoError(t, err)
	assert.Equal(t, DefaultSettings(), settings)
	assert.True(t, settings.BuiltInConventions)
}

// The assisted change of a range of commits, as Yontrack computes it from a
// change log: every commit counts, the assistants are distinct and sorted, the
// session links distinct.
func TestSummarize(t *testing.T) {
	change := Summarize([]Commit{
		commit("Fix\n\nCo-authored-by: Codex <codex@openai.com>"),
		commit("Fix\n\nCo-Authored-By: Claude <noreply@anthropic.com>\nClaude-Session: https://claude.ai/code/session_1"),
		commit("Plain"),
		commit("Fix\n\nClaude-Session: https://claude.ai/code/session_1"),
		commit("Fix\n\nClaude-Session: https://claude.ai/code/session_2"),
	}, builtIns)

	assert.Equal(t, Change{
		Assistants:      []string{"Claude Code", "Codex"},
		AssistedCommits: 4,
		TotalCommits:    5,
		SessionLinks:    []string{"https://claude.ai/code/session_1", "https://claude.ai/code/session_2"},
	}, change)
}

// "Looked and found none" is a fact too: empty lists, not missing ones.
func TestSummarizeNoAssistant(t *testing.T) {
	change := Summarize([]Commit{commit("Plain"), commit("Another")}, builtIns)

	assert.Equal(t, Change{
		Assistants:      []string{},
		AssistedCommits: 0,
		TotalCommits:    2,
		SessionLinks:    []string{},
	}, change)
}
