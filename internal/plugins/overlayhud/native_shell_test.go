package overlayhud

import "testing"

func TestMomentumNativeShellSpec(t *testing.T) {
	spec, ok := MomentumNativeShellSpec("http://localhost:8080")
	if !ok {
		t.Fatal("MomentumNativeShellSpec should succeed")
	}

	if !spec.Valid() {
		t.Fatal("native shell spec should be valid")
	}
	if spec.Target.ID != "momentum-overlay" {
		t.Fatalf("Target.ID = %q, want momentum-overlay", spec.Target.ID)
	}
	if spec.Target.Route != "/internal/momentum-overlay-preview" {
		t.Fatalf("Target.Route = %q, want preview route", spec.Target.Route)
	}
	if spec.URL != "http://localhost:8080/internal/momentum-overlay-preview" {
		t.Fatalf("URL = %q, want preview URL", spec.URL)
	}
	if spec.Title != "Momentum Overlay" {
		t.Fatalf("Title = %q, want Momentum Overlay", spec.Title)
	}
}

func TestMomentumNativeShellSpecRejectsInvalidBase(t *testing.T) {
	for _, input := range []string{"", "   ", "localhost:8080", "://bad"} {
		if spec, ok := MomentumNativeShellSpec(input); ok || spec.Valid() {
			t.Fatalf("MomentumNativeShellSpec(%q) = %+v, %v, want invalid", input, spec, ok)
		}
	}
}

func TestNativeShellSpecValidRequiresTargetURLAndTitle(t *testing.T) {
	spec, ok := MomentumNativeShellSpec("http://localhost:8080")
	if !ok {
		t.Fatal("MomentumNativeShellSpec should succeed")
	}

	spec.URL = ""
	if spec.Valid() {
		t.Fatal("spec without URL should be invalid")
	}

	spec, _ = MomentumNativeShellSpec("http://localhost:8080")
	spec.Title = "   "
	if spec.Valid() {
		t.Fatal("spec without title should be invalid")
	}

	spec, _ = MomentumNativeShellSpec("http://localhost:8080")
	spec.Target.ID = "other"
	if spec.Valid() {
		t.Fatal("spec with wrong target ID should be invalid")
	}
}
