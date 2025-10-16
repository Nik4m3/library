package api

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"net/http"
	"strings"
	"time"
)

// Посчитать за каждый месяц года, определенного пользователем, количество выдач книг.
func (a *API) reportIssuesPerMonth(w http.ResponseWriter, r *http.Request) {
	year := strings.TrimSpace(r.URL.Query().Get("year"))
	if year == "" {
		year = fmt.Sprintf("%d", time.Now().Year())
	}
	q := `
SELECT
  to_char(make_date($1::int, m, 1), 'TMMonth') AS month,
  COUNT(ab.id) AS issues
FROM generate_series(1, 12) AS m
LEFT JOIN accounting_books ab
  ON ab.date_issue >= make_date($1::int, m, 1)
 AND ab.date_issue <  (make_date($1::int, m, 1) + INTERVAL '1 month')
GROUP BY m
ORDER BY m;`
	rows, err := a.db.Query(context.Background(), q, year)
	if err != nil {
		bad(w, err, 500)
		return
	}
	defer rows.Close()
	type row struct {
		Month  string `json:"month"`
		Issues int    `json:"issues"`
	}
	var out []row
	for rows.Next() {
		var rr row
		if err := rows.Scan(&rr.Month, &rr.Issues); err != nil {
			bad(w, err, 500)
			return
		}
		out = append(out, rr)
	}
	writeJSON(w, out)
}

// Вывести название и возраст книги самой старой книги в каждом из залов.
func (a *API) reportOldestPerRoom(w http.ResponseWriter, r *http.Request) {
	q := `
SELECT
  rr.name AS room,
  b.name  AS title,
  (EXTRACT(YEAR FROM age(make_date(b.year_publication,1,1))))::int AS age
FROM reading_rooms rr
JOIN books b ON b.reading_room_id = rr.id
WHERE b.year_publication = (SELECT MIN(b2.year_publication) FROM books b2 WHERE b2.reading_room_id = rr.id)
GROUP BY rr.name, b.name, b.year_publication
ORDER BY rr.name;
`
	rows, err := a.db.Query(context.Background(), q)
	if err != nil {
		bad(w, err, 500)
		return
	}
	defer rows.Close()
	type row struct {
		Room  string `json:"room"`
		Title string `json:"title"`
		Age   int    `json:"age"`
	}
	var out []row
	for rows.Next() {
		var rr row
		if err := rows.Scan(&rr.Room, &rr.Title, &rr.Age); err != nil {
			bad(w, err, 500)
			return
		}
		out = append(out, rr)
	}
	writeJSON(w, out)
}

// Вывести читальный зал в котором содержаться книги только заданных пользователем типов (типов при поиске может быть определено несколько)
func (a *API) reportRoomsByGroups(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var raw []string
	if csv := strings.TrimSpace(r.URL.Query().Get("groups")); csv != "" {
		for _, s := range strings.Split(csv, ",") {
			if s = strings.TrimSpace(s); s != "" {
				raw = append(raw, s)
			}
		}
	}
	for _, s := range r.URL.Query()["group"] {
		if s = strings.TrimSpace(s); s != "" {
			raw = append(raw, s)
		}
	}
	if len(raw) == 0 {
		writeJSON(w, []string{})
		return
	}

	seen := make(map[uuid.UUID]struct{}, len(raw))
	uu := make([]uuid.UUID, 0, len(raw))
	for _, s := range raw {
		id, err := uuid.Parse(s)
		if err != nil {
			bad(w, fmt.Errorf("invalid uuid: %q", s), http.StatusBadRequest)
			return
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uu = append(uu, id)
	}

	const q = `
SELECT DISTINCT rr.name
FROM books b
JOIN reading_rooms rr ON rr.id = b.reading_room_id
WHERE b.book_group_id = ANY ($1::uuid[])
ORDER BY rr.name;
`
	rows, err := a.db.Query(ctx, q, uu)
	if err != nil {
		bad(w, err, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			bad(w, err, http.StatusInternalServerError)
			return
		}
		out = append(out, name)
	}
	writeJSON(w, out)
}

// Вывести 5 лучших книг, которые за прошедший месяц пользовались наибольшим спросом.
func (a *API) reportTop5LastMonth(w http.ResponseWriter, r *http.Request) {
	q := `
SELECT b.name AS title, COUNT(*) AS cnt
FROM accounting_books ab
JOIN books b ON b.id = ab.book_id
WHERE ab.date_issue >= date_trunc('month', current_date) - interval '1 month'
  AND ab.date_issue <  date_trunc('month', current_date)
GROUP BY b.id, b.name
ORDER BY cnt DESC, b.name
LIMIT 5;
`
	rows, err := a.db.Query(context.Background(), q)
	if err != nil {
		bad(w, err, 500)
		return
	}
	defer rows.Close()
	type row struct {
		Title string `json:"title"`
		Count int    `json:"count"`
	}
	var out []row
	for rows.Next() {
		var rr row
		if err := rows.Scan(&rr.Title, &rr.Count); err != nil {
			bad(w, err, 500)
			return
		}
		out = append(out, rr)
	}
	writeJSON(w, out)
}
