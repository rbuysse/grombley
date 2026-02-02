package main

import (
	"crypto/tls"
	"net/http"
	"testing"
)

func TestConstructFileURL(t *testing.T) {
	// Save original config and restore after tests
	originalConfig := config
	defer func() { config = originalConfig }()

	t.Run("uses request host when redirect_url is empty", func(t *testing.T) {
		config = Config{
			ServePath:   "/i/",
			RedirectURL: "",
		}

		req := &http.Request{
			Host: "example.com:3000",
			TLS:  nil,
		}

		result := constructFileURL(req, "test.jpg")
		expected := "http://example.com:3000/i/test.jpg"

		if result != expected {
			t.Errorf("Expected %s, but got %s", expected, result)
		}
	})

	t.Run("uses https when request has TLS", func(t *testing.T) {
		config = Config{
			ServePath:   "/i/",
			RedirectURL: "",
		}

		req := &http.Request{
			Host: "example.com",
			TLS:  &tls.ConnectionState{},
		}

		result := constructFileURL(req, "test.jpg")
		expected := "https://example.com/i/test.jpg"

		if result != expected {
			t.Errorf("Expected %s, but got %s", expected, result)
		}
	})

	t.Run("uses redirect_url when configured", func(t *testing.T) {
		config = Config{
			ServePath:   "/i/",
			RedirectURL: "https://cdn.example.com",
		}

		req := &http.Request{
			Host: "upload.example.com:3000",
			TLS:  nil,
		}

		result := constructFileURL(req, "test.jpg")
		expected := "https://cdn.example.com/i/test.jpg"

		if result != expected {
			t.Errorf("Expected %s, but got %s", expected, result)
		}
	})

	t.Run("redirect_url ignores request TLS state", func(t *testing.T) {
		config = Config{
			ServePath:   "/i/",
			RedirectURL: "http://cdn.example.com",
		}

		req := &http.Request{
			Host: "upload.example.com",
			TLS:  &tls.ConnectionState{}, // Has TLS but redirect_url should be used as-is
		}

		result := constructFileURL(req, "test.jpg")
		expected := "http://cdn.example.com/i/test.jpg"

		if result != expected {
			t.Errorf("Expected %s, but got %s", expected, result)
		}
	})

	t.Run("redirect_url with different serve_path", func(t *testing.T) {
		config = Config{
			ServePath:   "/media/",
			RedirectURL: "https://cdn.example.com",
		}

		req := &http.Request{
			Host: "example.com",
			TLS:  nil,
		}

		result := constructFileURL(req, "photo.png")
		expected := "https://cdn.example.com/media/photo.png"

		if result != expected {
			t.Errorf("Expected %s, but got %s", expected, result)
		}
	})

	t.Run("redirect_url with trailing slash", func(t *testing.T) {
		config = Config{
			ServePath:   "/i/",
			RedirectURL: "https://cdn.example.com/",
		}

		req := &http.Request{
			Host: "example.com",
			TLS:  nil,
		}

		result := constructFileURL(req, "image.gif")
		expected := "https://cdn.example.com//i/image.gif"

		if result != expected {
			t.Errorf("Expected %s, but got %s", expected, result)
		}
	})

	t.Run("redirect_url without scheme", func(t *testing.T) {
		config = Config{
			ServePath:   "/i/",
			RedirectURL: "cdn.example.com",
		}

		req := &http.Request{
			Host: "example.com",
			TLS:  nil,
		}

		result := constructFileURL(req, "test.jpg")
		expected := "https://cdn.example.com/i/test.jpg"

		if result != expected {
			t.Errorf("Expected %s, but got %s", expected, result)
		}
	})
}
