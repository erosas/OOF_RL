package overlayhud

import "testing"

func TestMomentumSurfaceTargetConstants(t *testing.T) {
	target := MomentumSurfaceTarget()

	if target.ID != "momentum-overlay" {
		t.Fatalf("ID = %q, want momentum-overlay", target.ID)
	}
	if target.Name != "Momentum Overlay" {
		t.Fatalf("Name = %q, want Momentum Overlay", target.Name)
	}
	if target.Route != "/internal/momentum-overlay-preview" {
		t.Fatalf("Route = %q, want /internal/momentum-overlay-preview", target.Route)
	}
}

func TestSurfaceTargetURL(t *testing.T) {
	target := MomentumSurfaceTarget()

	got, ok := target.URL("http://localhost:8080")
	if !ok {
		t.Fatal("URL should succeed")
	}
	if got != "http://localhost:8080/internal/momentum-overlay-preview" {
		t.Fatalf("URL = %q, want preview URL", got)
	}
}

func TestSurfaceTargetURLReplacesPathAndDropsQuery(t *testing.T) {
	target := MomentumSurfaceTarget()

	got, ok := target.URL("http://localhost:8080/app?debug=1#section")
	if !ok {
		t.Fatal("URL should succeed")
	}
	if got != "http://localhost:8080/internal/momentum-overlay-preview" {
		t.Fatalf("URL = %q, want preview URL without original path/query/fragment", got)
	}
}

func TestSurfaceTargetURLRejectsInvalidBase(t *testing.T) {
	target := MomentumSurfaceTarget()

	for _, input := range []string{"", "   ", "localhost:8080", "://bad"} {
		if got, ok := target.URL(input); ok || got != "" {
			t.Fatalf("URL(%q) = %q, %v, want unavailable empty", input, got, ok)
		}
	}
}
