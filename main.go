package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	"main.go/books"
	"main.go/mail-service"
	"main.go/token"
	"main.go/users"
)

func goDotEnvVariable(key string) string {

	// load .env file
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatal("Error loading .env file", err)

	}

	return os.Getenv(key)
}

var (
	port        = goDotEnvVariable("PORT")
	connStr     = goDotEnvVariable("CONN_STR")
	driverName  = goDotEnvVariable("DRIVERNAME")
	tableName   = goDotEnvVariable("TABLENAME")
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

type EmailData struct {
	Email   string `json:"email"`
	Content string `json:"content"`
}

var db *sql.DB
var limiter = rate.NewLimiter(rate.Limit(100)/3, 100)
var log = logrus.New()

func main() {
	var err error
	db, err = sql.Open(driverName, connStr)
	if err != nil {
		fmt.Println("Error opening database:", err)
		return
	}
	defer db.Close()

	router := mux.NewRouter()

	router.HandleFunc("/", getRegisterPage)
	router.HandleFunc("/login_form", rateLimitedHandler(getLoginPage))
	router.HandleFunc("/checkmail", rateLimitedHandler(getCheckMailPage))
	router.HandleFunc("/activate/{link}", activate)
	router.HandleFunc("/register", rateLimitedHandler(registerUser))
	router.HandleFunc("/login", rateLimitedHandler(loginUser))
	router.HandleFunc("/sendotp", rateLimitedHandler(handleOTP))
	router.HandleFunc("/otp", rateLimitedHandler(getOTP))

	router.HandleFunc("/userList", rateLimitedHandler(getUserList))
	router.HandleFunc("/sendemail", rateLimitedHandler(handleSendEmail))
	router.HandleFunc("/sendemailall", rateLimitedHandler(handleSendEmailAll))

	router.HandleFunc("/library", rateLimitedHandler(getLibrary))
	router.HandleFunc("/profile", rateLimitedHandler(getProfile))

	router.HandleFunc("/changepsswd", rateLimitedHandler(getPsswd))
	router.HandleFunc("/change", rateLimitedHandler(changePassword))

	router.HandleFunc("/borrow", rateLimitedHandler(handleBorrowBook))
	router.HandleFunc("/return", rateLimitedHandler(handleReturnBook))
	router.HandleFunc("/deleteuser", rateLimitedHandler(handleDeleteUser))

	// Serving static files
	router.PathPrefix("/book-covers/").Handler(http.StripPrefix("/book-covers/", http.FileServer(http.Dir("book-covers"))))
	router.PathPrefix("/styles/").Handler(http.StripPrefix("/styles/", http.FileServer(http.Dir("styles"))))

	log.Info("Server listening on port", port)
	fmt.Println("Server listening on port", port)
	http.ListenAndServe(port, router)
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

func getCheckMailPage(w http.ResponseWriter, r *http.Request) {
	templating(w, "checkemail.html", nil)
}

func getOTP(w http.ResponseWriter, r *http.Request) {
	templating(w, "otp-page.html", nil)
}

func getProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := books.DefaultBookService.ShowBorrowedBooks(w, r, db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getPsswd(w http.ResponseWriter, r *http.Request) {
	templating(w, "change-password.html", nil)
}

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

func handleSendEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var emailData EmailData
	err := json.NewDecoder(r.Body).Decode(&emailData)
	if err != nil {
		http.Error(w, "Error decoding JSON", http.StatusBadRequest)
		return
	}

	email := emailData.Email
	content := emailData.Content

	fmt.Println("Email:", email)
	fmt.Println("Content:", content)

	err = mail.SendEmail(email, content)
	if err != nil {
		http.Error(w, "Error sending email", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func handleSendEmailAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := mail.SendEmailAll(db, 10000)
	if err != nil {
		http.Error(w, "Error sending email", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}
	userID := r.Form.Get("user_id")

	err = users.DefaultUserService.DeleteUser(db, userID)
	if err != nil {
		http.Error(w, "Error deleting user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
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

func handleReturnBook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := books.DefaultBookService.ReturnBook(w, r, db)
	if err != nil {
		http.Error(w, "Error returning book", http.StatusInternalServerError)
	}
}

func handleOTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Warn("Invalid HTTP method for handleOTP")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	email := r.FormValue("email")

	if email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

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
	link := mux.Vars(r)["link"]
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
	username := r.FormValue("email")
	password := r.FormValue("password")

	if username == "" || password == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
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

	err = users.DefaultUserService.AuthenticateUser(db, username, password, token)
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
