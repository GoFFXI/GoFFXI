package tools

import (
	"strconv"
	"strings"
)

func GetVersionsFromString(versions string) (int, int, int) {
	parts := strings.Split(strings.TrimSpace(versions), ".")
	if len(parts) != 3 {
		return 0, 0, 0
	}

	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])
	patch, _ := strconv.Atoi(parts[2])

	return major, minor, patch
}

func VersionMajorMatches(version, compareVersion string) bool {
	major, _, _ := GetVersionsFromString(version)
	compareMajor, _, _ := GetVersionsFromString(compareVersion)

	return major == compareMajor
}

func VersionMinorMatches(version, compareVersion string) bool {
	minor, _, _ := GetVersionsFromString(version)
	compareMinor, _, _ := GetVersionsFromString(compareVersion)

	return minor == compareMinor
}

func VersionPatchMatches(version, compareVersion string) bool {
	_, _, patch := GetVersionsFromString(version)
	_, _, comparePatch := GetVersionsFromString(compareVersion)

	return patch == comparePatch
}

func VersionIsEqualTo(version, compareVersion string) bool {
	major, minor, patch := GetVersionsFromString(version)
	compareMajor, compareMinor, comparePatch := GetVersionsFromString(compareVersion)

	return major == compareMajor && minor == compareMinor && patch == comparePatch
}

func VersionMajorAndMinorAreEqual(version, compareVersion string) bool {
	major, minor, _ := GetVersionsFromString(version)
	compareMajor, compareMinor, _ := GetVersionsFromString(compareVersion)

	return major == compareMajor && minor == compareMinor
}
