package summon

// Where fetched files wait and what a checked one is: summon owns every file
// it brings in, from fetch to quarantine to validation, and hands the caller
// checked files as values.

import (
	"io"
	"io/fs"
	"os"

	"github.com/lucky7xz/drako/internal/profile"
)

// Kind is how a source is fetched.
type Kind int

const (
	Folder Kind = iota // a folder on disk, read in place
	Git                // a repository, cloned into a quarantine
	HTTP               // one file, downloaded into a quarantine
)

// Classify says how source is fetched: a git address, else a folder that is
// on disk, else a file over HTTP.
func Classify(source string) Kind {
	if IsGitURL(source) {
		return Git
	}
	if info, err := os.Stat(source); err == nil && info.IsDir() {
		return Folder
	}
	return HTTP
}

// Quarantine is a private temporary folder where fetched files wait until
// they are checked. A fresh one per summon, so one a Ctrl-C left behind
// cannot get in the way. Close deletes it and everything in it.
type Quarantine struct {
	Dir string
}

// NewQuarantine makes an empty quarantine.
func NewQuarantine() (*Quarantine, error) {
	dir, err := os.MkdirTemp("", "drako-summon-*")
	if err != nil {
		return nil, err
	}
	return &Quarantine{Dir: dir}, nil
}

// Close deletes the quarantine.
func (q *Quarantine) Close() error {
	return os.RemoveAll(q.Dir)
}

// File is a fetched file that passed its checks, waiting in a quarantine or
// in the folder it is summoned from. Profile is the decoded deck of a
// .profile.toml; a spec leaves it empty.
type File struct {
	Path    string
	Info    fs.FileInfo
	Profile profile.File
}

// Open opens the file for installing.
func (f File) Open() (io.ReadCloser, error) {
	return os.Open(f.Path)
}

// Commands lists every command the deck holds, as written, every platform's
// variant included (profile.WrittenCommands).
func (f File) Commands() ([]profile.Written, error) {
	raw, err := os.ReadFile(f.Path)
	if err != nil {
		return nil, err
	}
	return profile.WrittenCommands(raw)
}
