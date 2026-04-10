package repository // Layer ini adalah persistence layer (akses database).
// Repository bertanggung jawab:
// - Menjalankan query SQL
// - Mapping hasil query ke struct
// Repository TIDAK boleh:
// - Validasi business rule
// - Parsing HTTP
// - Mengandung logic aplikasi

import (
	"context"                  // Membawa lifecycle request (timeout, cancel) sampai ke database
	"database/sql"             // Standard library Go untuk SQL database
	"errors"                   // Untuk membuat dan membandingkan error
	"notes-api/internal/modul" // Entity/domain model Note
)

//////////////////////////////////////////////////////////////////
// STRUCT & CONSTRUCTOR
//////////////////////////////////////////////////////////////////

// NoteRepository adalah struct yang membungkus koneksi database.
// Kita tidak expose *sql.DB langsung ke layer lain.
type NoteRepository struct {
	db *sql.DB
	// *sql.DB adalah connection pool.
	// Artinya:
	// - Bukan 1 koneksi tunggal
	// - Thread-safe
	// - Mengatur koneksi secara otomatis
}

// Constructor untuk repository.
// Dependency Injection: kita inject *sql.DB dari luar.
func NewNoteRepository(db *sql.DB) *NoteRepository {
	return &NoteRepository{db: db}
}

//////////////////////////////////////////////////////////////////
// CREATE
//////////////////////////////////////////////////////////////////

// Create menyimpan note baru ke database.
func (r *NoteRepository) Create(ctx context.Context, note *modul.Note) error {
	// Query SQL untuk insert note baru, archived default false, created_at current timestamp
	query := `
		INSERT INTO notes (title, content, archived, created_at)
		VALUES (?, ?, false, NOW())
	`
	// Placeholder "?" mencegah SQL injection.
	// archived default false.
	// NOW() mengambil waktu dari database server (lebih konsisten daripada time.Now()).

	// Eksekusi query dengan judul dan konten note, gunakan context untuk timeout/cancel
	result, err := r.db.ExecContext(ctx, query, note.Title, note.Content)
	// ExecContext dipakai untuk query yang tidak mengembalikan rows.
	// ctx memungkinkan:
	// - Query dibatalkan jika request timeout
	// - Graceful shutdown

	if err != nil { // Jika gagal insert, kembalikan error
		return err
	}

	id, err := result.LastInsertId() // Ambil ID yang di-generate database untuk note baru
	// Mengambil ID auto increment yang dibuat oleh database.
	if err != nil {
		return err
	}

	note.ID = int(id) // Set ID note pada struct agar bisa dikembalikan ke service/handler
	// Set ID ke struct supaya layer atas tahu ID resource baru.

	return nil // Berhasil, return nil
}

//////////////////////////////////////////////////////////////////
// SET ARCHIVED
//////////////////////////////////////////////////////////////////

// SetArchived mengubah status archived menjadi true/false.
func (r *NoteRepository) SetArchived(ctx context.Context, id int, archived bool) error {
	// Eksekusi query UPDATE untuk set kolom archived berdasarkan ID
	result, err := r.db.ExecContext(
		ctx,
		"UPDATE notes SET archived = ? WHERE id = ?",
		archived, // true → archive, false → unarchive
		id,
	)
	// UPDATE berdasarkan primary key.

	if err != nil { // Jika gagal query, return error
		return err
	}

	// Cek berapa baris yang terpengaruh oleh UPDATE
	rowsAffected, err := result.RowsAffected()
	// Mengecek berapa baris yang terpengaruh.
	if err != nil {
		return err
	}

	// Jika tidak ada baris terpengaruh → ID tidak ditemukan
	if rowsAffected == 0 {
		// Artinya ID tidak ditemukan.
		return errors.New("note not found")
	}

	return nil // Sukses update
}

//////////////////////////////////////////////////////////////////
// GET ALL
//////////////////////////////////////////////////////////////////

// GetAll mengambil banyak data note.
// archived pointer digunakan untuk optional filter.
func (r *NoteRepository) GetAll(ctx context.Context, archived *bool) ([]modul.Note, error) {
	// Query dasar untuk ambil semua kolom note
	query := `
		SELECT id, title, content, archived, created_at
		FROM notes
	`
	// Slice untuk menampung parameter query
	args := []interface{}{}
	// Slice untuk menyimpan parameter dinamis.

	// Tambahkan filter archived jika diberikan
	if archived != nil {
		query += " WHERE archived = ?"
		args = append(args, *archived)
		// Jika user mengirim filter.
	} else {
		query += " WHERE archived = false"
		// Default behavior: hanya tampilkan yang belum di-archive.
	}

	// Eksekusi query dengan context dan parameter args
	rows, err := r.db.QueryContext(ctx, query, args...)
	// QueryContext digunakan untuk SELECT banyak baris.
	if err != nil {
		return nil, err
	}
	defer rows.Close() // pastikan rows ditutup setelah selesai
	// WAJIB ditutup.
	// Jika tidak → koneksi pool bisa habis (connection leak).

	var notes []modul.Note

	// Iterasi hasil query dan scan ke struct Note
	for rows.Next() {
		var n modul.Note

		if err := rows.Scan(
			&n.ID,
			&n.Title,
			&n.Content,
			&n.Archived,
			&n.CreatedAt,
		); err != nil {
			return nil, err
		}
		// Scan memetakan kolom → field struct.
		// Urutan harus sama dengan SELECT.

		notes = append(notes, n)
	}

	if err := rows.Err(); err != nil {
		// Mengecek error selama iterasi.
		return nil, err
	}

	return notes, nil // Return slice notes yang sudah di-scan
}

//////////////////////////////////////////////////////////////////
// GET BY ID
//////////////////////////////////////////////////////////////////

// GetByID mengambil satu note berdasarkan ID.
func (r *NoteRepository) GetByID(ctx context.Context, id int) (*modul.Note, error) {
	// Query SQL untuk mengambil note berdasarkan ID
	query := `
		SELECT id, title, content, archived, created_at
		FROM notes
		WHERE id = ?
	`

	var n modul.Note // Variabel struct Note untuk menampung hasil query

	// Eksekusi query dan scan hasil ke struct Note
	err := r.db.QueryRowContext(ctx, query, id).
		Scan(
			&n.ID,
			&n.Title,
			&n.Content,
			&n.Archived,
			&n.CreatedAt,
		)
	// QueryRowContext dipakai jika hanya 1 row yang diharapkan.

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Jika tidak ada baris yang ditemukan → kembalikan error "note not found"
			return nil, errors.New("note not found")
		}
		// Error lain dari query → return error
		return nil, err
	}

	return &n, nil // Berhasil → kembalikan pointer ke note
}

//////////////////////////////////////////////////////////////////
// UPDATE
//////////////////////////////////////////////////////////////////

// Update mengubah title dan content berdasarkan ID.
func (r *NoteRepository) Update(ctx context.Context, id int, note *modul.Note) error {
	// Eksekusi query UPDATE untuk ubah title dan content berdasarkan ID
	result, err := r.db.ExecContext(
		ctx,
		"UPDATE notes SET title = ?, content = ? WHERE id = ?",
		note.Title,
		note.Content,
		id,
	)
	if err != nil { // Jika gagal query → return error
		return err
	}

	// Cek berapa baris yang terpengaruh oleh UPDATE
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 { // Jika tidak ada baris terpengaruh → ID tidak ditemukan
		return errors.New("note not found")
	}

	return nil // Sukses update
}

//////////////////////////////////////////////////////////////////
// DELETE
//////////////////////////////////////////////////////////////////

// Delete menghapus data note secara permanen (hard delete).
func (r *NoteRepository) Delete(ctx context.Context, id int) error {
	// Eksekusi query DELETE berdasarkan ID
	result, err := r.db.ExecContext(
		ctx,
		"DELETE FROM notes WHERE id = ?",
		id,
	)
	if err != nil { // Jika query gagal → return error
		return err
	}

	// Cek berapa baris yang terpengaruh oleh DELETE
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 { // Jika tidak ada baris terhapus → ID tidak ditemukan
		return errors.New("note not found")
	}

	return nil // Sukses delete
}
