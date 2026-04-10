package service // Layer service bertanggung jawab atas business logic (aturan bisnis aplikasi)

import (
	"context"                       // Digunakan untuk membawa context dari HTTP layer ke repository (mendukung timeout, cancel, tracing, dsb)
	"errors"                        // Untuk membuat error secara manual
	"notes-api/internal/modul"      // Berisi domain entity Note (struktur data utama aplikasi)
	"notes-api/internal/repository" // Layer repository untuk akses database
	"strings"                       // Untuk manipulasi string (misalnya trim whitespace)
)

// NoteService adalah struct yang merepresentasikan layer business logic.
// Dia tidak berinteraksi langsung dengan HTTP (itu tugas handler).
// Dia juga tidak tahu detail database (itu tugas repository).
// Jadi posisinya adalah middle layer: handler -> service -> repository.
type NoteService struct {
	repo *repository.NoteRepository // Dependency ke repository (Dependency Injection)
}

// NewNoteService adalah constructor.
// Tujuannya untuk membuat instance NoteService dengan repository yang sudah di-inject.
// Pola ini disebut Dependency Injection supaya service tidak membuat repository sendiri.
func NewNoteService(repo *repository.NoteRepository) *NoteService {
	return &NoteService{repo: repo} // Mengembalikan pointer ke NoteService
}

// ======================= CREATE =======================
// Method ini dipanggil saat endpoint POST /notes dipanggil.
func (s *NoteService) Create(ctx context.Context, note *modul.Note) error {

	// Sanitasi input:
	// Menghapus spasi di awal dan akhir string.
	// Ini mencegah user mengirim "   belajar   " dan dianggap valid padahal kosong.
	note.Title = strings.TrimSpace(note.Title)
	note.Content = strings.TrimSpace(note.Content)

	// Business rule:
	// Title wajib diisi.
	// Kalau kosong setelah di-trim, berarti invalid.
	if note.Title == "" {
		return errors.New("title is required") // Kembalikan error ke handler
	}

	// Jika validasi lolos,
	// delegasikan ke repository untuk disimpan ke database.
	return s.repo.Create(ctx, note)
}

// ======================= ARCHIVE =======================
// Dipanggil saat PATCH /notes/{id}/archive
func (s *NoteService) Archive(ctx context.Context, id int) error {

	// Validasi dasar:
	// ID tidak boleh <= 0 (tidak masuk akal dalam database)
	if id <= 0 {
		return errors.New("invalid id")
	}

	// Panggil repository untuk update field archived = true
	// Kita tidak update seluruh data, hanya field archived.
	return s.repo.SetArchived(ctx, id, true)
}

// ======================= UNARCHIVE =======================
// Dipanggil saat PATCH /notes/{id}/unarchive
func (s *NoteService) Unarchive(ctx context.Context, id int) error {

	// Validasi ID
	if id <= 0 {
		return errors.New("invalid id")
	}

	// Set archived menjadi false
	return s.repo.SetArchived(ctx, id, false)
}

// ======================= LIST =======================
// Dipanggil saat GET /notes
func (s *NoteService) List(ctx context.Context, archived *bool) ([]modul.Note, error) {

	// archived adalah pointer ke bool.
	// Kenapa pointer?
	// Karena nil berarti "tidak difilter"
	// true berarti ambil yang archived
	// false berarti ambil yang tidak archived

	// Tidak ada business rule khusus di sini,
	// jadi langsung delegasikan ke repository.
	return s.repo.GetAll(ctx, archived)
}

// ======================= GET BY ID =======================
// Dipanggil saat GET /notes/{id}
func (s *NoteService) GetByID(ctx context.Context, id int) (*modul.Note, error) {

	// Validasi ID
	if id <= 0 {
		return nil, errors.New("invalid id")
	}

	// Ambil data dari repository.
	// Return berupa pointer karena bisa saja nil jika tidak ditemukan.
	return s.repo.GetByID(ctx, id)
}

// ======================= UPDATE =======================
// Dipanggil saat PUT /notes/{id}
func (s *NoteService) Update(ctx context.Context, id int, note *modul.Note) error {

	// Validasi ID
	if id <= 0 {
		return errors.New("invalid id")
	}

	// Sanitasi input lagi (best practice)
	note.Title = strings.TrimSpace(note.Title)
	note.Content = strings.TrimSpace(note.Content)

	// Business rule tetap sama:
	// Title tidak boleh kosong.
	if note.Title == "" {
		return errors.New("title is required")
	}

	// Delegasikan update ke repository.
	return s.repo.Update(ctx, id, note)
}

// ======================= DELETE =======================
// Dipanggil saat DELETE /notes/{id}
func (s *NoteService) Delete(ctx context.Context, id int) error {

	// Validasi ID
	if id <= 0 {
		return errors.New("invalid id")
	}

	// Delegasikan proses delete ke repository.
	return s.repo.Delete(ctx, id)
}
