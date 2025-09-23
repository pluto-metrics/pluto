package sql

import (
	"testing"
)

func TestQuoteInt(t *testing.T) {
	result := Quote(42)
	expected := "42"
	if result != expected {
		t.Errorf("Quote(42) = %s; want %s", result, expected)
	}
}

func TestQuoteString(t *testing.T) {
	result := Quote("hello")
	expected := "'hello'"
	if result != expected {
		t.Errorf("Quote('hello') = %s; want %s", result, expected)
	}
}

func TestQuoteStringWithSpecialChars(t *testing.T) {
	result := Quote("hel'lo\\world")
	expected := "'hel\\'lo\\\\world'"
	if result != expected {
		t.Errorf("Quote('hel\\'lo\\\\world') = %s; want %s", result, expected)
	}
}

func TestQuoteBytes(t *testing.T) {
	result := Quote([]byte("hello"))
	expected := "'hello'"
	if result != expected {
		t.Errorf("Quote([]byte('hello')) = %s; want %s", result, expected)
	}
}

func TestQuotePanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Quote should panic for unsupported type")
		}
	}()
	Quote(3.14)
}

func TestColumn(t *testing.T) {
	result := Column("col")
	expected := "`col`"
	if result != expected {
		t.Errorf("Column('col') = %s; want %s", result, expected)
	}
}

func TestColumnWithSpecialChars(t *testing.T) {
	result := Column("col`\\")
	expected := "`col\\`\\\\`"
	if result != expected {
		t.Errorf("Column('col\\`\\\\') = %s; want %s", result, expected)
	}
}
