package main

// Tests for install.sh. Each run gets a temp HOME and a PATH holding only
// stand-in curl, wget and uname plus the real tools the script needs, so the
// script never reaches the network and its tool list is pinned down here.

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const releases = "https://github.com/lucky7xz/drako/releases/"

// The real tools install.sh may call. curl, wget and uname are stand-ins.
var installTools = []string{"sh", "awk", "cat", "chmod", "cp", "gzip", "id", "mkdir", "mktemp", "mv", "rm", "sha256sum", "tar"}

// fakeCurl serves $FAKE_RELEASES/<tag or latest>/<file> for release URLs and
// logs each URL to $FAKE_LOG. A missing file fails like curl -f does.
const fakeCurl = `#!/bin/sh
out= url=
while [ $# -gt 0 ]; do
	case "$1" in
	-o | -O | --proto) [ "$1" = --proto ] || out=$2; shift 2 ;;
	-*) shift ;;
	*) url=$1; shift ;;
	esac
done
echo "$url" >> "$FAKE_LOG"
case "$url" in
` + releases + `latest/download/*) f="$FAKE_RELEASES/latest/${url##*/}" ;;
` + releases + `download/*) rest=${url#` + releases + `download/}; f="$FAKE_RELEASES/$rest" ;;
*) exit 22 ;;
esac
[ -f "$f" ] || exit 22
cat "$f" > "$out"
`

const fakeUname = `#!/bin/sh
case "$1" in -s) echo "$FAKE_OS" ;; -m) echo "$FAKE_ARCH" ;; *) echo "$FAKE_OS" ;; esac
`

// The "binary" inside the fake archives.
func fakeDrako(tag string) []byte {
	return []byte("#!/bin/sh\necho \"lucky7xz/drako " + tag + "\"\n")
}

type installEnv struct {
	t        *testing.T
	shell    string // absolute path to the shell that runs install.sh
	home     string
	bin      string // the only folder on PATH
	releases string
	log      string
	os, arch string
	extra    []string // more environment, e.g. DRAKO_VERSION=v1.0.0
}

func newInstallEnv(t *testing.T, downloader string) *installEnv {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("install.sh is for Linux, macOS and FreeBSD")
	}
	shell, err := exec.LookPath("dash") // stricter than bash about POSIX
	if err != nil {
		shell, err = exec.LookPath("sh")
		if err != nil {
			t.Skip("no sh")
		}
	}
	dir := t.TempDir()
	e := &installEnv{
		t: t, shell: shell,
		home:     filepath.Join(dir, "home"),
		bin:      filepath.Join(dir, "bin"),
		releases: filepath.Join(dir, "releases"),
		log:      filepath.Join(dir, "urls.log"),
		os:       "Linux", arch: "x86_64",
	}
	for _, d := range []string{e.home, e.bin, e.releases} {
		must(t, os.MkdirAll(d, 0o755))
	}
	for _, tool := range installTools {
		p, err := exec.LookPath(tool)
		if err != nil {
			t.Skipf("%s not found", tool)
		}
		must(t, os.Symlink(p, filepath.Join(e.bin, tool)))
	}
	must(t, os.WriteFile(filepath.Join(e.bin, downloader), []byte(fakeCurl), 0o755))
	must(t, os.WriteFile(filepath.Join(e.bin, "uname"), []byte(fakeUname), 0o755))
	return e
}

// publish puts a release under tag (and under "latest" when latest is set),
// with one archive per asset and a checksums.txt listing them all.
func (e *installEnv) publish(tag string, latest bool, assets ...string) {
	e.t.Helper()
	var sums strings.Builder
	files := map[string][]byte{}
	for _, a := range assets {
		data := tarGz(e.t, "drako", fakeDrako(tag))
		files[a] = data
		fmt.Fprintf(&sums, "%x  %s\n", sha256.Sum256(data), a)
	}
	files["checksums.txt"] = []byte(sums.String())
	dirs := []string{tag}
	if latest {
		dirs = append(dirs, "latest")
	}
	for _, d := range dirs {
		must(e.t, os.MkdirAll(filepath.Join(e.releases, d), 0o755))
		for name, data := range files {
			must(e.t, os.WriteFile(filepath.Join(e.releases, d, name), data, 0o644))
		}
	}
}

// run feeds script to the shell on stdin, as curl | sh does.
func (e *installEnv) run(script []byte) (string, error) {
	cmd := exec.Command(e.shell)
	cmd.Stdin = bytes.NewReader(script)
	cmd.Dir = e.home
	cmd.Env = append([]string{
		"HOME=" + e.home,
		"PATH=" + e.bin,
		"FAKE_RELEASES=" + e.releases,
		"FAKE_LOG=" + e.log,
		"FAKE_OS=" + e.os,
		"FAKE_ARCH=" + e.arch,
	}, e.extra...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (e *installEnv) urls() []string {
	data, err := os.ReadFile(e.log)
	if os.IsNotExist(err) {
		return nil
	}
	must(e.t, err)
	return strings.Fields(string(data))
}

func installScript(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("install.sh")
	must(t, err)
	return data
}

func TestInstallLatest(t *testing.T) {
	e := newInstallEnv(t, "curl")
	e.publish("v1.0.0", true, "drako_Linux_x86_64.tar.gz", "drako_Linux_arm64.tar.gz")

	out, err := e.run(installScript(t))
	if err != nil {
		t.Fatalf("install failed: %v\n%s", err, out)
	}
	bin := filepath.Join(e.home, ".local", "bin", "drako")
	got, err := os.ReadFile(bin)
	must(t, err)
	if !bytes.Equal(got, fakeDrako("v1.0.0")) {
		t.Errorf("installed file is not the one from the archive")
	}
	if fi, _ := os.Stat(bin); fi.Mode().Perm() != 0o755 {
		t.Errorf("mode = %v, want 0755", fi.Mode().Perm())
	}
	for _, want := range []string{"installed " + bin, "lucky7xz/drako v1.0.0", "is not on your PATH"} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	wantURLs := []string{
		releases + "latest/download/drako_Linux_x86_64.tar.gz",
		releases + "latest/download/checksums.txt",
	}
	if got := e.urls(); strings.Join(got, " ") != strings.Join(wantURLs, " ") {
		t.Errorf("fetched %v, want %v", got, wantURLs)
	}
	if left, _ := filepath.Glob(filepath.Join(e.home, ".local", "bin", ".drako*")); len(left) > 0 {
		t.Errorf("temp file left behind: %v", left)
	}
}

func TestInstallVersionAndDirWithWget(t *testing.T) {
	e := newInstallEnv(t, "wget")
	e.publish("v0.9.0", false, "drako_Darwin_arm64.tar.gz")
	e.publish("v1.0.0", true, "drako_Darwin_arm64.tar.gz")
	e.os, e.arch = "Darwin", "arm64"
	dir := filepath.Join(e.home, "my bin") // a space, and not there yet
	e.extra = []string{"DRAKO_VERSION=0.9.0", "DRAKO_INSTALL_DIR=" + dir}

	out, err := e.run(installScript(t))
	if err != nil {
		t.Fatalf("install failed: %v\n%s", err, out)
	}
	got, err := os.ReadFile(filepath.Join(dir, "drako"))
	must(t, err)
	if !bytes.Equal(got, fakeDrako("v0.9.0")) {
		t.Errorf("installed %q, want v0.9.0", got)
	}
	if u := e.urls(); len(u) != 2 || !strings.HasPrefix(u[0], releases+"download/v0.9.0/") {
		t.Errorf("fetched %v, want v0.9.0 files", u)
	}
}

func TestInstallRefusesBadDownloads(t *testing.T) {
	cases := []struct {
		name  string
		setup func(e *installEnv)
		want  string
	}{
		{"checksum mismatch", func(e *installEnv) {
			e.publish("v1.0.0", true, "drako_Linux_x86_64.tar.gz")
			f := filepath.Join(e.releases, "latest", "drako_Linux_x86_64.tar.gz")
			must(t, os.WriteFile(f, tarGz(t, "drako", []byte("#!/bin/sh\necho evil\n")), 0o644))
		}, "checksum mismatch"},
		{"asset missing from release", func(e *installEnv) {
			e.publish("v1.0.0", true, "drako_Linux_arm64.tar.gz")
		}, "download failed"},
		{"asset missing from checksums", func(e *installEnv) {
			e.publish("v1.0.0", true, "drako_Linux_x86_64.tar.gz")
			must(t, os.WriteFile(filepath.Join(e.releases, "latest", "checksums.txt"), []byte("00  drako_Linux_x86_64.tar.gz.bak\n"), 0o644))
		}, "lists no drako_Linux_x86_64.tar.gz"},
		{"version that does not exist", func(e *installEnv) {
			e.publish("v1.0.0", true, "drako_Linux_x86_64.tar.gz")
			e.extra = []string{"DRAKO_VERSION=v9.9.9"}
		}, "download failed"},
		{"version that is not a version", func(e *installEnv) {
			e.extra = []string{"DRAKO_VERSION=../../evil"}
		}, "should look like v0.5.0"},
		{"armv6", func(e *installEnv) { e.arch = "armv6l" }, "no release for Linux on armv6l"},
		{"FreeBSD on arm64", func(e *installEnv) { e.os, e.arch = "FreeBSD", "arm64" }, "no release for FreeBSD on arm64"},
		{"Windows", func(e *installEnv) { e.os = "MINGW64_NT-10.0" }, "Linux, macOS and FreeBSD only"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := newInstallEnv(t, "curl")
			c.setup(e)
			out, err := e.run(installScript(t))
			if err == nil {
				t.Fatalf("install succeeded, want failure:\n%s", out)
			}
			if !strings.Contains(out, c.want) {
				t.Errorf("output lacks %q:\n%s", c.want, out)
			}
			if _, err := os.Stat(filepath.Join(e.home, ".local", "bin", "drako")); err == nil {
				t.Errorf("drako was installed")
			}
		})
	}
}

// curl | sh runs whatever arrived. Cut the script after every line: no part
// short of the whole may download or install anything.
func TestInstallCutOffRunsNothing(t *testing.T) {
	e := newInstallEnv(t, "curl")
	e.publish("v1.0.0", true, "drako_Linux_x86_64.tar.gz")
	script := installScript(t)
	lines := bytes.SplitAfter(script, []byte("\n"))
	for n := range lines {
		part := bytes.Join(lines[:n], nil)
		if bytes.Equal(part, script) {
			break
		}
		e.run(part)
		if u := e.urls(); len(u) > 0 {
			t.Fatalf("the first %d lines fetched %v", n, u)
		}
		if _, err := os.Stat(filepath.Join(e.home, ".local", "bin")); err == nil {
			t.Fatalf("the first %d lines created ~/.local/bin", n)
		}
	}
}

func TestInstallWarnsAboutAnotherDrako(t *testing.T) {
	e := newInstallEnv(t, "curl")
	e.publish("v1.0.0", true, "drako_Linux_x86_64.tar.gz")
	must(t, os.WriteFile(filepath.Join(e.bin, "drako"), fakeDrako("old"), 0o755))

	out, err := e.run(installScript(t))
	if err != nil {
		t.Fatalf("install failed: %v\n%s", err, out)
	}
	if want := "your shell runs " + filepath.Join(e.bin, "drako") + " first"; !strings.Contains(out, want) {
		t.Errorf("output lacks %q:\n%s", want, out)
	}
}

func tarGz(t *testing.T, name string, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(zw)
	must(t, tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(data)), Typeflag: tar.TypeReg}))
	_, err := tw.Write(data)
	must(t, err)
	must(t, tw.Close())
	must(t, zw.Close())
	return buf.Bytes()
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
