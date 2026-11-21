//go:build race

package parser_test

// raceDetectorEnabled reports whether this test binary was built with the
// race detector. See race_off_test.go for the counterpart.
//
// The constant is duplicated from packages decoder and validator rather than
// shared: it is test-only in each, and exporting it would put a build-tag
// detail into a public API.
const raceDetectorEnabled = true
