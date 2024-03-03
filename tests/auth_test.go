package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegisterUser(t *testing.T) {
	// Create a request body with required form values
	requestBody := strings.NewReader("email=test@example.com&username=testuser&password=testpassword")

	req, err := http.NewRequest("POST", "/register", requestBody)
	if err != nil {
		t.Fatal(err)
	}

	// Create a recorder to capture the response
	rr := httptest.NewRecorder()

	// Call the handler function with the request
	handler := http.HandlerFunc(registerUser)
	handler.ServeHTTP(rr, req)

	// Check the response status code
	if status := rr.Code; status != http.StatusSeeOther {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusSeeOther)
	}
}

func TestLoginUser(t *testing.T) {
	// Create a request body with required form values
	requestBody := strings.NewReader("username=testuser&password=testpassword")

	req, err := http.NewRequest("POST", "/login", requestBody)
	if err != nil {
		t.Fatal(err)
	}

	// Create a recorder to capture the response
	rr := httptest.NewRecorder()

	// Call the handler function with the request
	handler := http.HandlerFunc(loginUser)
	handler.ServeHTTP(rr, req)

	// Check the response status code
	if status := rr.Code; status != http.StatusSeeOther {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusSeeOther)
	}
}

// Note: These tests assume that the registerUser and loginUser functions
// perform the necessary redirection upon successful registration/login.
