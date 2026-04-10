package database // ini nama package. Biasanya berisi logic untuk koneksi database

import (
	"context"      // untuk mengatur batas waktu (timeout)
	"database/sql" // package bawaan Go untuk koneksi dan query database
	"errors"       // untuk membuat error manual
	"os"           // untuk membaca environment variable
	"time"         // untuk mengatur durasi waktu

	_ "github.com/go-sql-driver/mysql"
	// ini driver MySQL.
	// tanda _ artinya kita tidak pakai langsung,
	// tapi hanya supaya drivernya terdaftar di database/sql
)

func New() (*sql.DB, error) {
	// fungsi ini bertugas membuat koneksi ke database
	// lalu mengembalikannya ke file lain (biasanya main.go)

	dsn := os.Getenv("DB_DSN")
	// ambil data koneksi database dari environment variable
	// contoh isi DB_DSN:
	// user:password@tcp(localhost:3306)/namadb?parseTime=true

	if dsn == "" {
		// kalau DB_DSN belum diset, langsung error
		// supaya aplikasi tidak jalan tanpa koneksi database
		return nil, errors.New("DB_DSN is not set")
	}

	db, err := sql.Open("mysql", dsn)
	// membuat koneksi ke MySQL
	// sebenarnya ini belum benar-benar connect,
	// hanya menyiapkan koneksi

	if err != nil {
		// kalau format DSN salah atau driver tidak ada, akan error
		return nil, err
	}

	// =========================
	// Pengaturan Connection Pool
	// =========================

	db.SetMaxOpenConns(25)
	// maksimal 25 koneksi aktif ke database
	// supaya database tidak kelebihan beban

	db.SetMaxIdleConns(10)
	// maksimal 10 koneksi yang standby (tidak dipakai tapi tetap terbuka)
	// supaya tidak perlu buka koneksi baru terus-menerus

	db.SetConnMaxLifetime(5 * time.Minute)
	// satu koneksi maksimal dipakai 5 menit
	// setelah itu akan dibuat koneksi baru
	// ini untuk mencegah koneksi lama bermasalah

	// =========================
	// Cek apakah database bisa dihubungi
	// =========================

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// kita kasih batas waktu 5 detik untuk cek koneksi
	// supaya tidak menunggu selamanya kalau database mati

	defer cancel()
	// setelah fungsi selesai, context dibersihkan

	if err := db.PingContext(ctx); err != nil {
		// benar-benar mencoba connect ke database
		// kalau database mati atau salah password, akan error

		_ = db.Close()
		// tutup koneksi kalau gagal

		return nil, err
	}

	// kalau semua berhasil, kembalikan koneksi database
	return db, nil
}

// 1️⃣ context itu apa?
// context dipakai untuk memberi batas waktu (timeout) atau membatalkan proses.
// Contoh di kode kamu: // ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// Artinya:
// “Coba konek ke database, tapi kalau lebih dari 5 detik, hentikan.”
// Kenapa penting?
// Supaya aplikasi tidak hang kalau database mati.

// 2️⃣ Connection Pool itu apa?
// *sql.DB bukan 1 koneksi.
// Dia adalah tempat penyimpanan banyak koneksi (pool).
// Bayangkan seperti:
// Gudang berisi 25 koneksi siap pakai.
// Kenapa perlu pool?
// Supaya tidak buat koneksi baru setiap request API.
// Lebih cepat dan hemat resource.
// Contoh di kode kamu: // db.SetMaxOpenConns(25)
// Maksimal 25 koneksi aktif ke database.

// 3️⃣ Kenapa pakai Environment Variable?
// dsn := os.Getenv("DB_DSN")
// Kenapa tidak langsung tulis username/password di kode?
// Karena:
// Lebih aman (tidak expose password di source code)
// Mudah ganti database tanpa ubah kode
// Best practice di production
// Biasanya diset di .env atau server environment.

// 4️⃣ Kenapa perlu PingContext?
// db.PingContext(ctx)
// Karena:
// sql.Open() belum benar-benar konek.
// PingContext() memastikan:
// Database benar-benar hidup dan bisa diakses.
// Kalau gagal → aplikasi langsung stop.
// Ini bagus untuk REST API supaya tidak jalan dalam keadaan rusak.
