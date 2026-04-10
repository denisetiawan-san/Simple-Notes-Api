package route // Package ini bertanggung jawab atas routing.
// Routing = memetakan URL + HTTP Method → handler function.
// Layer ini tidak boleh berisi business logic atau query database.

import (
	"net/http"                   // HTTP server bawaan Go
	"notes-api/internal/handler" // Import NoteHandler (HTTP layer)
)

//////////////////////////////////////////////////////////////////
// HELPER
//////////////////////////////////////////////////////////////////

// methodNotAllowed adalah helper function untuk mengirim response 405.
// 405 = Method Not Allowed
// Digunakan ketika endpoint benar, tetapi HTTP method salah.
func methodNotAllowed(w http.ResponseWriter) {

	w.Header().Set("Content-Type", "application/json")
	// Set response sebagai JSON agar konsisten.

	w.WriteHeader(http.StatusMethodNotAllowed)
	// Status 405 → endpoint ada, tapi method tidak didukung.

	w.Write([]byte(`{"error":"method not allowed"}`))
	// Kirim response body manual dalam bentuk JSON.
	// Di sini kita tidak pakai writeJSON karena layer route
	// sebaiknya tidak tergantung helper di handler.
}

//////////////////////////////////////////////////////////////////
// REGISTER ROUTES
//////////////////////////////////////////////////////////////////

// Register bertugas mendaftarkan semua endpoint ke router.
// mux adalah *http.ServeMux (router bawaan Go).
// h adalah instance NoteHandler yang sudah dibuat di main.go.
func Register(mux *http.ServeMux, h *handler.NoteHandler) {

	//////////////////////////////////////////////////////////////////
	// ROUTE: /notes
	//////////////////////////////////////////////////////////////////
	// Endpoint ini menangani dua operasi:
	// POST /notes → Create
	// GET  /notes → List
	mux.HandleFunc("/notes", func(w http.ResponseWriter, r *http.Request) {

		// r.Method berisi HTTP method dari request.
		// Contoh: "GET", "POST", "PUT", dll.
		switch r.Method {

		case http.MethodPost:
			// Jika request adalah POST,
			// maka kita delegasikan ke handler.Create.
			h.Create(w, r)

		case http.MethodGet:
			// Jika GET,
			// maka panggil handler.List.
			h.List(w, r)

		default:
			// Jika method selain POST/GET,
			// maka kembalikan 405.
			methodNotAllowed(w)
		}
	})

	//////////////////////////////////////////////////////////////////
	// ROUTE: /notes/{id}
	//////////////////////////////////////////////////////////////////
	// Path ini menggunakan path parameter {id}.
	// Artinya:
	// /notes/5
	// /notes/10
	//
	// Endpoint ini menangani:
	// GET    → ambil 1 note
	// PUT    → update note
	// DELETE → hapus note
	mux.HandleFunc("/notes/{id}", func(w http.ResponseWriter, r *http.Request) {

		switch r.Method {

		case http.MethodGet:
			// GET /notes/{id}
			h.GetByID(w, r)

		case http.MethodPut:
			// PUT /notes/{id}
			// PUT berarti replace seluruh resource.
			h.Update(w, r)

		case http.MethodDelete:
			// DELETE /notes/{id}
			h.Delete(w, r)

		default:
			methodNotAllowed(w)
		}
	})

	//////////////////////////////////////////////////////////////////
	// ROUTE: /notes/{id}/archive
	//////////////////////////////////////////////////////////////////
	// Endpoint khusus untuk mengubah field archived menjadi true.
	// Menggunakan PATCH karena hanya mengubah sebagian field.
	//
	// PATCH /notes/{id}/archive
	mux.HandleFunc("/notes/{id}/archive", func(w http.ResponseWriter, r *http.Request) {

		// Di sini kita tidak pakai switch,
		// karena hanya menerima 1 method: PATCH.
		if r.Method != http.MethodPatch {

			methodNotAllowed(w)
			return
			// return penting agar tidak lanjut eksekusi.
		}

		h.Archive(w, r)
	})

	//////////////////////////////////////////////////////////////////
	// ROUTE: /notes/{id}/unarchive
	//////////////////////////////////////////////////////////////////
	// Endpoint untuk membatalkan archive.
	// PATCH karena hanya mengubah satu field.
	//
	// PATCH /notes/{id}/unarchive
	mux.HandleFunc("/notes/{id}/unarchive", func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPatch {

			methodNotAllowed(w)
			return
		}

		h.Unarchive(w, r)
	})
}
