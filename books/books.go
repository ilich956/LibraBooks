// books.go
package books

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/sirupsen/logrus"
)

type Book struct {
	ID         int    `json:"id"`
	BookName   string `json:"book_name"`
	BookAuthor string `json:"book_author"`
	BookGenre  string `json:"book_genre"`
	BookDate   string `json:"book_date"`
	// User_id       int    `json:"user_id"`
	ImageFilename string `json:"image_filename"`
}

var DefaultBookService bookService

type bookService struct{}

func (bookService) ShowBooks(w http.ResponseWriter, r *http.Request, db *sql.DB) error {
	filter := r.URL.Query().Get("filter")
	sort := r.URL.Query().Get("sort")
	pageStr := r.URL.Query().Get("page")
	limit := 10

	// Convert page parameter to integer
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	offset := (page - 1) * limit

	query := "SELECT * FROM books"
	if filter != "" {
		query += " WHERE book_name LIKE '%" + filter + "%' OR book_author LIKE '%" + filter + "%' OR book_genre LIKE '%" + filter + "%'"
	}
	if sort != "" {
		query += " ORDER BY " + sort
	}
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)

	rows, err := db.Query(query)
	if err != nil {
		logrus.WithError(err).Error("Error querying database for books")
		return err
	}
	defer rows.Close()

	var books []Book

	for rows.Next() {
		var b Book

		err := rows.Scan(&b.ID, &b.BookName, &b.BookAuthor, &b.BookGenre, &b.BookDate)
		if err != nil {
			logrus.WithError(err).Error("Error scanning row for books")
			return err
		}
		b.ImageFilename = fmt.Sprintf("img%d.jpg", b.ID)
		books = append(books, b)
	}

	totalPages, err := getTotalPages(db, limit, filter)
	if err != nil {
		logrus.WithError(err).Error("Error calculating total number of pages")
		return err
	}

	err = renderBooksHTML(w, books, page, totalPages, filter, sort)
	if err != nil {
		logrus.WithError(err).Error("Error rendering HTML for books")
		return err
	}

	return nil
}

func getTotalPages(db *sql.DB, limit int, filter string) (int, error) {
	countQuery := "SELECT COUNT(*) FROM books"
	if filter != "" {
		countQuery += " WHERE book_name LIKE '%" + filter + "%' OR book_author LIKE '%" + filter + "%' OR book_genre LIKE '%" + filter + "%'"
	}

	var totalBooks int
	err := db.QueryRow(countQuery).Scan(&totalBooks)
	if err != nil {
		return 0, err
	}

	totalPages := (totalBooks + limit - 1) / limit
	return totalPages, nil
}

func renderBooksHTML(w http.ResponseWriter, books []Book, currentPage, totalPages int, filter, sort string) error {
	tmpl, err := template.ParseFiles("library.html")
	if err != nil {
		return err
	}

	data := struct {
		Books       []Book
		PrevPage    int
		Pages       []int
		NextPage    int
		CurrentPage int
		TotalPages  int
		Filter      string
		Sort        string
	}{
		Books:       books,
		CurrentPage: currentPage,
		TotalPages:  totalPages,
		Filter:      filter,
		Sort:        sort,
	}

	for i := 1; i <= totalPages; i++ {
		data.Pages = append(data.Pages, i)
	}

	if currentPage > 1 {
		data.PrevPage = currentPage - 1
	}
	if currentPage < totalPages {
		data.NextPage = currentPage + 1
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		return err
	}

	return nil
}
