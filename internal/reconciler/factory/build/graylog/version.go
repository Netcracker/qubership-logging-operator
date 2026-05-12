package graylog

import (
	"regexp"
	"strings"
)

// versionRE matches the first dotted-semver substring in a Docker image reference.
// Pulled from the historical controller logic so callers stay byte-compatible.
var versionRE = regexp.MustCompile(`([0-9]+)\.([0-9]+)\.([0-9]+)`)

// IsV5 reports whether the given Graylog Docker image is a 5.x release. Returns false
// when the image has no parseable semver substring or the major version is not 5.
// Kept in this package so the factory and controller share one implementation.
func IsV5(dockerImage string) bool {
	match := versionRE.FindString(dockerImage)
	if match == "" {
		return false
	}
	return strings.Split(match, ".")[0] == "5"
}
