=======================================================================
simple API notes with dependency service-repository pattern,
dengan CRUD lengkap + Soft Delete (Archive/Unarchive) + Filtering

project memiliki:
-Layered dependency
-Clean separation of concern
-CRUD lengkap
-Soft delete strategy
-Filtering support
-Context-aware design
-HTTP timeout protection
-Graceful shutdown
-Maintainable structure
-Scalable structure untuk multi-resource

flow sistem saat API berjalan (runtime flow)
= Client Request → Route → Handler → Service → Repository → Database → balik lagi ke atas (response)

Secara runtime, alurnya:
→ Client Request
→ Route (menentukan endpoint & method)
→ Handler (parsing request, validasi HTTP, panggil service)
→ Service (business logic / rule domain)
→ Repository (eksekusi query SQL)
→ Database (simpan / ambil data)
→ hasil dikembalikan naik lewat layer yang sama ( database -> repository -> service -> handler -> client )
→ Response dikirim ke client

struktur project ini

notes-api/
├── cmd/
│ └── server/
│ └── main.go  
│
├── internal/
│ ├── route/
│ │ └── route.go  
│ │
│ ├── handler/
│ │ └── handler.go  
│ │
│ ├── modul/
│ │ └── note.go  
│ │
│ ├── database/
│ | └── database.go  
| |
│ ├── repository/
│ │ └── noterepository.go  
│ │
│ └── service/
│ └── noteservice.go  
│
├── migrations/
│ └── notes_api.sql
│
├── go.mod
└── go.sum

api yang support project ini

| Method | Endpoint                |
| ------ | ----------------------- |
| POST   | `/notes`                |
| GET    | `/notes`                |
| GET    | `/notes/{id}`           |
| GET    | `/notes?archived=true`  |
| PUT    | `/notes/{id}`           |
| PATCH  | `/notes/{id}/archive`   |
| PATCH  | `/notes/{id}/unarchive` |
| DELETE | `/notes/{id}`           |

setup project

- clone project ke local kamu
- setup database local mysql kamu menggunakan file yang sudah disediakan
- copy/rename .env.example menjadi .env dan setting isinya sesuai database kamu
- kemudian di path /cmd/server jalankan perintah go run . untuk jalankan servernya
- pastikan sebelum jalankan go run . mysql sudah dihidupkan agar tidak error
- pastikan sebelum jalankan go run . mysql sudah dihidupkan agar tidak error
- akses url ini di browser untuk lihat hasilnya http://localhost:8080/notes

test api menggunakan curl gitbash

perintah curl terminal git bash
create
curl.exe -X POST http://localhost:8080/notes -H "Content-Type: application/json" -d '{ "title": "Belajar Clean Code", "content": "Refactor ala senior" }'

read
curl.exe http://localhost:8080/notes

read by id
curl.exe http://localhost:8080/notes/3

update
curl.exe -X PUT http://localhost:8080/notes/3 -H "Content-Type: application/json" -d '{ "title": "Updated Title", "content": "Updated content" }'

archive
curl.exe -X PATCH http://localhost:8080/notes/3/archive

lihat list archive
curl.exe http://localhost:8080/notes?archived=true

unarchive
curl.exe -X PATCH http://localhost:8080/notes/3/unarchive

delete
curl.exe -X DELETE http://localhost:8080/notes/3

by = denisetiawan-san (api builder).
