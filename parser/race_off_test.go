//go:build !race

package parser_test

// raceDetectorEnabled reports whether this test binary was built with the
// race detector. See race_on_test.go for the counterpart.
const raceDetectorEnabled = false
