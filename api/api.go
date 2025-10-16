package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strings"
	"time"

	_ "github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type API struct {
	db *pgxpool.Pool
}

func NewAPI(db *pgxpool.Pool) *API { return &API{db: db} }

func (a *API) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/books", a.listBooks)
	r.Post("/books", a.createBook)
	r.Put("/books/{id}", a.updateBook)
	r.Delete("/books/{id}", a.deleteBook)

	r.Get("/users", a.listUsers)
	r.Post("/users", a.createUser)
	r.Put("/users/{id}", a.updateUser)
	r.Delete("/users/{id}", a.deleteUser)

	r.Get("/authors", a.listAuthors)
	r.Post("/authors", a.createAuthor)
	r.Put("/authors/{id}", a.updateAuthor)
	r.Delete("/authors/{id}", a.deleteAuthor)

	r.Get("/places", a.listPlaces)
	r.Post("/places", a.createPlace)
	r.Put("/places/{id}", a.updatePlace)
	r.Delete("/places/{id}", a.deletePlace)

	r.Get("/publishers", a.listPublishers)
	r.Post("/publishers", a.createPublisher)
	r.Put("/publishers/{id}", a.updatePublisher)
	r.Delete("/publishers/{id}", a.deletePublisher)

	r.Get("/groups", a.listGroups)
	r.Post("/groups", a.createGroup)
	r.Put("/groups/{id}", a.updateGroup)
	r.Delete("/groups/{id}", a.deleteGroup)

	r.Get("/rooms", a.listRooms)
	r.Post("/rooms", a.createRoom)
	r.Put("/rooms/{id}", a.updateRoom)
	r.Delete("/rooms/{id}", a.deleteRoom)

	r.Get("/loans", a.listLoans)
	r.Post("/loans/issue", a.issueBook)
	r.Post("/loans/return", a.returnBook)
	r.Delete("/loans/{id}", a.deleteLoan)

	r.Get("/reports/issues-per-month", a.reportIssuesPerMonth)
	r.Get("/reports/oldest-per-room", a.reportOldestPerRoom)
	r.Get("/reports/rooms-by-groups", a.reportRoomsByGroups)
	r.Get("/reports/top5-last-month", a.reportTop5LastMonth)

	return r
}

type BookRow struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	PubYear       int    `json:"pub_year"`
	Pages         int    `json:"pages"`
	Copies        int    `json:"copies"`
	AuthorID      string `json:"author_id"`
	AuthorName    string `json:"author_name"`
	GroupID       string `json:"group_id"`
	GroupName     string `json:"group_name,omitempty"`
	PlaceID       string `json:"place_id"`
	PlaceName     string `json:"place_name"`
	PublisherID   string `json:"publisher_id"`
	PublisherName string `json:"publisher_name"`
	RoomID        string `json:"room_id"`
	RoomName      string `json:"room_name,omitempty"`
}

type BookUpsert struct {
	Title       string `json:"title"`
	AuthorID    string `json:"author_id"`
	PubYear     int    `json:"pub_year"`
	GroupID     string `json:"group_id"`
	PlaceID     string `json:"place_id"`
	PublisherID string `json:"publisher_id"`
	Pages       int    `json:"pages"`
	Copies      int    `json:"copies"`
	RoomID      string `json:"room_id"`
}

type UserRow struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	DateBirth    string  `json:"date_birth"`
	Phone        *string `json:"phone,omitempty"`
	TicketNumber int     `json:"ticket_number"`
}

type DictRow struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type LoanRow struct {
	ID         string  `json:"id"`
	UserID     string  `json:"user_id"`
	UserName   string  `json:"user_name"`
	BookID     string  `json:"book_id"`
	BookTitle  string  `json:"book_title"`
	DateIssue  string  `json:"date_issue"`
	DateReturn *string `json:"date_return,omitempty"`
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}
func bad(w http.ResponseWriter, err error, code int) {
	http.Error(w, err.Error(), code)
}
func parseDDMMYYYY(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}
	s = strings.ReplaceAll(s, ".", "/")
	t, err := time.Parse("02/01/2006", s)
	if err != nil {
		if t2, err2 := time.Parse("2006-01-02", s); err2 == nil {
			return t2, nil
		}
		return time.Time{}, fmt.Errorf("expected DD/MM/YYYY")
	}
	return t, nil
}
func dmy(t time.Time) string { return t.Format("02/01/2006") }

func (a *API) listBooks(w http.ResponseWriter, r *http.Request) {
	q := `
SELECT
  b.id, b.name, b.year_publication, b.pages, b.number_copies,
  a.id, a.name,
  g.id, g.name,
  p.id, p.name,
  ph.id, ph.name,
  rr.id, rr.name
FROM books b
JOIN book_authors a        ON a.id = b.author_id
JOIN book_groups  g        ON g.id = b.book_group_id
JOIN place_publications p  ON p.id = b.place_publication_id
JOIN publishing_houses ph  ON ph.id = b.published_house_id
JOIN reading_rooms rr      ON rr.id = b.reading_room_id
ORDER BY b.name, a.name, b.year_publication;
`
	rows, err := a.db.Query(context.Background(), q)
	if err != nil {
		bad(w, err, 500)
		return
	}
	defer rows.Close()

	var out []BookRow
	for rows.Next() {
		var br BookRow
		if err := rows.Scan(
			&br.ID, &br.Title, &br.PubYear, &br.Pages, &br.Copies,
			&br.AuthorID, &br.AuthorName,
			&br.GroupID, &br.GroupName,
			&br.PlaceID, &br.PlaceName,
			&br.PublisherID, &br.PublisherName,
			&br.RoomID, &br.RoomName,
		); err != nil {
			bad(w, err, 500)
			return
		}
		out = append(out, br)
	}
	writeJSON(w, out)
}

func (a *API) createBook(w http.ResponseWriter, r *http.Request) {
	var in BookUpsert
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		bad(w, err, 400)
		return
	}
	if in.Title == "" || in.AuthorID == "" || in.GroupID == "" || in.PlaceID == "" || in.PublisherID == "" || in.RoomID == "" || in.PubYear == 0 {
		bad(w, fmt.Errorf("missing required fields"), 400)
		return
	}
	if in.Copies <= 0 {
		in.Copies = 1
	}
	if in.Pages <= 0 {
		in.Pages = 1
	}

	q := `
INSERT INTO books(name, reading_room_id, author_id, place_publication_id, published_house_id,
                  year_publication, book_group_id, pages, number_copies)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
RETURNING id;
`
	var id string
	if err := a.db.QueryRow(context.Background(), q,
		in.Title, in.RoomID, in.AuthorID, in.PlaceID, in.PublisherID,
		in.PubYear, in.GroupID, in.Pages, in.Copies,
	).Scan(&id); err != nil {
		bad(w, err, 400)
		return
	}

	r2, _ := a.db.Query(context.Background(), `
SELECT
  b.id, b.name, b.year_publication, b.pages, b.number_copies,
  a.id, a.name,
  g.id, g.name,
  p.id, p.name,
  ph.id, ph.name,
  rr.id, rr.name
FROM books b
JOIN book_authors a        ON a.id = b.author_id
JOIN book_groups  g        ON g.id = b.book_group_id
JOIN place_publications p  ON p.id = b.place_publication_id
JOIN publishing_houses ph  ON ph.id = b.published_house_id
JOIN reading_rooms rr      ON rr.id = b.reading_room_id
WHERE b.id=$1`, id)
	defer func() {
		if r2 != nil {
			r2.Close()
		}
	}()
	if r2 != nil && r2.Next() {
		var br BookRow
		_ = r2.Scan(
			&br.ID, &br.Title, &br.PubYear, &br.Pages, &br.Copies,
			&br.AuthorID, &br.AuthorName,
			&br.GroupID, &br.GroupName,
			&br.PlaceID, &br.PlaceName,
			&br.PublisherID, &br.PublisherName,
			&br.RoomID, &br.RoomName,
		)
		writeJSON(w, br)
		return
	}
	writeJSON(w, map[string]string{"id": id})
}

func (a *API) updateBook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in BookUpsert
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		bad(w, err, 400)
		return
	}
	if in.Title == "" || in.AuthorID == "" || in.GroupID == "" || in.PlaceID == "" || in.PublisherID == "" || in.RoomID == "" || in.PubYear == 0 {
		bad(w, fmt.Errorf("missing required fields"), 400)
		return
	}
	if in.Copies <= 0 {
		in.Copies = 1
	}
	if in.Pages <= 0 {
		in.Pages = 1
	}

	cmd, err := a.db.Exec(context.Background(), `
UPDATE books
SET name=$1, reading_room_id=$2, author_id=$3, place_publication_id=$4, published_house_id=$5,
    year_publication=$6, book_group_id=$7, pages=$8, number_copies=$9
WHERE id=$10`, in.Title, in.RoomID, in.AuthorID, in.PlaceID, in.PublisherID, in.PubYear, in.GroupID, in.Pages, in.Copies, id)
	if err != nil {
		bad(w, err, 400)
		return
	}
	if cmd.RowsAffected() == 0 {
		bad(w, fmt.Errorf("not found"), 404)
		return
	}
	a.listBooks(w, r)
}

func (a *API) deleteBook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	cmd, err := a.db.Exec(context.Background(), `DELETE FROM books WHERE id=$1`, id)
	if err != nil {
		bad(w, err, 400)
		return
	}
	if cmd.RowsAffected() == 0 {
		bad(w, fmt.Errorf("not found"), 404)
		return
	}
	w.WriteHeader(204)
}

func (a *API) listUsers(w http.ResponseWriter, r *http.Request) {
	q := `SELECT id, name, to_char(date_birth,'DD/MM/YYYY'), phone, ticket_number FROM users ORDER BY ticket_number`
	rows, err := a.db.Query(context.Background(), q)
	if err != nil {
		bad(w, err, 500)
		return
	}
	defer rows.Close()

	var out []UserRow
	for rows.Next() {
		var u UserRow
		if err := rows.Scan(&u.ID, &u.Name, &u.DateBirth, &u.Phone, &u.TicketNumber); err != nil {
			bad(w, err, 500)
			return
		}
		out = append(out, u)
	}
	writeJSON(w, out)
}

func (a *API) createUser(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name      string  `json:"name"`
		DateBirth string  `json:"date_birth"`
		Phone     *string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		bad(w, err, 400)
		return
	}
	if in.Name == "" || in.DateBirth == "" {
		bad(w, fmt.Errorf("name and date_birth are required"), 400)
		return
	}
	d, err := parseDDMMYYYY(in.DateBirth)
	if err != nil {
		bad(w, fmt.Errorf("date_birth must be DD/MM/YYYY"), 400)
		return
	}

	var id string
	err = a.db.QueryRow(context.Background(),
		`INSERT INTO users(name, date_birth, phone) VALUES($1,$2,$3) RETURNING id`,
		in.Name, d, in.Phone,
	).Scan(&id)
	if err != nil {
		bad(w, err, 400)
		return
	}

	var u UserRow
	err = a.db.QueryRow(context.Background(),
		`SELECT id, name, to_char(date_birth,'DD/MM/YYYY'), phone, ticket_number FROM users WHERE id=$1`, id).
		Scan(&u.ID, &u.Name, &u.DateBirth, &u.Phone, &u.TicketNumber)
	if err != nil {
		bad(w, err, 500)
		return
	}
	writeJSON(w, u)
}

func (a *API) updateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in struct {
		Name      string  `json:"name"`
		DateBirth string  `json:"date_birth"`
		Phone     *string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		bad(w, err, 400)
		return
	}
	if in.Name == "" || in.DateBirth == "" {
		bad(w, fmt.Errorf("name and date_birth are required"), 400)
		return
	}
	d, err := parseDDMMYYYY(in.DateBirth)
	if err != nil {
		bad(w, fmt.Errorf("date_birth must be DD/MM/YYYY"), 400)
		return
	}

	cmd, err := a.db.Exec(context.Background(),
		`UPDATE users SET name=$1, date_birth=$2, phone=$3 WHERE id=$4`,
		in.Name, d, in.Phone, id)
	if err != nil {
		bad(w, err, 400)
		return
	}
	if cmd.RowsAffected() == 0 {
		bad(w, fmt.Errorf("not found"), 404)
		return
	}
	a.listUsers(w, r)
}

func (a *API) deleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	cmd, err := a.db.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, id)
	if err != nil {
		bad(w, err, 400)
		return
	}
	if cmd.RowsAffected() == 0 {
		bad(w, fmt.Errorf("not found"), 404)
		return
	}
	w.WriteHeader(204)
}

func scanDict(rows pgxRows) ([]DictRow, error) {
	defer rows.Close()
	var out []DictRow
	for rows.Next() {
		var d DictRow
		if err := rows.Scan(&d.ID, &d.Name); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

type pgxRows interface {
	Next() bool
	Scan(...any) error
	Close()
}

func (a *API) listAuthors(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query(context.Background(), `SELECT id, name FROM book_authors ORDER BY name`)
	if err != nil {
		bad(w, err, 500)
		return
	}
	out, err := scanDict(rows)
	if err != nil {
		bad(w, err, 500)
		return
	}
	writeJSON(w, out)
}
func (a *API) createAuthor(w http.ResponseWriter, r *http.Request) {
	a.createDict(w, r, "book_authors")
}
func (a *API) updateAuthor(w http.ResponseWriter, r *http.Request) {
	a.updateDict(w, r, "book_authors")
}
func (a *API) deleteAuthor(w http.ResponseWriter, r *http.Request) {
	a.deleteDict(w, r, "book_authors")
}

func (a *API) listPlaces(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query(context.Background(), `SELECT id, name FROM place_publications ORDER BY name`)
	if err != nil {
		bad(w, err, 500)
		return
	}
	out, err := scanDict(rows)
	if err != nil {
		bad(w, err, 500)
		return
	}
	writeJSON(w, out)
}
func (a *API) createPlace(w http.ResponseWriter, r *http.Request) {
	a.createDict(w, r, "place_publications")
}
func (a *API) updatePlace(w http.ResponseWriter, r *http.Request) {
	a.updateDict(w, r, "place_publications")
}
func (a *API) deletePlace(w http.ResponseWriter, r *http.Request) {
	a.deleteDict(w, r, "place_publications")
}

func (a *API) listPublishers(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query(context.Background(), `SELECT id, name FROM publishing_houses ORDER BY name`)
	if err != nil {
		bad(w, err, 500)
		return
	}
	out, err := scanDict(rows)
	if err != nil {
		bad(w, err, 500)
		return
	}
	writeJSON(w, out)
}
func (a *API) createPublisher(w http.ResponseWriter, r *http.Request) {
	a.createDict(w, r, "publishing_houses")
}
func (a *API) updatePublisher(w http.ResponseWriter, r *http.Request) {
	a.updateDict(w, r, "publishing_houses")
}
func (a *API) deletePublisher(w http.ResponseWriter, r *http.Request) {
	a.deleteDict(w, r, "publishing_houses")
}

func (a *API) listGroups(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query(context.Background(), `SELECT id, name FROM book_groups ORDER BY name`)
	if err != nil {
		bad(w, err, 500)
		return
	}
	out, err := scanDict(rows)
	if err != nil {
		bad(w, err, 500)
		return
	}
	writeJSON(w, out)
}
func (a *API) createGroup(w http.ResponseWriter, r *http.Request) { a.createDict(w, r, "book_groups") }
func (a *API) updateGroup(w http.ResponseWriter, r *http.Request) { a.updateDict(w, r, "book_groups") }
func (a *API) deleteGroup(w http.ResponseWriter, r *http.Request) { a.deleteDict(w, r, "book_groups") }

func (a *API) listRooms(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query(context.Background(), `SELECT id, name FROM reading_rooms ORDER BY name`)
	if err != nil {
		bad(w, err, 500)
		return
	}
	out, err := scanDict(rows)
	if err != nil {
		bad(w, err, 500)
		return
	}
	writeJSON(w, out)
}
func (a *API) createRoom(w http.ResponseWriter, r *http.Request) { a.createDict(w, r, "reading_rooms") }
func (a *API) updateRoom(w http.ResponseWriter, r *http.Request) { a.updateDict(w, r, "reading_rooms") }
func (a *API) deleteRoom(w http.ResponseWriter, r *http.Request) { a.deleteDict(w, r, "reading_rooms") }

func (a *API) createDict(w http.ResponseWriter, r *http.Request, table string) {
	var in DictRow
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		bad(w, err, 400)
		return
	}
	if in.Name == "" {
		bad(w, fmt.Errorf("name required"), 400)
		return
	}
	var id string
	err := a.db.QueryRow(context.Background(), fmt.Sprintf(`INSERT INTO %s(name) VALUES($1) RETURNING id`, table), in.Name).Scan(&id)
	if err != nil {
		bad(w, err, 400)
		return
	}
	writeJSON(w, DictRow{ID: id, Name: in.Name})
}
func (a *API) updateDict(w http.ResponseWriter, r *http.Request, table string) {
	id := chi.URLParam(r, "id")
	var in DictRow
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		bad(w, err, 400)
		return
	}
	if in.Name == "" {
		bad(w, fmt.Errorf("name required"), 400)
		return
	}
	cmd, err := a.db.Exec(context.Background(), fmt.Sprintf(`UPDATE %s SET name=$1 WHERE id=$2`, table), in.Name, id)
	if err != nil {
		bad(w, err, 400)
		return
	}
	if cmd.RowsAffected() == 0 {
		bad(w, fmt.Errorf("not found"), 404)
		return
	}
	writeJSON(w, DictRow{ID: id, Name: in.Name})
}
func (a *API) deleteDict(w http.ResponseWriter, r *http.Request, table string) {
	id := chi.URLParam(r, "id")
	cmd, err := a.db.Exec(context.Background(), fmt.Sprintf(`DELETE FROM %s WHERE id=$1`, table), id)
	if err != nil {
		bad(w, err, 400)
		return
	}
	if cmd.RowsAffected() == 0 {
		bad(w, fmt.Errorf("not found"), 404)
		return
	}
	w.WriteHeader(204)
}

func (a *API) listLoans(w http.ResponseWriter, r *http.Request) {
	active := r.URL.Query().Get("active") == "true"
	q := `
SELECT
  ab.id,
  u.id, u.name,
  b.id, b.name,
  to_char(ab.date_issue,'DD/MM/YYYY'),
  CASE WHEN ab.date_return IS NULL THEN NULL ELSE to_char(ab.date_return,'DD/MM/YYYY') END
FROM accounting_books ab
JOIN users u ON u.id = ab.user_id
JOIN books b ON b.id = ab.book_id
`
	if active {
		q += " WHERE ab.date_return IS NULL"
	}
	q += " ORDER BY ab.date_issue DESC, ab.id DESC LIMIT 300"

	rows, err := a.db.Query(context.Background(), q)
	if err != nil {
		bad(w, err, 500)
		return
	}
	defer rows.Close()

	var out []LoanRow
	for rows.Next() {
		var lr LoanRow
		if err := rows.Scan(&lr.ID, &lr.UserID, &lr.UserName, &lr.BookID, &lr.BookTitle, &lr.DateIssue, &lr.DateReturn); err != nil {
			bad(w, err, 500)
			return
		}
		out = append(out, lr)
	}
	writeJSON(w, out)
}

func (a *API) issueBook(w http.ResponseWriter, r *http.Request) {
	var in struct {
		UserID    string `json:"user_id"`
		BookID    string `json:"book_id"`
		IssueDate string `json:"issue_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		bad(w, err, 400)
		return
	}
	if in.UserID == "" || in.BookID == "" {
		bad(w, fmt.Errorf("user_id and book_id required"), 400)
		return
	}
	var d time.Time
	var err error
	if strings.TrimSpace(in.IssueDate) == "" {
		d = time.Now()
	} else {
		d, err = parseDDMMYYYY(in.IssueDate)
		if err != nil {
			bad(w, fmt.Errorf("issue_date must be DD/MM/YYYY"), 400)
			return
		}
	}
	var id string
	if err := a.db.QueryRow(context.Background(),
		`INSERT INTO accounting_books(user_id, book_id, date_issue) VALUES($1,$2,$3) RETURNING id`,
		in.UserID, in.BookID, d).Scan(&id); err != nil {
		bad(w, err, 400)
		return
	}
	writeJSON(w, map[string]string{"loan_id": id})
}

func (a *API) returnBook(w http.ResponseWriter, r *http.Request) {
	var in struct {
		LoanID     string `json:"loan_id"`
		ReturnDate string `json:"return_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		bad(w, err, 400)
		return
	}
	if in.LoanID == "" || in.ReturnDate == "" {
		bad(w, fmt.Errorf("loan_id and return_date required"), 400)
		return
	}
	d, err := parseDDMMYYYY(in.ReturnDate)
	if err != nil {
		bad(w, fmt.Errorf("return_date must be DD/MM/YYYY"), 400)
		return
	}

	var issued time.Time
	if err := a.db.QueryRow(context.Background(), `SELECT date_issue FROM accounting_books WHERE id=$1`, in.LoanID).Scan(&issued); err != nil {
		bad(w, fmt.Errorf("loan not found"), 404)
		return
	}
	if d.Before(issued) {
		bad(w, fmt.Errorf("Дата возврата %s раньше даты выдачи %s", dmy(d), dmy(issued)), 400)
		return
	}

	cmd, err := a.db.Exec(context.Background(), `UPDATE accounting_books SET date_return=$2 WHERE id=$1`, in.LoanID, d)
	if err != nil {
		bad(w, err, 400)
		return
	}
	if cmd.RowsAffected() == 0 {
		bad(w, fmt.Errorf("not found"), 404)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (a *API) deleteLoan(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	cmd, err := a.db.Exec(context.Background(), `DELETE FROM accounting_books WHERE id=$1`, id)
	if err != nil {
		bad(w, err, 400)
		return
	}
	if cmd.RowsAffected() == 0 {
		bad(w, fmt.Errorf("not found"), 404)
		return
	}
	w.WriteHeader(204)
}
