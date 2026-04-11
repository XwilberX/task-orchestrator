package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/XwilberX/task-orchestrator/internal/models"
	"github.com/XwilberX/task-orchestrator/internal/services"
	"github.com/XwilberX/task-orchestrator/pkg/response"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type DefinitionHandler struct {
	svc      *services.DefinitionService
	validate *validator.Validate
}

func NewDefinitionHandler(svc *services.DefinitionService) *DefinitionHandler {
	return &DefinitionHandler{svc: svc, validate: validator.New()}
}

func (h *DefinitionHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Post("/{id}/upload", h.UploadCode)
	return r
}

func (h *DefinitionHandler) Create(w http.ResponseWriter, r *http.Request) {
	d := models.DefaultDefinition()
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		response.BadRequest(w, err, "JSON inválido")
		return
	}
	if err := h.validate.Struct(d); err != nil {
		response.BadRequest(w, err, "Validación fallida")
		return
	}

	created, err := h.svc.Create(r.Context(), d)
	if err != nil {
		if errors.Is(err, services.ErrDefinitionNameTaken) {
			response.Conflict(w, err.Error())
			return
		}
		response.InternalError(w, err)
		return
	}
	response.Created(w, created)
}

func (h *DefinitionHandler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context())
	if err != nil {
		response.InternalError(w, err)
		return
	}
	response.OK(w, list, "")
}

func (h *DefinitionHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	d, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrDefinitionNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		response.InternalError(w, err)
		return
	}
	response.OK(w, d, "")
}

func (h *DefinitionHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var d models.Definition
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		response.BadRequest(w, err, "JSON inválido")
		return
	}
	if err := h.validate.Struct(d); err != nil {
		response.BadRequest(w, err, "Validación fallida")
		return
	}

	updated, err := h.svc.Update(r.Context(), id, d)
	if err != nil {
		if errors.Is(err, services.ErrDefinitionNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		if errors.Is(err, services.ErrDefinitionNameTaken) {
			response.Conflict(w, err.Error())
			return
		}
		response.InternalError(w, err)
		return
	}
	response.OK(w, updated, "")
}

// UploadCode reemplaza el código de una definición con el contenido de un archivo.
// Acepta multipart/form-data con el campo "file".
// Extensiones permitidas: .py .js .go .java
// Tamaño máximo: 5 MB (aplicado por el middleware MaxBodySize).
func (h *DefinitionHandler) UploadCode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := r.ParseMultipartForm(5 << 20); err != nil {
		response.BadRequest(w, err, "error al parsear el formulario multipart")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.BadRequest(w, err, "campo 'file' requerido")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := map[string]bool{".py": true, ".js": true, ".go": true, ".java": true}
	if !allowed[ext] {
		response.BadRequest(w, nil, "extensión no soportada (usa .py, .js, .go o .java)")
		return
	}

	code, err := io.ReadAll(file)
	if err != nil {
		response.InternalError(w, err)
		return
	}
	if len(code) == 0 {
		response.BadRequest(w, nil, "el archivo está vacío")
		return
	}

	updated, err := h.svc.UpdateCode(r.Context(), id, string(code))
	if err != nil {
		if errors.Is(err, services.ErrDefinitionNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		response.InternalError(w, err)
		return
	}
	response.OK(w, updated, "código actualizado desde archivo")
}

func (h *DefinitionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, services.ErrDefinitionNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		response.InternalError(w, err)
		return
	}
	response.OK(w, nil, "definición eliminada")
}
