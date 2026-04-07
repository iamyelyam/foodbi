package files

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/foodbi/backend/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	db        *pgxpool.Pool
	uploadDir string
}

func NewHandler(db *pgxpool.Pool) *Handler {
	dir := os.Getenv("UPLOAD_DIR")
	if dir == "" {
		dir = "./uploads"
	}
	os.MkdirAll(dir, 0755)
	return &Handler{db: db, uploadDir: dir}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/upload", h.Upload)
	r.Get("/", h.List)
	r.Get("/{id}", h.Get)
	return r
}

type FileRecord struct {
	ID        string `json:"id"`
	Filename  string `json:"filename"`
	MimeType  string `json:"mime_type"`
	Size      int64  `json:"size"`
	Status    string `json:"status"` // uploaded, processing, processed
	CreatedAt string `json:"created_at"`
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	userID := middleware.GetUserID(r.Context())

	r.ParseMultipartForm(10 << 20) // 10MB max

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	id := uuid.New()
	ext := filepath.Ext(header.Filename)
	storedName := id.String() + ext
	destPath := filepath.Join(h.uploadDir, storedName)

	dest, err := os.Create(destPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save file")
		return
	}
	defer dest.Close()

	size, err := io.Copy(dest, file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to write file")
		return
	}

	mimeType := header.Header.Get("Content-Type")
	_, err = h.db.Exec(r.Context(),
		`INSERT INTO uploaded_files (id, company_id, uploaded_by, filename, stored_name, mime_type, size_bytes, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, 'uploaded', NOW())`,
		id, companyID, userID, header.Filename, storedName, mimeType, size)
	if err != nil {
		os.Remove(destPath)
		writeError(w, http.StatusInternalServerError, "failed to record file")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":       id,
		"filename": header.Filename,
		"size":     size,
		"status":   "uploaded",
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())

	rows, err := h.db.Query(r.Context(),
		`SELECT id, filename, mime_type, size_bytes, status, created_at
		 FROM uploaded_files WHERE company_id = $1 ORDER BY created_at DESC LIMIT 50`, companyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch files")
		return
	}
	defer rows.Close()

	var files []FileRecord
	for rows.Next() {
		var f FileRecord
		var t time.Time
		if err := rows.Scan(&f.ID, &f.Filename, &f.MimeType, &f.Size, &f.Status, &t); err != nil {
			continue
		}
		f.CreatedAt = t.Format(time.RFC3339)
		files = append(files, f)
	}
	if files == nil {
		files = []FileRecord{}
	}
	writeJSON(w, http.StatusOK, files)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r.Context())
	id := chi.URLParam(r, "id")

	var f FileRecord
	var t time.Time
	err := h.db.QueryRow(r.Context(),
		`SELECT id, filename, mime_type, size_bytes, status, created_at
		 FROM uploaded_files WHERE id = $1 AND company_id = $2`, id, companyID).
		Scan(&f.ID, &f.Filename, &f.MimeType, &f.Size, &f.Status, &t)
	if err != nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}
	f.CreatedAt = t.Format(time.RFC3339)
	writeJSON(w, http.StatusOK, f)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
