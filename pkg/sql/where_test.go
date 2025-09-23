package sql

import (
	"testing"
)

func TestNewWhere(t *testing.T) {
	w := NewWhere()
	if w == nil {
		t.Errorf("NewWhere() should not return nil")
	}
	if w.String() != "" {
		t.Errorf("NewWhere().String() = %s; want ''", w.String())
	}
}

func TestWhereAnd(t *testing.T) {
	w := NewWhere()
	w.And("a = 1")
	if w.String() != "a = 1" {
		t.Errorf("After And('a = 1'), String() = %s; want 'a = 1'", w.String())
	}
	w.And("b = 2")
	expected := "(a = 1) AND (b = 2)"
	if w.String() != expected {
		t.Errorf("After second And, String() = %s; want %s", w.String(), expected)
	}
}

func TestWhereAndEmpty(t *testing.T) {
	w := NewWhere()
	w.And("")
	w.And("a = 1")
	if w.String() != "a = 1" {
		t.Errorf("And with empty string should not change, String() = %s; want 'a = 1'", w.String())
	}
}

func TestWhereOr(t *testing.T) {
	w := NewWhere()
	w.Or("a = 1")
	if w.String() != "a = 1" {
		t.Errorf("After Or('a = 1'), String() = %s; want 'a = 1'", w.String())
	}
	w.Or("b = 2")
	expected := "(a = 1) OR (b = 2)"
	if w.String() != expected {
		t.Errorf("After second Or, String() = %s; want %s", w.String(), expected)
	}
}

func TestWhereOrEmpty(t *testing.T) {
	w := NewWhere()
	w.Or("")
	w.Or("a = 1")
	if w.String() != "a = 1" {
		t.Errorf("Or with empty string should not change, String() = %s; want 'a = 1'", w.String())
	}
}

func TestWhereAndf(t *testing.T) {
	w := NewWhere()
	w.Andf("%s = %d", "a", 1)
	expected := "a = 1"
	if w.String() != expected {
		t.Errorf("Andf('%%s = %%d', 'a', 1) = %s; want %s", w.String(), expected)
	}
}

func TestWhereSQL(t *testing.T) {
	w := NewWhere()
	if w.SQL() != "" {
		t.Errorf("Empty Where SQL() = %s; want ''", w.SQL())
	}
	w.And("a = 1")
	expected := "WHERE a = 1"
	if w.SQL() != expected {
		t.Errorf("SQL() = %s; want %s", w.SQL(), expected)
	}
}

func TestWherePreWhereSQL(t *testing.T) {
	w := NewWhere()
	if w.PreWhereSQL() != "" {
		t.Errorf("Empty Where PreWhereSQL() = %s; want ''", w.PreWhereSQL())
	}
	w.And("a = 1")
	expected := "PREWHERE a = 1"
	if w.PreWhereSQL() != expected {
		t.Errorf("PreWhereSQL() = %s; want %s", w.PreWhereSQL(), expected)
	}
}
