package sql

import (
	"testing"
)

func TestArrayElement(t *testing.T) {
	result := ArrayElement("arr", "1")
	expected := "arrayElement(arr, 1)"
	if result != expected {
		t.Errorf("ArrayElement(arr, 1) = %s; want %s", result, expected)
	}
}

func TestEq(t *testing.T) {
	result := Eq("a", "b")
	expected := "a = b"
	if result != expected {
		t.Errorf("Eq(a, b) = %s; want %s", result, expected)
	}
}

func TestNe(t *testing.T) {
	result := Ne("a", "b")
	expected := "a != b"
	if result != expected {
		t.Errorf("Ne(a, b) = %s; want %s", result, expected)
	}
}

func TestGt(t *testing.T) {
	result := Gt("a", "b")
	expected := "a > b"
	if result != expected {
		t.Errorf("Gt(a, b) = %s; want %s", result, expected)
	}
}

func TestGte(t *testing.T) {
	result := Gte("a", "b")
	expected := "a >= b"
	if result != expected {
		t.Errorf("Gte(a, b) = %s; want %s", result, expected)
	}
}

func TestLt(t *testing.T) {
	result := Lt("a", "b")
	expected := "a < b"
	if result != expected {
		t.Errorf("Lt(a, b) = %s; want %s", result, expected)
	}
}

func TestLte(t *testing.T) {
	result := Lte("a", "b")
	expected := "a <= b"
	if result != expected {
		t.Errorf("Lte(a, b) = %s; want %s", result, expected)
	}
}

func TestNot(t *testing.T) {
	result := Not("a")
	expected := "NOT a"
	if result != expected {
		t.Errorf("Not(a) = %s; want %s", result, expected)
	}
}

func TestMatch(t *testing.T) {
	result := Match("a", "b")
	expected := "match(a, b)"
	if result != expected {
		t.Errorf("Match(a, b) = %s; want %s", result, expected)
	}
}
