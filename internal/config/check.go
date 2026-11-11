package config

import (
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/lucky7xz/drako/internal/profile"
)

// CheckSettings lints raw config.toml: what sends the loader to rescue mode, and
// keys it ignores (a typo, or a setting drako no longer has).
func CheckSettings(src []byte) []profile.Finding {
	var settings AppSettings
	meta, err := toml.Decode(string(src), &settings)
	if err != nil {
		return []profile.Finding{{Level: "error", Msg: fmt.Sprintf("%v", err)}}
	}
	if keys := undecodedKeys(meta); len(keys) > 0 {
		return []profile.Finding{{Level: "warning", Msg: fmt.Sprintf("unrecognized key(s): %s (a typo, or a setting drako no longer has)", strings.Join(keys, ", "))}}
	}
	return nil
}
