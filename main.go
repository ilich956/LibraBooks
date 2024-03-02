package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	"main.go/books"
	"main.go/token"
	"main.go/users"
)

const (
	port        = ":8000"
	connStr     = "postgres://postgres:bayipket@localhost/adv_database?sslmode=disable"
	driverName  = "postgres"
	tableName   = "user_table"
	createTable = `
		CREATE TABLE IF NOT EXISTS user_table (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255),
			username VARCHAR(255),
			password VARCHAR(255),
			isActivated BOOLEAN DEFAULT FALSE
		);
	`
)

type ResponseData struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

var db *sql.DB
var limiter = rate.NewLimiter(1, 3)
var log = logrus.New()

func main() {
	var err error
	db, err = sql.Open(driverName, connStr)
	if err != nil {
		fmt.Println("Error opening database:", err)
		return
	}
	defer db.Close()

	_, err = db.Exec(createTable)
	if err != nil {
		fmt.Println("Error creating user_table:", err)
		return
	}
	router := mux.NewRouter()

	// router.Use(middleware.AttachTokenToRequest)

	router.HandleFunc("/", getRegisterPage)
	router.HandleFunc("/login_form", rateLimitedHandler(getLoginPage))
	router.HandleFunc("/checkmail", rateLimitedHandler(getCheckMailPage))
	router.HandleFunc("/activate/{link}", activate)
	router.HandleFunc("/register", rateLimitedHandler(registerUser))
	router.HandleFunc("/login", rateLimitedHandler(loginUser))
	router.HandleFunc("/userList", rateLimitedHandler(getUserList))
	router.HandleFunc("/library", rateLimitedHandler(getLibrary))
	router.HandleFunc("/profile", rateLimitedHandler(getProfile))
	router.HandleFunc("/changepsswd", rateLimitedHandler(getPsswd))
	router.HandleFunc("/change", rateLimitedHandler(changePassword))
	router.HandleFunc("/sendotp", rateLimitedHandler(handleOTP))
	router.HandleFunc("/otp", rateLimitedHandler(getOTP))
	router.HandleFunc("/borrow", rateLimitedHandler(handleBorrowBook))

	// Serving static files
	router.PathPrefix("/book-covers/").Handler(http.StripPrefix("/book-covers/", http.FileServer(http.Dir("book-covers"))))
	router.PathPrefix("/styles/").Handler(http.StripPrefix("/styles/", http.FileServer(http.Dir("styles"))))

	log.Info("Server listening on port", port)
	fmt.Println("Server listening on port", port)
	http.ListenAndServe(port, router)
	// http.Handle("/book-covers/", http.StripPrefix("/book-covers/", http.FileServer(http.Dir("book-covers"))))
	// http.Handle("/styles/", http.StripPrefix("/styles/", http.FileServer(http.Dir("styles"))))
	// http.HandleFunc("/", rateLimitedHandler(userHandler))

	// log.Info("Server listening on port", port)
	// fmt.Println("Server listening on port", port)
	// http.ListenAndServe(port, nil)
}

func rateLimitedHandler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			log.Warn("Rate limit exceeded")
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	}
}

// func userHandler(w http.ResponseWriter, r *http.Request) {
// 	switch r.URL.Path {
// 	case "/":
// 		getRegisterPage(w, r) //handleRegistration(w, r)
// 	case "/login_form":
// 		getLoginPage(w, r) //handleLogin(w, r)
// 	case "/checkmail":
// 		getCheckMailPage(w, r)
// 	case "/activate/{link}":
// 		activate(w, r)
// 	case "/register":
// 		registerUser(w, r)
// 	case "/login":
// 		loginUser(w, r)
// 	case "/userList":
// 		getUserList(w, r)
// 	case "/library":
// 		getLibrary(w, r)
// 	}
// 	// http.Error(w, "Method not all", http.StatusMethodNotAllowed)
// }

func getCheckMailPage(w http.ResponseWriter, r *http.Request) {
	templating(w, "checkemail.html", nil)
}

func getOTP(w http.ResponseWriter, r *http.Request) {
	templating(w, "otp-page.html", nil)
}

func getProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return // Return after calling http.Error
	}

	err := books.DefaultBookService.ShowBorrowedBooks(w, r, db)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return // Return after calling http.Error
	}
}

func getPsswd(w http.ResponseWriter, r *http.Request) {
	templating(w, "change-password.html", nil)
}

// BIBLIOTEKAAAAAAAAAAAAA
func getLibrary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := books.DefaultBookService.ShowBooks(w, r, db)
	if err != nil {
		http.Error(w, "Error showing library", http.StatusInternalServerError)
	}
}

func handleBorrowBook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := books.DefaultBookService.BorrowBook(w, r, db)
	if err != nil {
		http.Error(w, "Error showing library", http.StatusInternalServerError)
	}
}

func handleOTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Warn("Invalid HTTP method for handleOTP")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user credentials from the request
	email := r.FormValue("email")

	// Validate if username and password are provided
	if email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	// Authenticate user with provided credentials
	err := users.DefaultUserService.OTPservice(db, email)
	if err != nil {
		log.WithError(err).Warn("Authentication failed")
		http.Error(w, "error otp", http.StatusUnauthorized)
		return
	}

	http.Redirect(w, r, "/login_form", http.StatusSeeOther)
}

func activate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	link := mux.Vars(r)["link"] // Get the variables from the request URL
	// Extract the value of ":link" parameter
	fmt.Println(link)
	err := users.DefaultUserService.Activate(db, link)
	if err != nil {
		http.Error(w, "Error activating account", http.StatusInternalServerError)
	}

	http.Redirect(w, r, "/login_form", http.StatusSeeOther)
}

func changePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	password := r.FormValue("password")
	newpassword := r.FormValue("newpassword")
	email := r.FormValue("email")

	if password == "" || newpassword == "" || email == "" {
		http.Error(w, " Password is required", http.StatusBadRequest)
		return
	}

	err := users.DefaultUserService.ChangePassword(db, email, password, newpassword)
	if err != nil {
		log.WithError(err).Warn("Password changing failed")
		http.Error(w, "Invalid password", http.StatusUnauthorized)
		return
	}

	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

////////////////////////////////////////////////////////////////////////

func getUserList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Warn("Invalid HTTP method for getUserList")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := users.DefaultUserService.ShowUserList(w, r, db)
	if err != nil {
		log.WithError(err).Error("Error showing user list")
		http.Error(w, "Error showing user list", http.StatusInternalServerError)
	}
}

func registerUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Warn("Invalid HTTP method for registerUser")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	newUser := getUser(r)

	token, err := token.GenerateToken()
	if err != nil {
		log.WithError(err).Error("Error generating token")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:  "token",
		Value: token,
	})

	err = users.DefaultUserService.CreateUser(db, newUser, token)
	if err != nil {
		fileName := "register.html"
		t, _ := template.ParseFiles(fileName)
		t.ExecuteTemplate(w, fileName, "Email is already registered")
		return
	}

	log.WithFields(logrus.Fields{
		"action": "register",
		"user":   newUser,
	}).Info("User registered successfully")

	http.Redirect(w, r, "/checkmail", http.StatusSeeOther)
}

func loginUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Warn("Invalid HTTP method for loginUser")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user credentials from the request
	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	token, err := token.GenerateToken()
	if err != nil {
		log.WithError(err).Error("Error generating token")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:  "token",
		Value: token,
	})

	// Authenticate user with provided credentials
	err = users.DefaultUserService.AuthenticateUser(db, username, password)
	if err != nil {
		log.WithError(err).Warn("Authentication failed")
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	http.Redirect(w, r, "/library", http.StatusSeeOther)
}

func getUser(r *http.Request) users.User {
	email := r.FormValue("email")
	username := r.FormValue("username")
	password := r.FormValue("password")

	return users.User{
		Email:    email,
		Username: username,
		Password: password,
	}
}

func getRegisterPage(w http.ResponseWriter, r *http.Request) {
	templating(w, "register.html", nil)
}

func getLoginPage(w http.ResponseWriter, r *http.Request) {
	templating(w, "login.html", nil)

}

func templating(w http.ResponseWriter, filename string, data interface{}) {
	t, _ := template.ParseFiles(filename)
	t.ExecuteTemplate(w, filename, data)
}

func init() {
	// Create or open the log file
	file, err := os.OpenFile("logfile.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		// Set the logrus output to the file
		log.SetOutput(file)
	} else {
		// If unable to open the log file, log to standard output
		log.Warn("Failed to open log file. Logging to standard output.")
	}

	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetLevel(logrus.InfoLevel)

	log.Info("Logging initialized")
}
