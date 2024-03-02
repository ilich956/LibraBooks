package users

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"text/template"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"main.go/mail-service"
)

const (
	port       = ":8080"
	connStr    = "postgres://postgres:bayipket@localhost/adv_database?sslmode=disable"
	driverName = "postgres"
	tableName  = "user_table"
)

type User struct {
	Email    string
	Username string
	Password string
}

type authUser struct {
	Email        string
	Username     string
	PasswordHash string
	Confirmation string
}

var DefaultUserService userService
var log = logrus.New()

type userService struct {
}

func goDotEnvVariable(key string) string {

	// load .env file
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatal("Error loading .env file", err)

	}

	return os.Getenv(key)
}

func getPasswordHash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 0)
	return string(hash), err
}

func (userService) CreateUser(db *sql.DB, newUser User) error {
	err := checkUsername(db, newUser.Username)

	if err != nil {
		log.Warn("User already exists")
		return errors.New("user already exists")
	}

	passwordHash, err := getPasswordHash(newUser.Password)
	if err != nil {
		log.WithError(err).Error("Error generating password hash")
		return err
	}

	confiramtionString := uuid.New()
	fmt.Println(confiramtionString.String())

	newAuthUser := authUser{
		Email:        newUser.Email,
		Username:     newUser.Username,
		PasswordHash: passwordHash,
		Confirmation: confiramtionString.String(),
	}

	err = insertUserDB(db, newAuthUser)
	if err != nil {
		log.WithError(err).Error("Error inserting user into database")
		return err
	}

	//Confirmation link
	apiURL := goDotEnvVariable("API_URL")
	confiramtionLink := "activate/" + confiramtionString.String()
	fullLink := apiURL + "/" + confiramtionLink
	fmt.Println(fullLink)

	err = mail.SendConfirmationEmail(newAuthUser.Email, fullLink)
	if err != nil {
		log.WithError(err).Error("error sending confirmation email")
		return errors.New("error sending confirmation email")
	}

	log.WithFields(logrus.Fields{
		"action": "create_user",
		"user":   newUser.Username,
	}).Info("User created successfully")

	return nil
}

func (userService) Activate(db *sql.DB, link string) error {

	var count int
	fmt.Println("Activate link:" + link)

	// Prepare the SQL query with placeholders for parameters
	query := "SELECT COUNT(*) FROM user_table WHERE confirmation = $1"

	// Execute the SQL query with the link parameter
	err := db.QueryRow(query, link).Scan(&count)
	if err != nil {
		return fmt.Errorf("error checking link existence: %s", err)
	}
	fmt.Println("Count:", count)

	if count == 0 {
		return errors.New("link not found")
	}

	// Prepare the SQL query for updating isActivated
	updateQuery := "UPDATE user_table SET isactivated = true WHERE confirmation = $1"

	// Execute the SQL update query with the link parameter
	_, err = db.Exec(updateQuery, link)
	if err != nil {
		log.Printf("Error updating user: %v\n", err)
		return fmt.Errorf("error updating user: %s", err)
	}

	return nil
}

func (userService) AuthenticateUser(db *sql.DB, username string, password string) error {
	var storedPasswordHash string
	err := db.QueryRow("SELECT password FROM "+tableName+" WHERE username = $1", username).Scan(&storedPasswordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			// Username not found
			log.WithError(err).Warn("User not found")
			return errors.New("user not found")
		}
		log.WithError(err).Error("Error retrieving user password hash from database")
		return fmt.Errorf("error retrieving user password hash: %s", err)
	}

	// Compare the stored password hash with the provided password
	err = bcrypt.CompareHashAndPassword([]byte(storedPasswordHash), []byte(password))
	if err != nil {
		// Passwords don't match
		return errors.New("incorrect password")
	}

	return nil
}

func (userService) ShowUserList(w http.ResponseWriter, r *http.Request, db *sql.DB) error {
	users, err := getUserListDB(db)

	if err != nil {
		log.WithError(err).Error("Error getting user list from database")
		return errors.New("cant get userlist")
	}
	ts, err := template.ParseFiles("userList.html")
	if err != nil {
		log.WithError(err).Error("Error parsing user list template")
		return err
	}
	log.Info("User list displayed successfully")
	ts.Execute(w, users)

	return nil
}

func getUserListDB(db *sql.DB) ([]authUser, error) {
	rows, err := db.Query(`SELECT "username","password" FROM user_table`)
	if err != nil {
		return nil, fmt.Errorf("error querying database: %s", err)
	}
	defer rows.Close()

	var users []authUser

	for rows.Next() {
		var u authUser
		err := rows.Scan(&u.Username, &u.PasswordHash)
		if err != nil {
			return nil, fmt.Errorf("error scanning row: %s", err)
		}
		users = append(users, u)
	}
	return users, nil
}

func checkUsername(db *sql.DB, username string) error {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM " + tableName + " WHERE username = '" + username + "'").Scan(&count)
	if err != nil {
		log.WithError(err).Error("Error checking username uniqueness")
		return fmt.Errorf("error checking username uniqueness: %s", err)
	}

	if count > 0 {
		log.Warn("User already exists")
		return errors.New("user already exists")
	}

	return nil
}

func insertUserDB(db *sql.DB, data authUser) error {
	_, err := db.Exec("INSERT INTO "+tableName+" (email, username, password, confirmation) VALUES ($1, $2, $3, $4)",
		data.Email, data.Username, data.PasswordHash, data.Confirmation)
	if err != nil {
		log.WithError(err).Error("Error inserting user into database")
		return fmt.Errorf("error inserting user into database: %s", err)
	}

	log.WithFields(logrus.Fields{
		"action": "insert_user",
		"user":   data.Username,
	}).Info("User inserted into database successfully")

	return nil
}
