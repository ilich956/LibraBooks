// books.go
package books

import (
	"database/sql"
	"errors"
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
	Borrowed      string `json:"borrowed`
}

type BorrowedBook struct {
	BookName   string `json:"book_name"`
	BookAuthor string `json:"book_author"`
	BookGenre  string `json:"book_genre"`
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

	query := "SELECT * FROM books WHERE borrowed = false"
	if filter != "" {
		query += " AND (book_name LIKE '%" + filter + "%' OR book_author LIKE '%" + filter + "%' OR book_genre LIKE '%" + filter + "%')"
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

		err := rows.Scan(&b.ID, &b.BookName, &b.BookAuthor, &b.BookGenre, &b.BookDate, &b.Borrowed)
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

func (bookService) BorrowBook(w http.ResponseWriter, r *http.Request, db *sql.DB) error {
	// Parse form data to get the book ID
	err := r.ParseForm()
	if err != nil {
		return err
	}

	bookID := r.Form.Get("book_id")
	if bookID == "" {
		return errors.New("book ID is required")
	}

	cookie, err := r.Cookie("token")
	if err != nil {
		return errors.New("token not found in cookies")
	}

	token := cookie.Value

	// Convert book ID to integer
	id, err := strconv.Atoi(bookID)
	if err != nil {
		return err
	}

	_, err = db.Exec("INSERT INTO borrowings (book_id, user_id, borrowed_at) VALUES ($1, (SELECT id FROM user_table WHERE token = $2), CURRENT_TIMESTAMP)", id, token)
	if err != nil {
		return err
	}

	// Update the database to mark the book as borrowed
	_, err = db.Exec("UPDATE books SET borrowed = true WHERE id = $1", id)
	if err != nil {
		return err
	}

	// Respond with a success message or any necessary response
	fmt.Fprintf(w, "Book with ID %d has been borrowed successfully", id)

	return nil
}

func (bookService) ShowBorrowedBooks(w http.ResponseWriter, r *http.Request, db *sql.DB) error {
	// Retrieve user ID from token in request cookies
	cookie, err := r.Cookie("token")
	if err != nil {
		return errors.New("token not found in cookies")
	}
	token := cookie.Value

	var userID int
	var username string

	err = db.QueryRow("SELECT id, username FROM user_table WHERE token = $1", token).Scan(&userID, &username)
	if err != nil {
		return err
	}

	// Query the database to get borrowed books for the user
	rows, err := db.Query("SELECT book_name, book_author, book_genre FROM books INNER JOIN borrowings ON books.id = borrowings.book_id WHERE borrowings.user_id = (SELECT id FROM user_table WHERE token = $1)", token)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Create a slice to hold BorrowedBook objects
	var borrowedBooks []BorrowedBook

	// Iterate over the rows and populate the borrowedBooks slice
	for rows.Next() {
		var b BorrowedBook
		err := rows.Scan(&b.BookName, &b.BookAuthor, &b.BookGenre)
		if err != nil {
			return err
		}
		borrowedBooks = append(borrowedBooks, b)
	}

	// Render the borrowed books HTML template
	tmpl, err := template.ParseFiles("profile.html")
	if err != nil {
		return err
	}

	data := struct {
		Username      string
		BorrowedBooks []BorrowedBook
	}{
		Username:      username,
		BorrowedBooks: borrowedBooks,
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		return err
	}

	return nil
}

func (bookService) ReturnBook(w http.ResponseWriter, r *http.Request, db *sql.DB) error {
	// Parse form data to get the book name
	err := r.ParseForm()
	if err != nil {
		return err
	}

	bookName := r.Form.Get("book_name")
	if bookName == "" {
		return errors.New("book name is required")
	}

	// Retrieve user ID using token from cookies
	cookie, err := r.Cookie("token")
	if err != nil {
		return errors.New("token not found in cookies")
	}

	token := cookie.Value
	var userID int
	err = db.QueryRow("SELECT id FROM user_table WHERE token = $1", token).Scan(&userID)
	if err != nil {
		return err
	}

	// Check if the user has borrowed the book
	var exists bool
	err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM books INNER JOIN borrowings ON books.id = borrowings.book_id WHERE books.book_name = $1 AND borrowings.user_id = $2)", bookName, userID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("the user has not borrowed this book")
	}

	// Remove borrowing record from the database
	_, err = db.Exec("DELETE FROM borrowings WHERE book_id IN (SELECT id FROM books WHERE book_name = $1) AND user_id = $2", bookName, userID)
	if err != nil {
		return err
	}

	// Update the database to mark the book as returned
	_, err = db.Exec("UPDATE books SET borrowed = false WHERE book_name = $1", bookName)
	if err != nil {
		return err
	}

	// Respond with a success message or any necessary response
	fmt.Fprintf(w, "Book '%s' has been returned successfully", bookName)

	return nil
}
