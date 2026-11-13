package summon

// Where fetched files wait and what a checked one is: summon owns every file
// it brings in, from fetch to quarantine to validation, and hands the caller
// checked files as values.

import (
	"fmt"
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

// File is a fetched file, read once: the checks and the install all use
// Data. Profile is the decoded deck of a .profile.toml; a spec leaves it empty.
type File struct {
	Path    string
	Info    fs.FileInfo
	Data    []byte
	Profile profile.File
}

// Read reads the regular file at path, up to ProfileMaxSize. Info is set
// whenever the file could be opened.
func Read(path string) (File, error) {
	f := File{Path: path}
	fh, err := os.Open(path)
	if err != nil {
		return f, err
	}
	defer fh.Close()
	if f.Info, err = fh.Stat(); err != nil {
		return f, err
	}
	if !f.Info.Mode().IsRegular() {
		return f, fmt.Errorf("%s is not a regular file", path)
	}
	data, err := io.ReadAll(io.LimitReader(fh, ProfileMaxSize+1))
	if err != nil {
		return f, err
	}
	if len(data) > ProfileMaxSize {
		return f, fmt.Errorf("file too large (max %d bytes)", ProfileMaxSize)
	}
	f.Data = data
	return f, nil
}

// Commands lists every command the deck holds, as written, every platform's
// variant included (profile.WrittenCommands).
func (f File) Commands() ([]profile.Written, error) {
	return profile.WrittenCommands(f.Data)
}
