package department_test

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"deps-api/internal/test"
)

// helpers

func setup(t *testing.T) (*httptest.Server, func()) {
	t.Helper()
	db := test.DB(t)
	srv := test.NewServer(t, db)
	return srv, func() { test.Truncate(t, db) }
}

func createDept(t *testing.T, srv *httptest.Server, name string, parentID *uint) uint {
	t.Helper()
	body := map[string]any{"name": name}
	if parentID != nil {
		body["parent_id"] = *parentID
	}
	resp := test.Do(t, srv, http.MethodPost, "/departments", body)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var result struct {
		ID uint `json:"id"`
	}
	test.DecodeJSON(t, resp, &result)
	return result.ID
}

func createEmployee(t *testing.T, srv *httptest.Server, deptID uint, fullName, position string) uint {
	t.Helper()
	body := map[string]any{"full_name": fullName, "position": position}
	resp := test.Do(t, srv, http.MethodPost, fmt.Sprintf("/departments/%d/employees", deptID), body)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var result struct {
		ID uint `json:"id"`
	}
	test.DecodeJSON(t, resp, &result)
	return result.ID
}

// --- request validation ---

func TestRequestValidation_Departments(t *testing.T) {
	srv, clean := setup(t)
	clean()

	cases := []struct {
		name       string
		method     string
		path       string
		body       any
		wantStatus int
	}{
		{
			name:       "create with empty name",
			method:     http.MethodPost,
			path:       "/departments",
			body:       map[string]any{"name": ""},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "create with whitespace-only name",
			method:     http.MethodPost,
			path:       "/departments",
			body:       map[string]any{"name": "   "},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "create with non-existent parent",
			method:     http.MethodPost,
			path:       "/departments",
			body:       map[string]any{"name": "X", "parent_id": 9999},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "update with no fields",
			method:     http.MethodPatch,
			path:       "/departments/9999",
			body:       map[string]any{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "update with empty name",
			method:     http.MethodPatch,
			path:       "/departments/9999",
			body:       map[string]any{"name": ""},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "get with non-numeric depth",
			method:     http.MethodGet,
			path:       "/departments/9999?depth=abc",
			body:       nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "get with negative depth",
			method:     http.MethodGet,
			path:       "/departments/9999?depth=-1",
			body:       nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "delete reassign mode without target",
			method:     http.MethodDelete,
			path:       "/departments/9999?mode=reassign",
			body:       nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "delete reassign mode with invalid target",
			method:     http.MethodDelete,
			path:       "/departments/9999?mode=reassign&reassign_to_department_id=abc",
			body:       nil,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "delete reassign mode with zero target",
			method:     http.MethodDelete,
			path:       "/departments/9999?mode=reassign&reassign_to_department_id=0",
			body:       nil,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := test.Do(t, srv, tc.method, tc.path, tc.body)
			require.Equal(t, tc.wantStatus, resp.StatusCode)
		})
	}
}

func TestRequestValidation_Employees(t *testing.T) {
	srv, clean := setup(t)
	clean()
	deptID := createDept(t, srv, "Engineering", nil)

	cases := []struct {
		name       string
		body       any
		wantStatus int
	}{
		{
			name:       "missing full_name",
			body:       map[string]any{"position": "Dev"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty full_name",
			body:       map[string]any{"full_name": "", "position": "Dev"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "whitespace full_name",
			body:       map[string]any{"full_name": "   ", "position": "Dev"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing position",
			body:       map[string]any{"full_name": "Alice"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty position",
			body:       map[string]any{"full_name": "Alice", "position": ""},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "non-existent department",
			body:       map[string]any{"full_name": "Bob", "position": "Dev"},
			wantStatus: http.StatusNotFound,
		},
	}

	for i, tc := range cases {
		path := fmt.Sprintf("/departments/%d/employees", deptID)
		if i == len(cases)-1 {
			path = "/departments/9999/employees"
		}
		t.Run(tc.name, func(t *testing.T) {
			resp := test.Do(t, srv, http.MethodPost, path, tc.body)
			require.Equal(t, tc.wantStatus, resp.StatusCode)
		})
	}
}

// --- business rules ---

func TestBusinessRule_MaxDepth(t *testing.T) {
	srv, clean := setup(t)
	clean()

	// build a chain of 5 levels
	var parentID *uint
	for i := 1; i <= 5; i++ {
		id := createDept(t, srv, fmt.Sprintf("Level%d", i), parentID)
		idCopy := id
		parentID = &idCopy
	}

	// 6th level must fail with 422
	resp := test.Do(t, srv, http.MethodPost, "/departments", map[string]any{
		"name":      "Level6",
		"parent_id": *parentID,
	})
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
}

func TestBusinessRule_SiblingNameUniqueness(t *testing.T) {
	srv, clean := setup(t)
	clean()

	// two root departments with the same name → 409
	createDept(t, srv, "Engineering", nil)
	resp := test.Do(t, srv, http.MethodPost, "/departments", map[string]any{"name": "Engineering"})
	require.Equal(t, http.StatusConflict, resp.StatusCode)

	// same name is allowed under different parents
	parentA := createDept(t, srv, "ParentA", nil)
	parentB := createDept(t, srv, "ParentB", nil)
	createDept(t, srv, "Shared", &parentA)
	resp = test.Do(t, srv, http.MethodPost, "/departments", map[string]any{"name": "Shared", "parent_id": parentB})
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// duplicate among same siblings → 409
	resp = test.Do(t, srv, http.MethodPost, "/departments", map[string]any{"name": "Shared", "parent_id": parentA})
	require.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestBusinessRule_SelfParent(t *testing.T) {
	srv, clean := setup(t)
	clean()

	id := createDept(t, srv, "HR", nil)
	resp := test.Do(t, srv, http.MethodPatch, fmt.Sprintf("/departments/%d", id),
		map[string]any{"parent_id": id})
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestBusinessRule_CircularParent(t *testing.T) {
	srv, clean := setup(t)
	clean()

	// A → B → C; moving A under C must fail
	a := createDept(t, srv, "A", nil)
	b := createDept(t, srv, "B", &a)
	c := createDept(t, srv, "C", &b)

	resp := test.Do(t, srv, http.MethodPatch, fmt.Sprintf("/departments/%d", a),
		map[string]any{"parent_id": c})
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestBusinessRule_ReassignEmployeesOnDelete(t *testing.T) {
	srv, clean := setup(t)
	clean()

	src := createDept(t, srv, "Source", nil)
	dst := createDept(t, srv, "Destination", nil)
	createEmployee(t, srv, src, "Alice", "Engineer")

	// delete src with reassign to dst
	resp := test.Do(t, srv, http.MethodDelete,
		fmt.Sprintf("/departments/%d?mode=reassign&reassign_to_department_id=%d", src, dst), nil)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	// src department is gone
	resp = test.Do(t, srv, http.MethodGet, fmt.Sprintf("/departments/%d", src), nil)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	// employee now lives in dst
	resp = test.Do(t, srv, http.MethodGet, fmt.Sprintf("/departments/%d?include_employees=true", dst), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var body struct {
		Employees []struct {
			FullName string `json:"full_name"`
		} `json:"employees"`
	}
	test.DecodeJSON(t, resp, &body)
	require.Len(t, body.Employees, 1)
	require.Equal(t, "Alice", body.Employees[0].FullName)
}

func TestBusinessRule_ReassignSelfDepartment(t *testing.T) {
	srv, clean := setup(t)
	clean()

	id := createDept(t, srv, "Solo", nil)
	resp := test.Do(t, srv, http.MethodDelete,
		fmt.Sprintf("/departments/%d?mode=reassign&reassign_to_department_id=%d", id, id), nil)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestBusinessRule_ReassignNonExistentTarget(t *testing.T) {
	srv, clean := setup(t)
	clean()

	id := createDept(t, srv, "Solo", nil)
	resp := test.Do(t, srv, http.MethodDelete,
		fmt.Sprintf("/departments/%d?mode=reassign&reassign_to_department_id=9999", id), nil)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestBusinessRule_IncludeEmployees(t *testing.T) {
	srv, clean := setup(t)
	clean()

	deptID := createDept(t, srv, "Engineering", nil)
	createEmployee(t, srv, deptID, "Alice", "Engineer")
	createEmployee(t, srv, deptID, "Bob", "Manager")

	t.Run("without param employees are omitted", func(t *testing.T) {
		resp := test.Do(t, srv, http.MethodGet, fmt.Sprintf("/departments/%d", deptID), nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		var body map[string]any
		test.DecodeJSON(t, resp, &body)
		_, hasEmployees := body["employees"]
		require.False(t, hasEmployees)
	})

	t.Run("with include_employees=true employees are returned sorted", func(t *testing.T) {
		resp := test.Do(t, srv, http.MethodGet,
			fmt.Sprintf("/departments/%d?include_employees=true", deptID), nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		var body struct {
			Employees []struct {
				FullName string `json:"full_name"`
			} `json:"employees"`
		}
		test.DecodeJSON(t, resp, &body)
		require.Len(t, body.Employees, 2)
		// sorted by created_at ASC, fullname ASC — both created same instant; name sort is the tiebreak
		require.Equal(t, "Alice", body.Employees[0].FullName)
		require.Equal(t, "Bob", body.Employees[1].FullName)
	})
}

func TestBusinessRule_DepthMaintenance(t *testing.T) {
	srv, clean := setup(t)
	clean()

	root := createDept(t, srv, "Root", nil)
	child := createDept(t, srv, "Child", &root)
	grandchild := createDept(t, srv, "Grandchild", &child)

	// move child to root → grandchild depth should update
	resp := test.Do(t, srv, http.MethodPatch, fmt.Sprintf("/departments/%d", child),
		map[string]any{"move_to_root": true})
	require.Equal(t, http.StatusOK, resp.StatusCode)

	resp = test.Do(t, srv, http.MethodGet, fmt.Sprintf("/departments/%d", grandchild), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var body struct {
		Depth int `json:"depth"`
	}
	test.DecodeJSON(t, resp, &body)
	require.Equal(t, 2, body.Depth)
}

// --- log coverage ---

func TestLogCoverage(t *testing.T) {
	srv, clean := setup(t)
	clean()

	t.Run("404 on missing department logs http error", func(t *testing.T) {
		spy := test.NewLogSpy(t)
		resp := test.Do(t, srv, http.MethodGet, "/departments/9999", nil)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
		require.True(t, spy.HasLevel(slog.LevelError))
		require.True(t, spy.HasMessage("http error"))
	})

	t.Run("400 on validation failure logs http error", func(t *testing.T) {
		spy := test.NewLogSpy(t)
		resp := test.Do(t, srv, http.MethodPost, "/departments", map[string]any{"name": ""})
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.True(t, spy.HasLevel(slog.LevelError))
		require.True(t, spy.HasMessage("http error"))
	})

	t.Run("successful request produces no error logs", func(t *testing.T) {
		spy := test.NewLogSpy(t)
		resp := test.Do(t, srv, http.MethodPost, "/departments", map[string]any{"name": "Logged"})
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		require.False(t, spy.HasLevel(slog.LevelError))
	})
}