//go:build race

package decoder

// raceDetectorEnabled reports whether this test binary was built with the
// race detector. See race_off_test.go for the counterpart.
const raceDetectorEnabled = true
