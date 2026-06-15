// Package handlers contains the annotated HTTP handlers shared by all example routers.
//
// Each function carries annotation tags in its doc-comment. Those tags are
// consumed at startup by the loader in each main.go to register routes with
// both the docwire documentation engine and the target HTTP framework.
//
// Annotation reference:
//
//	@Route    METHOD /path          – HTTP method + path (required)
//	@Summary  text                  – short operation title
//	@Desc     text                  – longer description
//	@Tags     tag1,tag2             – grouping tags
//	@Query    name "desc" required  – query-string parameter
//	@Path     name "desc"           – path parameter (always required)
//	@Body     "desc" required       – request body
//	@Response code TypeName "desc"  – possible response
//	@Auth     schemeName            – security scheme to apply
//	@Cache    duration              – suggested cache TTL (informational)
//	@For      nethttp|fiber         – target framework (default: nethttp)
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// ── net/http handlers ────────────────────────────────────────────────────────

// ListCustomers returns paginated customer records.
//
// @Route    GET /api/v1/customers
// @Summary  List Customers
// @Desc     Returns paginated customer records for the current authorization context.
// @Tags     customers
// @Query    page  "Page number (1-based)"   optional
// @Query    limit "Results per page (max 100)" optional
// @Query    status "Filter by account status"  optional
// @Response 200 CustomerResponse "Successful"
// @Response 401 ErrorResponse   "Unauthorized"
// @Auth     bearer
// @Cache    60s
// @For      nethttp
func ListCustomers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode([]CustomerResponse{
		{
			UUID:     "9f8e7d6c-5b4a-3f2e-1d0c-9b8a7f6e5d4c",
			Username: "wendel.santos",
			TierID:   42,
			Status:   "active",
			Settings: AccountSettings{TwoFactorEnabled: true, PreferredLang: "pt-BR"},
		},
	})
}

// GetCustomer returns a single customer by UUID.
//
// @Route    GET /api/v1/customers/{uuid}
// @Summary  Get Customer
// @Desc     Returns a single customer identified by UUID.
// @Tags     customers
// @Path     uuid "Customer UUID"
// @Response 200 CustomerResponse "Successful"
// @Response 404 ErrorResponse   "Customer not found"
// @Response 401 ErrorResponse   "Unauthorized"
// @Auth     bearer
// @For      nethttp
func GetCustomer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(CustomerResponse{
		UUID:     "9f8e7d6c-5b4a-3f2e-1d0c-9b8a7f6e5d4c",
		Username: "wendel.santos",
		TierID:   42,
		Status:   "active",
	})
}

// CreateCustomer registers a new customer account.
//
// @Route    POST /api/v1/customers
// @Summary  Create Customer
// @Desc     Registers a new customer account and returns the created resource.
// @Tags     customers
// @Body     "New customer data" required
// @Response 201 CustomerResponse "Customer created"
// @Response 400 ErrorResponse   "Validation error"
// @Response 401 ErrorResponse   "Unauthorized"
// @Auth     bearer
// @For      nethttp
func CreateCustomer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(CustomerResponse{
		UUID:     "new-uuid-here",
		Username: "new.user",
		TierID:   1,
		Status:   "active",
	})
}

// ReplaceCustomer performs a full replacement of a customer record.
//
// @Route    PUT /api/v1/customers/{uuid}
// @Summary  Replace Customer
// @Desc     Full replacement of a customer record by UUID.
// @Tags     customers
// @Path     uuid "Customer UUID"
// @Body     "Replacement data" required
// @Response 200 CustomerResponse "Customer updated"
// @Response 404 ErrorResponse   "Customer not found"
// @Auth     bearer
// @For      nethttp
func ReplaceCustomer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(CustomerResponse{
		UUID:   "9f8e7d6c-5b4a-3f2e-1d0c-9b8a7f6e5d4c",
		Status: "active",
	})
}

// DeleteCustomer permanently removes a customer account.
//
// @Route    DELETE /api/v1/customers/{uuid}
// @Summary  Delete Customer
// @Desc     Permanently removes a customer account identified by UUID.
// @Tags     customers
// @Path     uuid "Customer UUID"
// @Response 204 - "Deleted successfully"
// @Response 404 ErrorResponse "Customer not found"
// @Auth     bearer
// @For      nethttp
func DeleteCustomer(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

// ── Fiber handlers ───────────────────────────────────────────────────────────

// ListCustomersFiber returns paginated customer records (Fiber).
//
// @Route    GET /api/v1/customers
// @Summary  List Customers
// @Desc     Returns paginated customer records for the current authorization context.
// @Tags     customers
// @Query    page   "Page number (1-based)"      optional
// @Query    limit  "Results per page (max 100)" optional
// @Query    status "Filter by account status"   optional
// @Response 200 CustomerResponse "Successful"
// @Response 401 ErrorResponse   "Unauthorized"
// @Auth     bearer
// @Cache    60s
// @For      fiber
func ListCustomersFiber(c *fiber.Ctx) error {
	return c.JSON([]CustomerResponse{
		{
			UUID:     "7a9b2c1d-8f3e-4d5b-9a1c-3b2d1e4f5a6b",
			Username: "wendel.santos.fiber",
			TierID:   99,
			Status:   "active",
			Settings: AccountSettings{TwoFactorEnabled: true, PreferredLang: "pt-BR"},
		},
	})
}

// GetCustomerFiber returns a single customer by UUID (Fiber).
//
// @Route    GET /api/v1/customers/:uuid
// @Summary  Get Customer
// @Desc     Returns a single customer identified by UUID.
// @Tags     customers
// @Path     uuid "Customer UUID"
// @Response 200 CustomerResponse "Successful"
// @Response 404 ErrorResponse   "Customer not found"
// @Response 401 ErrorResponse   "Unauthorized"
// @Auth     bearer
// @For      fiber
func GetCustomerFiber(c *fiber.Ctx) error {
	return c.JSON(CustomerResponse{
		UUID:     c.Params("uuid"),
		Username: "wendel.santos.fiber",
		TierID:   99,
		Status:   "active",
	})
}

// CreateCustomerFiber registers a new customer account (Fiber).
//
// @Route    POST /api/v1/customers
// @Summary  Create Customer
// @Desc     Registers a new customer account and returns the created resource.
// @Tags     customers
// @Body     "New customer data" required
// @Response 201 CustomerResponse "Customer created"
// @Response 400 ErrorResponse   "Validation error"
// @Response 401 ErrorResponse   "Unauthorized"
// @Auth     bearer
// @For      fiber
func CreateCustomerFiber(c *fiber.Ctx) error {
	c.Status(http.StatusCreated)
	return c.JSON(CustomerResponse{
		UUID:     "new-uuid-here",
		Username: "new.user",
		TierID:   1,
		Status:   "active",
	})
}

// ReplaceCustomerFiber performs a full replacement of a customer record (Fiber).
//
// @Route    PUT /api/v1/customers/:uuid
// @Summary  Replace Customer
// @Desc     Full replacement of a customer record by UUID.
// @Tags     customers
// @Path     uuid "Customer UUID"
// @Body     "Replacement data" required
// @Response 200 CustomerResponse "Customer updated"
// @Response 404 ErrorResponse   "Customer not found"
// @Auth     bearer
// @For      fiber
func ReplaceCustomerFiber(c *fiber.Ctx) error {
	return c.JSON(CustomerResponse{
		UUID:   c.Params("uuid"),
		Status: "active",
	})
}

// DeleteCustomerFiber permanently removes a customer account (Fiber).
//
// @Route    DELETE /api/v1/customers/:uuid
// @Summary  Delete Customer
// @Desc     Permanently removes a customer account identified by UUID.
// @Tags     customers
// @Path     uuid "Customer UUID"
// @Response 204 - "Deleted successfully"
// @Response 404 ErrorResponse "Customer not found"
// @Auth     bearer
// @For      fiber
func DeleteCustomerFiber(c *fiber.Ctx) error {
	return c.SendStatus(http.StatusNoContent)
}
