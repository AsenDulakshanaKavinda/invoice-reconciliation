package main

import (
	"encoding/json"
	"ingestion-go/config"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupRouter(cfg *config.Config) *gin.Engine {
	gin.SetMode(gin.TestMode)
	app := &Application{Config: cfg}

	r := gin.New()
	r.GET("/info", app.handleInfo)
	return r
}

func TestHandleInfo(t *testing.T) {
	mockCfg := &config.Config{
		Port:        "8080",
		Environment: "test-env",
	}

	router := setupRouter(mockCfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/info", nil)
	router.ServeHTTP(w, req)

	// Verify HTTP status code
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	// Verify JSON payload
	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal JSON response: %v", err)
	}

	if response["environment"] != "test-env" {
		t.Errorf("expected environment 'test-env', got %s", response["environment"])
	}
	if response["status"] != "running" {
		t.Errorf("expected status 'running', got %s", response["status"])
	}
}