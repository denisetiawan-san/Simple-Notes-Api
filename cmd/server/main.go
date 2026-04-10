package main // package main adalah entry point aplikasi Go (wajib untuk executable)

import (
	"context"   // Digunakan untuk graceful shutdown (mengatur timeout, cancel, dll)
	"log"       // Logging standar Go
	"net/http"  // HTTP server bawaan Go
	"os"        // Akses environment & sistem operasi
	"os/signal" // Untuk menangkap sinyal dari OS (CTRL+C, kill, dll)
	"syscall"   // Akses sinyal level OS seperti SIGTERM
	"time"      // Untuk pengaturan timeout berbasis durasi waktu

	"github.com/joho/godotenv" // Library untuk load file .env ke environment variable

	// Internal project packages (arsitektur layered / clean architecture)
	"notes-api/internal/database"   // Layer koneksi database
	"notes-api/internal/handler"    // HTTP layer (delivery layer)
	"notes-api/internal/repository" // Data access layer
	"notes-api/internal/route"      // Routing definition
	"notes-api/internal/service"    // Business logic layer
)

func main() {

	_ = godotenv.Load()
	// Load file .env agar environment variable bisa dipakai.
	// Biasanya untuk DB_HOST, DB_PORT, DB_USER, dll.
	// Error diabaikan (_) karena kalau tidak ada .env pun app masih bisa jalan
	// (misalnya di production pakai environment variable langsung).

	// =========================
	// 1. Inisialisasi Database
	// =========================
	db, err := database.New()
	// database.New() kemungkinan:
	// - Membaca config dari environment
	// - Membuat koneksi ke database (misal PostgreSQL/MySQL)
	// - Return *sql.DB atau custom DB struct

	if err != nil {
		// Jika gagal koneksi DB → aplikasi tidak boleh lanjut
		log.Fatalf("failed to connect database: %v", err)
		// Fatalf akan:
		// 1. Print log
		// 2. Exit aplikasi
	}

	// Pastikan koneksi DB ditutup saat aplikasi berhenti
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("error closing database: %v", err)
		}
	}()
	// defer artinya fungsi ini akan dijalankan saat main() selesai.
	// Ini penting untuk mencegah memory leak atau koneksi menggantung.

	// =========================
	// 2. Dependency Injection
	// =========================
	// Flow dependency: Repository → Service → Handler

	repo := repository.NewNoteRepository(db)
	// Repository bertanggung jawab ke akses data (query ke database)
	// Dia menerima dependency DB.

	service := service.NewNoteService(repo)
	// Service adalah business logic layer.
	// Service menerima repository (tidak boleh langsung akses DB).

	handler := handler.NewNoteHandler(service)
	// Handler adalah HTTP adapter.
	// Handler menerima service (tidak boleh tahu repository atau DB).

	// Ini contoh Dependency Injection manual (tanpa framework).
	// Tujuannya:
	// - Loose coupling
	// - Mudah unit testing
	// - Clean architecture

	// =========================
	// 3. Routing
	// =========================
	mux := http.NewServeMux()
	// NewServeMux adalah HTTP router bawaan Go.
	// Dia bertugas memetakan URL → handler function.

	route.Register(mux, handler)
	// Di dalam route.Register biasanya:
	// mux.HandleFunc("/notes", handler.Create)
	// mux.HandleFunc("/notes/{id}", handler.GetByID)
	// dll
	// Jadi routing terpisah dari main (rapi & scalable).

	// =========================
	// 4. HTTP Server Config
	// =========================
	// Jangan langsung pakai http.ListenAndServe().
	// Kenapa?
	// Karena kita ingin konfigurasi timeout & graceful shutdown.

	server := &http.Server{
		Addr:         ":8080",          // Server jalan di port 8080
		Handler:      mux,              // Gunakan router mux sebagai handler utama
		ReadTimeout:  10 * time.Second, // Maksimal waktu membaca request dari client
		WriteTimeout: 10 * time.Second, // Maksimal waktu mengirim response ke client
		IdleTimeout:  60 * time.Second, // Timeout koneksi idle (keep-alive)
	}

	// Timeout ini penting untuk:
	// - Mencegah slowloris attack
	// - Mencegah koneksi menggantung terlalu lama
	// - Production best practice

	// =========================
	// 5. Jalankan Server (Goroutine)
	// =========================
	go func() {
		log.Println("server running on :8080")

		// ListenAndServe akan blocking.
		// Maka kita jalankan di goroutine agar main thread bisa lanjut
		// untuk menangani graceful shutdown.

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// http.ErrServerClosed adalah error normal saat Shutdown dipanggil.
			// Jadi hanya log fatal jika error selain itu.
			log.Fatalf("server error: %v", err)
		}
	}()

	// =========================
	// 6. Graceful Shutdown
	// =========================

	stop := make(chan os.Signal, 1)
	// Membuat channel untuk menerima sinyal OS.

	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	// os.Interrupt  → CTRL+C
	// SIGTERM       → kill command (production container biasanya pakai ini)
	// Jadi app bisa shutdown dengan benar.

	<-stop
	// Blocking sampai ada sinyal masuk.
	// Artinya aplikasi akan tetap hidup sampai user tekan CTRL+C
	// atau sistem kirim SIGTERM.

	log.Println("shutting down server...")

	// Beri waktu 5 detik untuk menyelesaikan request aktif
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// context ini akan:
	// - Memberi waktu maksimal 5 detik
	// - Setelah itu request yang belum selesai akan dibatalkan

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("server shutdown failed: %v", err)
	}
	// Shutdown akan:
	// - Stop menerima request baru
	// - Tunggu request yang sedang berjalan selesai
	// - Tutup koneksi dengan elegan (graceful)

	log.Println("server exited properly")
}
