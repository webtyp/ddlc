package tests

import (
	"testing"

	"webtyp.com/ddlc"
	"webtyp.com/fmt"
	"webtyp.com/model"
)

type mockModel struct {
	name string
	exts []ddlc.FieldExt
}

func (m *mockModel) ModelName() string               { return m.name }
func (m *mockModel) Schema() []model.Field           { return nil }
func (m *mockModel) Pointers() []any                 { return nil }
func (m *mockModel) IsNil() bool                     { return m == nil }
func (m *mockModel) EncodeFields(model.FieldWriter)  {}
func (m *mockModel) DecodeFields(model.FieldReader)  {}
func (m *mockModel) SchemaExt() []ddlc.FieldExt      { return m.exts }

func TestTopologicalSort_NoDeps(t *testing.T) {
	users := &mockModel{name: "users"}
	roles := &mockModel{name: "roles"}
	models := []model.Model{users, roles}

	sorted, err := ddlc.TopologicalSort(models)
	if err != nil {
		t.Fatal(err)
	}
	if len(sorted) != 2 {
		t.Errorf("expected 2 models, got %d", len(sorted))
	}
}

func TestTopologicalSort_WithFK(t *testing.T) {
	users := &mockModel{name: "users"}
	sessions := &mockModel{
		name: "sessions",
		exts: []ddlc.FieldExt{
			{Ref: "users"},
		},
	}
	models := []model.Model{sessions, users}

	sorted, err := ddlc.TopologicalSort(models)
	if err != nil {
		t.Fatal(err)
	}

	uIdx, sIdx := -1, -1
	for i, m := range sorted {
		if m.ModelName() == "users" {
			uIdx = i
		}
		if m.ModelName() == "sessions" {
			sIdx = i
		}
	}

	if uIdx > sIdx {
		t.Errorf("expected users before sessions, got users at %d and sessions at %d", uIdx, sIdx)
	}
}

func TestTopologicalSort_Cycle(t *testing.T) {
	a := &mockModel{
		name: "a",
		exts: []ddlc.FieldExt{{Ref: "b"}},
	}
	b := &mockModel{
		name: "b",
		exts: []ddlc.FieldExt{{Ref: "a"}},
	}
	models := []model.Model{a, b}

	_, err := ddlc.TopologicalSort(models)
	if err == nil {
		t.Fatal("expected error on circular dependency")
	}
	if !fmt.Contains(err.Error(), "circular") {
		t.Errorf("expected circular error, got: %v", err)
	}
}

type basicModel struct {
	name string
}

func (m *basicModel) ModelName() string               { return m.name }
func (m *basicModel) Schema() []model.Field           { return nil }
func (m *basicModel) Pointers() []any                 { return nil }
func (m *basicModel) IsNil() bool                     { return m == nil }
func (m *basicModel) EncodeFields(model.FieldWriter)  {}
func (m *basicModel) DecodeFields(model.FieldReader)  {}

func TestTopologicalSort_NoSchemaExt(t *testing.T) {
	m := &basicModel{name: "m"}
	models := []model.Model{m}

	sorted, err := ddlc.TopologicalSort(models)
	if err != nil {
		t.Fatal(err)
	}
	if len(sorted) != 1 || sorted[0].ModelName() != "m" {
		t.Errorf("unexpected result: %v", sorted)
	}
}
