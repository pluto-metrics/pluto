package sql

import (
	"testing"
)

func TestTemplate(t *testing.T) {
	tmpl := "SELECT {{column .Col}} FROM table WHERE {{column .Col}} = {{quote .Val}}"
	args := map[string]interface{}{
		"Col": "name",
		"Val": "test",
	}
	result, err := Template(tmpl, args)
	if err != nil {
		t.Errorf("Template failed: %v", err)
	}
	expected := "SELECT `name` FROM table WHERE `name` = 'test'"
	if result != expected {
		t.Errorf("Template() = %s; want %s", result, expected)
	}
}

func TestTemplateError(t *testing.T) {
	tmpl := "{{invalid}}"
	args := map[string]interface{}{}
	_, err := Template(tmpl, args)
	if err == nil {
		t.Errorf("Template should fail with invalid template")
	}
}
