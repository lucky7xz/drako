package host

import (
	"os"
	"os/user"
	"sync"
)

// WorkDir is the directory drako is working in.
func WorkDir() (string, error) { return os.Getwd() }

// HomeDir is the user's home directory, or "" when there is none to find.
func HomeDir() string {
	home, _ := os.UserHomeDir()
	return home
}

// Enter makes dir drako's working directory and returns where that is:
// reading the path back resolves "..", links and relative parts.
func Enter(dir string) (string, error) {
	if err := os.Chdir(dir); err != nil {
		return "", err
	}
	return os.Getwd()
}

// Dirs lists the folders inside path, by name, sorted.
func Dirs(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// Hostname is the machine's name, asked once per process.
var Hostname = sync.OnceValue(func() string {
	name, _ := os.Hostname()
	return name
})

// Username is the name of the user drako runs as, asked once per process:
// from the system's user database, else from $USER.
var Username = sync.OnceValue(func() string {
	if current, _ := user.Current(); current != nil {
		return current.Username
	}
	return os.Getenv("USER")
})
