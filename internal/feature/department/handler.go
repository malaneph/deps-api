package department

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"deps-api/internal/api"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func Register(mux *http.ServeMux, db *gorm.DB) {
	h := NewHandler(NewService(db))
	mux.HandleFunc("GET /departments", h.List)
	mux.HandleFunc("GET /departments/{id}", h.GetByID)
	mux.HandleFunc("POST /departments", h.Create)
	mux.HandleFunc("PATCH /departments/{id}", h.Update)
	mux.HandleFunc("DELETE /departments/{id}", h.Delete)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	departments, err := h.svc.List()
	if err != nil {
		api.HandleError(w, r, err)
		return
	}
	api.JSON(w, http.StatusOK, departments)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		api.HandleError(w, r, err)
		return
	}

	depth := 0
	if raw := r.URL.Query().Get("depth"); raw != "" {
		d, parseErr := strconv.Atoi(raw)
		if parseErr != nil || d < 0 {
			api.HandleError(w, r, api.ErrBadRequest("depth must be a non-negative integer"))
			return
		}
		depth = d
	}

	dept, err := h.svc.GetByID(id, depth)
	if err != nil {
		api.HandleError(w, r, mapErr(err))
		return
	}
	api.JSON(w, http.StatusOK, dept)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		ParentID *uint  `json:"parent_id"`
	}
	if err := api.Decode(w, r, &req); err != nil {
		api.HandleError(w, r, err)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		api.HandleError(w, r, api.ErrBadRequest("name is required"))
		return
	}

	dept, err := h.svc.Create(CreateInput{
		Name:     req.Name,
		ParentID: req.ParentID,
	})
	if err != nil {
		api.HandleError(w, r, mapErr(err))
		return
	}
	api.JSON(w, http.StatusCreated, dept)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		api.HandleError(w, r, err)
		return
	}

	var req struct {
		Name       *string `json:"name"`
		ParentID   *uint   `json:"parent_id"`
		MoveToRoot bool    `json:"move_to_root"`
	}
	if err := api.Decode(w, r, &req); err != nil {
		api.HandleError(w, r, err)
		return
	}
	if req.Name == nil && req.ParentID == nil && !req.MoveToRoot {
		api.HandleError(w, r, api.ErrBadRequest("at least one field must be provided"))
		return
	}
	if req.Name != nil {
		*req.Name = strings.TrimSpace(*req.Name)
		if *req.Name == "" {
			api.HandleError(w, r, api.ErrBadRequest("name must not be empty"))
			return
		}
	}

	dept, err := h.svc.Update(id, UpdateInput{
		Name:       req.Name,
		ParentID:   req.ParentID,
		MoveToRoot: req.MoveToRoot,
	})
	if err != nil {
		api.HandleError(w, r, mapErr(err))
		return
	}
	api.JSON(w, http.StatusOK, dept)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		api.HandleError(w, r, err)
		return
	}

	if err := h.svc.Delete(id); err != nil {
		api.HandleError(w, r, mapErr(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func mapErr(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return api.ErrNotFound("department not found")
	case errors.Is(err, ErrMaxDepth):
		return api.ErrUnprocessable("max department depth of 5 exceeded")
	case errors.Is(err, ErrDuplicateName):
		return api.ErrConflict("department name already exists in this parent")
	case errors.Is(err, ErrSelfParent):
		return api.ErrBadRequest("department cannot be its own parent")
	default:
		return err
	}
}

func pathID(r *http.Request) (uint, error) {
	raw := r.PathValue("id")
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return 0, api.ErrBadRequest("invalid id")
	}
	return uint(id), nil
}
