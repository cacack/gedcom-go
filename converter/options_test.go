package converter

import (
	"testing"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts == nil {
		t.Fatal("DefaultOptions() should not return nil")
	}

	if !opts.Validate {
		t.Error("Validate should be true by default")
	}

	if opts.StrictDataLoss {
		t.Error("StrictDataLoss should be false by default")
	}

	if !opts.PreserveUnknownTags {
		t.Error("PreserveUnknownTags should be true by default")
	}
}

func TestConvertOptions(t *testing.T) {
	t.Run("custom options", func(t *testing.T) {
		opts := &ConvertOptions{
			Validate:            false,
			StrictDataLoss:      true,
			PreserveUnknownTags: false,
		}

		if opts.Validate {
			t.Error("Validate should be false")
		}
		if !opts.StrictDataLoss {
			t.Error("StrictDataLoss should be true")
		}
		if opts.PreserveUnknownTags {
			t.Error("PreserveUnknownTags should be false")
		}
	})
}
