package handler // Package ini adalah HTTP layer (transport layer).
// Layer ini hanya menangani:
// - Parsing HTTP request
// - Validasi dasar (format, tipe data)
// - Memanggil service
// - Mengirim HTTP response
// Tidak boleh ada business logic atau query database di sini.

import (
	"encoding/json"              // Untuk encode (response) dan decode (request) JSON
	"net/http"                   // HTTP server dan interface handler bawaan Go
	"notes-api/internal/modul"   // Entity/domain model (struct Note)
	"notes-api/internal/service" // Business logic layer
	"strconv"                    // Konversi string -> int / bool
)

// NoteHandler adalah adapter antara HTTP layer dan service layer.
// Handler tidak tahu database.
// Handler tidak tahu query SQL.
// Handler hanya tahu cara berbicara HTTP.
type NoteHandler struct {
	service *service.NoteService
	// Dependency injection:
	// Handler menerima service dari luar (biasanya dari main.go).
	// Ini membuat handler tidak tightly coupled.
}

// Constructor untuk membuat instance NoteHandler.
// Pola ini disebut constructor injection.
func NewNoteHandler(s *service.NoteService) *NoteHandler {
	return &NoteHandler{service: s}
	// Mengembalikan pointer agar efisien (tidak copy struct).
}

//////////////////////////////////////////////////////////////////
// HELPER FUNCTIONS
//////////////////////////////////////////////////////////////////

// writeJSON adalah helper untuk mengirim response JSON.
// Supaya semua endpoint punya format response yang konsisten.
func writeJSON(w http.ResponseWriter, status int, data any) {

	w.Header().Set("Content-Type", "application/json")
	// Memberitahu client bahwa response berbentuk JSON.

	w.WriteHeader(status)
	// Set HTTP status code (200, 201, 400, dll).
	// Penting: WriteHeader harus dipanggil sebelum menulis body.

	json.NewEncoder(w).Encode(data)
	// Encode data menjadi JSON dan kirim ke response body.
	// Encoder langsung menulis ke ResponseWriter.
}

// writeError membungkus error agar konsisten.
// Response selalu berbentuk:
// { "error": "pesan" }
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}

//////////////////////////////////////////////////////////////////
// CREATE
//////////////////////////////////////////////////////////////////

// Endpoint: POST /notes
// Tujuan: membuat note baru.
func (h *NoteHandler) Create(w http.ResponseWriter, r *http.Request) {

	var note modul.Note
	// Siapkan struct kosong untuk menampung body JSON.

	// Decode request body ke struct.
	// r.Body adalah stream (io.Reader).
	if err := json.NewDecoder(r.Body).Decode(&note); err != nil {

		// Jika JSON tidak valid, return 400.
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Panggil service (business logic).
	// r.Context() digunakan agar:
	// - Bisa timeout
	// - Bisa cancel
	// - Bisa tracing
	if err := h.service.Create(r.Context(), &note); err != nil {

		// Error validasi (misal title kosong)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 201 Created adalah standar REST saat resource berhasil dibuat.
	writeJSON(w, http.StatusCreated, note)
}

//////////////////////////////////////////////////////////////////
// LIST
//////////////////////////////////////////////////////////////////

// Endpoint: GET /notes?archived=true
// Tujuan: mengambil list notes.
// archived adalah optional filter.
func (h *NoteHandler) List(w http.ResponseWriter, r *http.Request) {

	var archived *bool // Deklarasi pointer bool untuk filter "archived", awalnya nil (tidak filter)
	// Kenapa pointer?
	// Karena kita ingin membedakan:
	// - nil → tidak difilter
	// - false → filter archived=false
	// - true → filter archived=true

	// Ambil query parameter "archived" dari URL, misal ?archived=true
	if q := r.URL.Query().Get("archived"); q != "" {
		// Konversi string query menjadi boolean
		val, err := strconv.ParseBool(q)
		// ParseBool hanya menerima:
		// true/false/1/0

		if err != nil { // Jika query bukan boolean valid, kembalikan error 400 Bad Request
			writeError(w, http.StatusBadRequest, "invalid archived query value")
			return
		}

		archived = &val // Set pointer archived ke nilai boolean yang valid
	}

	// Panggil service layer untuk mengambil list notes, dengan optional filter archived
	notes, err := h.service.List(r.Context(), archived)
	if err != nil { // Jika gagal ambil data, kembalikan error 500 Internal Server Error
		writeError(w, http.StatusInternalServerError, "failed to fetch notes")
		return
	}

	writeJSON(w, http.StatusOK, notes) // Kirim response JSON dengan status 200 OK berisi data notes
}

//////////////////////////////////////////////////////////////////
// ARCHIVE
//////////////////////////////////////////////////////////////////

// Endpoint: PATCH /notes/{id}/archive
// Mengubah status archived menjadi true.
func (h *NoteHandler) Archive(w http.ResponseWriter, r *http.Request) {
	// Ambil parameter "id" dari URL path dan konversi ke integer
	id, err := strconv.Atoi(r.PathValue("id"))
	// r.PathValue("id") mengambil parameter dari URL.
	// Misalnya: /notes/5/archive → id = 5

	if err != nil { // Jika id bukan angka valid, kembalikan error 400 Bad Request
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	// Panggil service layer untuk menandai note sebagai archived
	if err := h.service.Archive(r.Context(), id); err != nil {
		// Jika gagal mengarsipkan, kembalikan error 500 Internal Server Error
		writeError(w, http.StatusInternalServerError, "failed to archive note")
		return
	}

	// Kirim response JSON 200 OK dengan pesan sukses
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "note archived",
	})
}

//////////////////////////////////////////////////////////////////
// UNARCHIVE
//////////////////////////////////////////////////////////////////

// Endpoint: PATCH /notes/{id}/unarchive
// Mengubah archived menjadi false.
func (h *NoteHandler) Unarchive(w http.ResponseWriter, r *http.Request) {
	// Ambil parameter "id" dari URL path dan konversi ke integer
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil { // Jika id bukan angka valid, kembalikan error 400 Bad Request
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	// Panggil service layer untuk menandai note sebagai unarchived (aktif kembali)
	if err := h.service.Unarchive(r.Context(), id); err != nil {
		// Jika gagal melakukan unarchive, kembalikan error 500 Internal Server Error
		writeError(w, http.StatusInternalServerError, "failed to unarchive note")
		return
	}

	// Kirim response JSON 200 OK dengan pesan sukses
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "note unarchived",
	})
}

//////////////////////////////////////////////////////////////////
// GET BY ID
//////////////////////////////////////////////////////////////////

// Endpoint: GET /notes/{id}
// Mengambil 1 resource berdasarkan ID.
func (h *NoteHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	// Ambil parameter "id" dari URL path dan konversi ke integer
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil { // Jika id bukan angka valid, kembalikan error 400 Bad Request
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	// Panggil service layer untuk mengambil note berdasarkan ID
	note, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		// Jika note tidak ditemukan, kembalikan error 404 Not Found
		writeError(w, http.StatusNotFound, "note not found")
		return
	}

	writeJSON(w, http.StatusOK, note) // Kirim response JSON 200 OK dengan data note
}

//////////////////////////////////////////////////////////////////
// UPDATE
//////////////////////////////////////////////////////////////////

// Endpoint: PUT /notes/{id}
// PUT artinya replace seluruh resource.
func (h *NoteHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Ambil parameter "id" dari URL path dan konversi ke integer
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil { // Jika id bukan angka valid, kembalikan error 400 Bad Request
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	// Deklarasi variabel note untuk menampung data dari request body
	var note modul.Note

	// Decode JSON request body menjadi struct note
	if err := json.NewDecoder(r.Body).Decode(&note); err != nil {
		// Jika JSON tidak valid, kembalikan error 400 Bad Request
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Panggil service layer untuk update note berdasarkan ID
	if err := h.service.Update(r.Context(), id, &note); err != nil {
		// Jika update gagal, kembalikan error 400 dengan pesan dari service
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Kirim response JSON 200 OK dengan pesan sukses
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "note updated successfully",
	})
}

//////////////////////////////////////////////////////////////////
// DELETE
//////////////////////////////////////////////////////////////////

// Endpoint: DELETE /notes/{id}
// Menghapus resource berdasarkan ID.
func (h *NoteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Ambil parameter "id" dari URL path dan konversi ke integer
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil { // Jika id bukan angka valid, kembalikan error 400 Bad Request
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	// Panggil service layer untuk menghapus note berdasarkan ID
	if err := h.service.Delete(r.Context(), id); err != nil {
		// Jika gagal menghapus, kembalikan error 500 Internal Server Error
		writeError(w, http.StatusInternalServerError, "failed to delete note")
		return
	}

	// 204 No Content → sukses tanpa body.
	w.WriteHeader(http.StatusNoContent)
}
