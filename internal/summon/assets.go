package summon

// Assets declared by summoned profiles: planning what would be copied,
// walking them under hard size/count limits, and the path-safety checks that
// keep a hostile repo from reading outside its own tree. Writing an asset
// into drako's folder is not done here: each allowed file is handed to the
// caller's sink.

import (
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/lucky7xz/drako/internal/profile"
)

// Asset copy limits
const (
	assetWarnSizeBytes = 1 * 1024 * 1024  // 1 MB warn
	AssetMaxFileBytes  = 5 * 1024 * 1024  // 5 MB per file hard limit
	AssetMaxTotalBytes = 50 * 1024 * 1024 // 50 MB total hard limit
	AssetMaxFileCount  = 500              // safety cap
)

// ProfileAssets returns the profile's declared assets (relative paths).
func ProfileAssets(profile profile.File) []string {
	if profile.Assets == nil {
		return nil
	}
	var out []string
	for _, raw := range *profile.Assets {
		if s := strings.TrimSpace(raw); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// AssetSink receives what CopyAssets finds. Install is handed each allowed
// file under the name it should have inside the profile's assets folder;
// Large is told about a file over the warning size just before it is handed
// over (inFolder: it came from a declared folder, and name is relative to it).
type AssetSink struct {
	Install func(rel string, r io.Reader) error
	Large   func(name string, size int64, inFolder bool)
}

// CopyAssets walks the assets a profile declares (paths relative to
// profileDir, which must stay inside repoRoot) and hands every file the
// limits allow to sink. It returns counts of copied/skipped/missing and the
// total bytes copied.
func CopyAssets(repoRoot, profileDir string, assets []string, sink AssetSink) (int, int, int, int64) {
	var copied, skipped, missing int
	var totalBytes int64
	var fileCount int

	for _, rel := range assets {
		// Resolve asset path relative to profileDir
		cleanRel, safe := cleanAssetRel(rel)
		if !safe {
			log.Printf("assets: skipping unsafe relative path %s", rel)
			skipped++
			continue
		}
		src := filepath.Join(profileDir, cleanRel)
		// Missing first, as in PlanAssets, so plan and summary agree.
		info, statErr := os.Stat(src)
		if statErr != nil {
			log.Printf("assets: missing %s", rel)
			missing++
			continue
		}
		// Ensure src is within repoRoot
		ok, err := IsPathWithinBase(repoRoot, src)
		if err != nil || !ok {
			log.Printf("assets: skipping unsafe path %s (err=%v)", rel, err)
			skipped++
			continue
		}
		if info.IsDir() {
			c, s, m, _ := copyDirWithLimits(src, cleanRel, sink, &fileCount, &totalBytes)
			copied += c
			skipped += s
			missing += m
			continue
		}
		// Single file copy with limits
		if !checkAssetFileAllowed(info.Size(), fileCount, totalBytes) {
			log.Printf("assets: skipping (limits) %s", rel)
			skipped++
			continue
		}
		if info.Size() > assetWarnSizeBytes {
			sink.Large(rel, info.Size(), false)
		}
		if err := hand(sink, cleanRel, src); err != nil {
			log.Printf("assets: copy failed %s -> %s: %v", src, cleanRel, err)
			skipped++
			continue
		}
		copied++
		fileCount++
		totalBytes += info.Size()
	}
	return copied, skipped, missing, totalBytes
}

// AssetPlan describes what will be copied for an asset
type AssetPlan struct {
	AssetRel  string
	DestRel   string
	IsDir     bool
	FileCount int
	Bytes     int64
	Missing   bool // declared, but not there
	Refused   bool // its path leaves the repository; never read, never copied
}

// PlanAssets enumerates assets to present a copy plan before confirmation
func PlanAssets(repoRoot, profileDir string, assets []string) []AssetPlan {
	var plans []AssetPlan
	for _, rel := range assets {
		cleanRel, safe := cleanAssetRel(rel)
		if !safe {
			plans = append(plans, AssetPlan{AssetRel: rel, DestRel: cleanRel, Refused: true})
			continue
		}
		src := filepath.Join(profileDir, cleanRel)
		// Missing is asked first: a path that is not there cannot be resolved,
		// so the within-the-repository check would call it unsafe.
		info, statErr := os.Stat(src)
		if statErr != nil {
			plans = append(plans, AssetPlan{AssetRel: rel, DestRel: cleanRel, Missing: true})
			continue
		}
		ok, err := IsPathWithinBase(repoRoot, src)
		if err != nil || !ok {
			plans = append(plans, AssetPlan{AssetRel: rel, DestRel: cleanRel, Refused: true})
			continue
		}
		if info.IsDir() {
			// Walk dir to count files/bytes
			var files int
			var bytes int64
			filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if !d.Type().IsRegular() {
					return nil
				}
				fi, e := d.Info()
				if e != nil {
					return nil
				}
				files++
				bytes += fi.Size()
				return nil
			})
			plans = append(plans, AssetPlan{
				AssetRel:  rel,
				DestRel:   cleanRel,
				IsDir:     true,
				FileCount: files,
				Bytes:     bytes,
				Missing:   false,
			})
		} else {
			plans = append(plans, AssetPlan{
				AssetRel:  rel,
				DestRel:   cleanRel,
				IsDir:     false,
				FileCount: 1,
				Bytes:     info.Size(),
				Missing:   false,
			})
		}
	}
	return plans
}

// cleanAssetRel normalizes an asset relative path and ensures it is safe (no abs, no parent escapes)
func cleanAssetRel(rel string) (string, bool) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return "", false
	}
	// Backslashes too: they separate paths on Windows.
	if strings.Contains(rel, `\`) || !filepath.IsLocal(rel) {
		return "", false
	}
	return filepath.Clean(rel), true
}

// hand opens the asset file at src and gives it to the sink under the name rel.
func hand(sink AssetSink, rel, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	return sink.Install(rel, in)
}

// copyDirWithLimits recursively hands the files of srcDir to sink, with
// size/count limits, under names that start with base.
func copyDirWithLimits(srcDir, base string, sink AssetSink, fileCount *int, totalBytes *int64) (int, int, int, int64) {
	copied := 0
	skipped := 0
	missing := 0
	var copiedBytes int64
	err := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		// Regular files only: a symlink could point outside the repo or at a device.
		if !d.Type().IsRegular() {
			skipped++
			return nil
		}
		info, serr := d.Info()
		if serr != nil {
			return serr
		}
		if !checkAssetFileAllowed(info.Size(), *fileCount, *totalBytes) {
			skipped++
			return nil
		}
		rel, rerr := filepath.Rel(srcDir, path)
		if rerr != nil {
			skipped++
			return nil
		}
		if info.Size() > assetWarnSizeBytes {
			sink.Large(rel, info.Size(), true)
		}
		if err := hand(sink, filepath.Join(base, rel), path); err != nil {
			skipped++
			return nil
		}
		*fileCount++
		*totalBytes += info.Size()
		copiedBytes += info.Size()
		copied++
		return nil
	})
	if err != nil {
		log.Printf("assets: copy dir error %s: %v", srcDir, err)
	}
	return copied, skipped, missing, copiedBytes
}

// checkAssetFileAllowed enforces per-file and aggregate limits
func checkAssetFileAllowed(size int64, fileCount int, totalBytes int64) bool {
	if size > AssetMaxFileBytes {
		return false
	}
	if fileCount+1 > AssetMaxFileCount {
		return false
	}
	if totalBytes+size > AssetMaxTotalBytes {
		return false
	}
	return true
}

// IsPathWithinBase checks if target is within base after resolving symlinks
func IsPathWithinBase(base, target string) (bool, error) {
	baseAbs, err := filepath.Abs(base)
	if err != nil {
		return false, err
	}

	// Resolve the base too, so both sides are compared in the same form.
	// Where /home is a symlink — Fedora atomic points it at /var/home — an
	// unresolved base can never contain a resolved target, and every asset
	// reads as [missing]:
	//   base    /home/you/decks
	//   target  /var/home/you/decks/serve.sh
	// The target is still resolved in full, so the escape check below holds.
	if resolvedBase, rerr := filepath.EvalSymlinks(baseAbs); rerr == nil {
		baseAbs = resolvedBase
	}

	// Use EvalSymlinks to resolve the true final path.
	// Example: If the repo contains a file 'script.sh' which is actually a symlink
	// pointing to '../../../../etc/passwd', EvalSymlinks reveals that true path.
	// We then check if that resolved path is still inside the repo folder.
	// If it points outside, we reject it to prevent stealing system files.
	targetResolved, err := filepath.EvalSymlinks(target)
	if err != nil {
		// If it doesn't exist, EvalSymlinks fails. We can fallback to Abs + Clean if we are creating it,
		// but for existing source files in the repo, it MUST exist.
		// If checking destination, it might not exist yet.
		// However, this function is primarily used to check if a SOURCE file inside the repo
		// is actually safe to copy (i.e. it doesn't point outside the repo).
		return false, err
	}

	targetAbs, err := filepath.Abs(targetResolved)
	if err != nil {
		return false, err
	}

	rel, err := filepath.Rel(baseAbs, targetAbs)
	if err != nil {
		return false, err
	}
	// Must not start with ".."
	return !strings.HasPrefix(rel, ".."), nil
}
