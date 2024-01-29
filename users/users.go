package users

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"text/template"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"

	"golang.org/x/crypto/bcrypt"
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
}

var DefaultUserService userService
var log = logrus.New()

type userService struct {
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

	newAuthUser := authUser{
		Email:        newUser.Email,
		Username:     newUser.Username,
		PasswordHash: passwordHash,
	}

	err = insertUserDB(db, newAuthUser)
	if err != nil {
		log.WithError(err).Error("Error inserting user into database")
		return err
	}

	log.WithFields(logrus.Fields{
		"action": "create_user",
		"user":   newUser.Username,
	}).Info("User created successfully")

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
	_, err := db.Exec("INSERT INTO "+tableName+" (email, username, password) VALUES ($1, $2, $3)",
		data.Email, data.Username, data.PasswordHash)
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
