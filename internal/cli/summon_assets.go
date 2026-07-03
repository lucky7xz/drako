package cli

// Assets declared by summoned profiles: planning what would be copied,
// copying under hard size/count limits, and the path-safety checks that keep
// a hostile repo from writing or reading outside its own tree.

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/paths"
)

// Asset copy limits
const (
	assetWarnSizeBytes = 1 * 1024 * 1024  // 1 MB warn
	assetMaxFileBytes  = 5 * 1024 * 1024  // 5 MB per file hard limit
	assetMaxTotalBytes = 50 * 1024 * 1024 // 50 MB total hard limit
	assetMaxFileCount  = 500              // safety cap
)

// readAssetsFromProfile parses a profile file and returns declared assets (relative paths)
func readAssetsFromProfile(profilePath string) ([]string, error) {
	data, err := os.ReadFile(profilePath)
	if err != nil {
		return nil, err
	}
	var profile config.ProfileFile
	if _, err := toml.Decode(string(data), &profile); err != nil {
		return nil, err
	}
	if profile.Assets == nil {
		return nil, nil
	}
	var out []string
	for _, raw := range *profile.Assets {
		s := strings.TrimSpace(raw)
		if s == "" {
			continue
		}
		// Clean path to normalize separators and remove leading "./"
		s = filepath.Clean(s)
		s = strings.TrimPrefix(s, "./")
		out = append(out, s)
	}
	return out, nil
}

// copyAssetsList copies a list of assets (files or directories) from the cloned repo to configDir.
// - repoRoot: tempDir where repo was cloned
// - profileDir: directory of the profile file (assets are relative to this)
// - profileName: name of the profile (used for subfolder isolation)
// Returns counts of copied/skipped/missing and total bytes copied.
func copyAssetsList(repoRoot, profileDir string, assets []string, profileName string) (int, int, int, int64) {
	configDir, err := paths.ConfigDir()
	if err != nil {
		log.Printf("assets: could not resolve config dir: %v", err)
		return 0, 0, len(assets), 0
	}
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
		// Ensure src is within repoRoot
		ok, err := isPathWithinBase(repoRoot, src)
		if err != nil || !ok {
			log.Printf("assets: skipping unsafe path %s (err=%v)", rel, err)
			skipped++
			continue
		}
		// Destination is the assets/ directory + profile name + original relative path
		// e.g. ~/.config/drako/assets/my-profile/script.sh
		dst := filepath.Join(paths.AssetsDir(configDir, profileName), cleanRel)

		info, statErr := os.Stat(src)
		if statErr != nil {
			log.Printf("assets: missing %s", rel)
			missing++
			continue
		}
		if info.IsDir() {
			c, s, m, _ := copyDirWithLimits(src, dst, &fileCount, &totalBytes)
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
			fmt.Printf("  ⚠️  Large asset: %s (%.1f MB)\n", rel, float64(info.Size())/(1024*1024))
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			log.Printf("assets: mkdir failed for %s: %v", dst, err)
			skipped++
			continue
		}
		if err := copyFile(src, dst); err != nil {
			log.Printf("assets: copy failed %s -> %s: %v", src, dst, err)
			skipped++
			continue
		}
		copied++
		fileCount++
		totalBytes += info.Size()
	}
	return copied, skipped, missing, totalBytes
}

// assetPlanItem describes what will be copied for an asset
type assetPlanItem struct {
	AssetRel  string
	DestRel   string
	IsDir     bool
	FileCount int
	Bytes     int64
	Missing   bool
}

// planAssetsList enumerates assets to present a copy plan before confirmation
func planAssetsList(repoRoot, profileDir string, assets []string) []assetPlanItem {
	var plans []assetPlanItem
	for _, rel := range assets {
		cleanRel, safe := cleanAssetRel(rel)
		if !safe {
			plans = append(plans, assetPlanItem{AssetRel: rel, DestRel: cleanRel, Missing: true})
			continue
		}
		src := filepath.Join(profileDir, cleanRel)
		ok, err := isPathWithinBase(repoRoot, src)
		if err != nil || !ok {
			plans = append(plans, assetPlanItem{AssetRel: rel, DestRel: cleanRel, Missing: true})
			continue
		}
		info, statErr := os.Stat(src)
		if statErr != nil {
			plans = append(plans, assetPlanItem{AssetRel: rel, DestRel: cleanRel, Missing: true})
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
				if d.IsDir() {
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
			plans = append(plans, assetPlanItem{
				AssetRel:  rel,
				DestRel:   cleanRel,
				IsDir:     true,
				FileCount: files,
				Bytes:     bytes,
				Missing:   false,
			})
		} else {
			plans = append(plans, assetPlanItem{
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
	if filepath.IsAbs(rel) {
		return "", false
	}
	clean := filepath.Clean(rel)
	clean = strings.TrimPrefix(clean, "./")
	// Reject parent traversal
	if clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return "", false
	}
	return clean, true
}

// copyDirWithLimits recursively copies files from srcDir to dstDir with size/count limits
func copyDirWithLimits(srcDir, dstDir string, fileCount *int, totalBytes *int64) (int, int, int, int64) {
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
		dst := filepath.Join(dstDir, rel)
		if info.Size() > assetWarnSizeBytes {
			fmt.Printf("  ⚠️  Large asset file: %s (%.1f MB)\n", rel, float64(info.Size())/(1024*1024))
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			skipped++
			return nil
		}
		if err := copyFile(path, dst); err != nil {
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
	if size > assetMaxFileBytes {
		return false
	}
	if fileCount+1 > assetMaxFileCount {
		return false
	}
	if totalBytes+size > assetMaxTotalBytes {
		return false
	}
	return true
}

// isPathWithinBase checks if target is within base after resolving symlinks
func isPathWithinBase(base, target string) (bool, error) {
	baseAbs, err := filepath.Abs(base)
	if err != nil {
		return false, err
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
