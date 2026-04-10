package modul // package ini merepresentasikan domain layer (entity / business object)

import "time" // digunakan untuk representasi waktu (timestamp)

// Note adalah representasi entity utama dalam sistem.
// Struct ini kemungkinan:
// - dipakai oleh service layer
// - dipakai oleh repository (DB mapping)
// - dipakai oleh handler untuk response JSON
//
// Dalam arsitektur yang lebih besar, ini disebut domain model.
type Note struct {
	ID int `json:"id"` // Primary identifier note.
	// Biasanya auto-increment dari database.
	// Dikirim ke client sebagai "id".

	Title string `json:"title"` // Judul note.
	// Biasanya wajib diisi (validasi di service layer).

	Content string `json:"content"` // Isi dari note.
	// Bisa panjang (text/blob di database).

	Archived bool `json:"archived"` // Status soft state.
	// true  -> note sudah di-archive
	// false -> note aktif
	// Digunakan untuk filtering di endpoint List.

	CreatedAt time.Time `json:"created_at"` // Timestamp kapan note dibuat.
	// Biasanya di-set saat Create() di service layer.
	// Menggunakan time.Time agar bisa serialisasi ke JSON ISO8601.
}
