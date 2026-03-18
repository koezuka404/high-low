package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestRespondSuccessAndError(t *testing.T) {
	e := echo.New()

	{
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if err := respondSuccess(c, http.StatusOK, map[string]any{"x": 1}); err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var body SuccessResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if !body.Success {
			t.Fatal("expected success=true")
		}
	}

	{
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if err := respondError(c, http.StatusBadRequest, "invalid_input", "bad"); err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
		var body ErrorResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Success {
			t.Fatal("expected success=false")
		}
		if body.Error.Code != "invalid_input" || body.Error.Message != "bad" {
			t.Fatalf("unexpected error body: %+v", body.Error)
		}
	}
}

