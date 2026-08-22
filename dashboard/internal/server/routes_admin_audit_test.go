package server

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestAuditRoutesRequireAdministrator(t *testing.T) {
	app := fiber.New()
	routes := &adminRoutes{}
	app.Post("/api/v1/admin/audit/chat/search", routes.requireAdmin, routes.searchChatAudit)
	request := httptest.NewRequest("POST", "/api/v1/admin/audit/chat/search", nil)
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 401 {
		t.Fatalf("status=%d want=401", response.StatusCode)
	}
}

func TestCSVFormulaInjectionMitigation(t *testing.T) {
	for _, value := range []string{"=cmd", "+1", "-1", "@sum"} {
		if got := csvSafe(value); got != "'"+value {
			t.Fatalf("csvSafe(%q)=%q", value, got)
		}
	}
	if got := csvSafe("hello"); got != "hello" {
		t.Fatalf("ordinary value changed: %q", got)
	}
}
