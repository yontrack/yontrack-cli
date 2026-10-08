package assistants

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Separators of the fields of a commit, and of the commits: neither can be in
// a commit message.
const (
	fieldSeparator  = "\x1f"
	commitSeparator = "\x00"
)

// gitLogFormat is hash, author name and email, committer name and email, and
// the full message.
var gitLogFormat = strings.Join([]string{"%H", "%an", "%ae", "%cn", "%ce", "%B"}, "%x1f")

// GitLog reads the commits of a range `A..B` of the git repository in dir -
// the current directory when empty - by running git, which must be
// installed. Every commit of the range is read, merges included, as Yontrack
// does for a change log.
func GitLog(dir string, revisionRange string) ([]Commit, error) {
	if !isRange(revisionRange) {
		return nil, fmt.Errorf("expected a range <previous-commit>..<commit>, got %s", revisionRange)
	}
	git, err := exec.LookPath("git")
	if err != nil {
		return nil, fmt.Errorf("git is required to read the commits of %s: %w", revisionRange, err)
	}

	cmd := exec.Command(git, "log", "-z", "--no-show-signature", "--format="+gitLogFormat, revisionRange, "--")
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && stderr.Len() > 0 {
			return nil, fmt.Errorf("cannot read the commits of %s: %s", revisionRange, strings.TrimSpace(stderr.String()))
		}
		return nil, fmt.Errorf("cannot read the commits of %s: %w", revisionRange, err)
	}

	var commits []Commit
	for _, record := range strings.Split(stdout.String(), commitSeparator) {
		if record == "" {
			continue
		}
		fields := strings.SplitN(record, fieldSeparator, 6)
		if len(fields) != 6 {
			return nil, fmt.Errorf("cannot read the commits of %s: unexpected output of git log", revisionRange)
		}
		commits = append(commits, Commit{
			Hash:           fields[0],
			AuthorName:     fields[1],
			AuthorEmail:    fields[2],
			CommitterName:  fields[3],
			CommitterEmail: fields[4],
			Message:        fields[5],
		})
	}
	return commits, nil
}

// isRange tells whether a value is an explicit `A..B` range, and nothing git
// could take for an option.
func isRange(value string) bool {
	from, to, found := strings.Cut(value, "..")
	return found &&
		from != "" && to != "" &&
		!strings.HasPrefix(from, "-") && !strings.HasPrefix(to, "-") &&
		!strings.HasPrefix(to, ".") && !strings.Contains(to, "..")
}
