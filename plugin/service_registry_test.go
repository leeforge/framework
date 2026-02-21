package plugin

import (
	"testing"
)

type mockService struct {
	Name string
}

func TestServiceRegistry_RegisterAndResolve(t *testing.T) {
	sr := NewServiceRegistry()
	svc := &mockService{Name: "test"}

	if err := sr.Register("audit.service", svc); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	got, err := Resolve[*mockService](sr, "audit.service")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if got.Name != "test" {
		t.Errorf("got Name=%q, want %q", got.Name, "test")
	}
}

func TestServiceRegistry_DuplicateRegisterFails(t *testing.T) {
	sr := NewServiceRegistry()
	svc := &mockService{Name: "a"}

	if err := sr.Register("key", svc); err != nil {
		t.Fatalf("first Register failed: %v", err)
	}

	if err := sr.Register("key", svc); err == nil {
		t.Fatal("duplicate Register should fail")
	}
}

func TestServiceRegistry_ResolveNotFound(t *testing.T) {
	sr := NewServiceRegistry()

	_, err := Resolve[*mockService](sr, "nonexistent")
	if err == nil {
		t.Fatal("Resolve nonexistent should fail")
	}
}

func TestServiceRegistry_ResolveWrongType(t *testing.T) {
	sr := NewServiceRegistry()
	sr.Register("key", "a string, not a *mockService")

	_, err := Resolve[*mockService](sr, "key")
	if err == nil {
		t.Fatal("Resolve with wrong type should fail")
	}
}

func TestServiceRegistry_MustRegisterPanicsOnDuplicate(t *testing.T) {
	sr := NewServiceRegistry()
	sr.Register("key", "value")

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("MustRegister should panic on duplicate")
		}
	}()

	sr.MustRegister("key", "value2")
}

func TestServiceRegistry_MustResolvePanicsOnMissing(t *testing.T) {
	sr := NewServiceRegistry()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("MustResolve should panic on missing key")
		}
	}()

	MustResolve[string](sr, "nope")
}

func TestServiceRegistry_Has(t *testing.T) {
	sr := NewServiceRegistry()
	sr.Register("exists", "yes")

	if !sr.Has("exists") {
		t.Error("Has should return true for existing key")
	}
	if sr.Has("nope") {
		t.Error("Has should return false for missing key")
	}
}

func TestServiceRegistry_Keys(t *testing.T) {
	sr := NewServiceRegistry()
	sr.Register("a.svc", "1")
	sr.Register("b.svc", "2")

	keys := sr.Keys()
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
}
