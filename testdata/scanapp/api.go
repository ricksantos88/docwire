// Package scanapp is a fixture exercising docwire.Scan end to end: a package
// doc comment carrying the global metadata plus annotated handlers and the DTOs
// they reference, all in source form.
//
// @Title       Scan Fixture API
// @Version     v2.3.0
// @Description Fixture used by docwire.Scan tests.
// @Server      http://localhost:9090  "Local"
// @Security    bearer  bearerJWT
package scanapp

import "net/http"

// RegisterRequest is the request DTO inferred from Register's body.
type RegisterRequest struct {
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required" format:"email"`
	Age      int32  `json:"age"`
	Password string `json:"-"`
}

// UserResponse is the 201 response payload.
type UserResponse struct {
	ID       string   `json:"id" example:"u_123"`
	Name     string   `json:"name"`
	Roles    []string `json:"roles"`
	internal string
}

// ErrorResponse is the shared error envelope.
type ErrorResponse struct {
	Code    string `json:"code" example:"VALIDATION_ERROR"`
	Message string `json:"message"`
}

// PaginatedUsers nests a slice of UserResponse to test recursive registration.
type PaginatedUsers struct {
	Data  []UserResponse `json:"data"`
	Total int            `json:"total"`
}

// Register creates a public account.
//
// @Route    POST /v1/register
// @Summary  Register a manager
// @Tags     auth
// @Body     "New manager data" required
// @Response 201 UserResponse  "Created"
// @Response 422 ErrorResponse "Validation"
func Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	_ = req
}

// List returns users.
//
// @Route    GET /v1/users
// @Summary  List users
// @Tags     users
// @Query    limit "Page size" optional
// @Response 200 PaginatedUsers "Page"
// @Response 401 ErrorResponse  "Unauthorized"
// @Auth     bearer
func List(w http.ResponseWriter, r *http.Request) {}
