package errs

import (
	"errors"
	"testing"
)

func TestNewErrorWithCode(t *testing.T) {
	err := NewErrorWithCode("test error", 404)
	if err == nil {
		t.Fatal("NewErrorWithCode should return an error")
	}
	var ewc ErrorWithCode
	if !errors.As(err, &ewc) {
		t.Fatal("Error should be of type ErrorWithCode")
	}
	if ewc.Error() != "test error" {
		t.Errorf("Error() = %s; want 'test error'", ewc.Error())
	}
	if ewc.Code != 404 {
		t.Errorf("Code = %d; want 404", ewc.Code)
	}
}

func TestNewErrorfWithCode(t *testing.T) {
	err := NewErrorfWithCode(500, "error %d: %s", 123, "message")
	if err == nil {
		t.Fatal("NewErrorfWithCode should return an error")
	}
	var ewc ErrorWithCode
	if !errors.As(err, &ewc) {
		t.Fatal("Error should be of type ErrorWithCode")
	}
	expectedMsg := "error 123: message"
	if ewc.Error() != expectedMsg {
		t.Errorf("Error() = %s; want '%s'", ewc.Error(), expectedMsg)
	}
	if ewc.Code != 500 {
		t.Errorf("Code = %d; want 500", ewc.Code)
	}
}

func TestErrorWithCodeError(t *testing.T) {
	ewc := ErrorWithCode{err: "custom error", Code: 400}
	if ewc.Error() != "custom error" {
		t.Errorf("Error() = %s; want 'custom error'", ewc.Error())
	}
}
