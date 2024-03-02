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

	router.HandleFunc("/", getRegisterPage)
	router.HandleFunc("/login_form", rateLimitedHandler(getLoginPage))
	router.HandleFunc("/checkmail", rateLimitedHandler(getCheckMailPage))
	router.HandleFunc("/activate/{link}", activate)
	router.HandleFunc("/register", rateLimitedHandler(registerUser))
	router.HandleFunc("/login", rateLimitedHandler(loginUser))
	router.HandleFunc("/userList", rateLimitedHandler(getUserList))
	router.HandleFunc("/library", rateLimitedHandler(getLibrary))

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

// BIBLIOTEKAAAAAAAAAAAAA
func getLibrary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := books.DefaultBookService.ShowBooks(w, r, db)
	if err != nil {
		http.Error(w, "Error showing user list", http.StatusInternalServerError)
	}
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
	err := users.DefaultUserService.CreateUser(db, newUser)
	if err != nil {
		fileName := "register.html"
		t, _ := template.ParseFiles(fileName)
		t.ExecuteTemplate(w, fileName, "Create unique username")
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

	// Validate if username and password are provided
	if username == "" || password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Authenticate user with provided credentials
	err := users.DefaultUserService.AuthenticateUser(db, username, password)
	if err != nil {
		log.WithError(err).Warn("Authentication failed")
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Authentication successful, redirect user to library page
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

///////////////////////////////////////////////

// func handleGetUser(w http.ResponseWriter, r *http.Request) {
// 	users, err := getFromDB()
// 	if err != nil {
// 		http.Error(w, "Status", http.StatusInternalServerError)
// 		return
// 	}
// 	ts, err := template.ParseFiles("userList.html")
// 	if err != nil {
// 		http.Error(w, "Loh", http.StatusInternalServerError)
// 		return
// 	}
// 	ts.Execute(w, users)

// }

// func getFromDB() ([]RegistrationData, error) {
// 	rows, err := db.Query(`SELECT "username","password" FROM user_table`)
// 	if err != nil {
// 		return nil, fmt.Errorf("Error querying database: %s", err)
// 	}
// 	defer rows.Close()

// 	var users []RegistrationData

// 	for rows.Next() {
// 		var u RegistrationData
// 		err := rows.Scan(&u.Username, &u.Password)
// 		if err != nil {
// 			return nil, fmt.Errorf("Error scanning row: %s", err)
// 		}
// 		users = append(users, u)
// 	}
// 	fmt.Print(users)
// 	return users, nil
// }

// func handleRegistration(w http.ResponseWriter, r *http.Request) {
// 	body, err := ioutil.ReadAll(r.Body)
// 	if err != nil {
// 		handleError(w, "Invalid JSON format")
// 		return
// 	}

// 	var registrationData RegistrationData
// 	err = json.Unmarshal(body, &registrationData)
// 	if err != nil {
// 		handleError(w, "Invalid JSON format")
// 		return
// 	}

// 	if registrationData.Password != registrationData.ConfirmPassword {
// 		handleError(w, "Password and confirm password do not match")
// 		return
// 	}

// 	// Insert user registration data into the database
// 	err = insertUser(registrationData)
// 	if err != nil {
// 		handleError(w, "Error inserting user data into the database")
// 		return
// 	}

// 	fmt.Printf("Received registration data: %+v\n", registrationData)

// 	response := ResponseData{
// 		Status:  "success",
// 		Message: "Registration data successfully received and inserted into the database",
// 	}

// 	responseJSON, err := json.Marshal(response)
// 	if err != nil {
// 		handleError(w, "Error")
// 		return
// 	}

// 	w.Header().Set("Content-Type", "application/json")
// 	w.Write(responseJSON)
// }

// func insertUser(data RegistrationData) error {
// 	_, err := db.Exec("INSERT INTO "+tableName+" (name, email, username, password) VALUES ($1, $2, $3, $4)",
// 		data.Name, data.Email, data.Username, data.Password)
// 	return err

// 	//Remove name
// }

// func handleError(w http.ResponseWriter, message string) {
// 	response := ResponseData{
// 		Status:  "400",
// 		Message: message,
// 	}

// 	responseJSON, err := json.Marshal(response)
// 	if err != nil {
// 		http.Error(w, "Error", http.StatusInternalServerError)
// 		return
// 	}

// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusBadRequest)
// 	w.Write(responseJSON)
// }
