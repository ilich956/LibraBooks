package users

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"text/template"

	_ "github.com/lib/pq"

	"golang.org/x/crypto/bcrypt"
)

const (
	port        = ":8080"
	connStr     = "postgres://postgres:bayipket@localhost/adv_database?sslmode=disable"
	driverName  = "postgres"
	tableName   = "user_table"
	createTable = `
		CREATE TABLE IF NOT EXISTS user_table (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255),
			username VARCHAR(255),
			password VARCHAR(255)
		);
	`
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

type userService struct {
}

func getPasswordHash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 0)
	return string(hash), err
}

func (userService) CreateUser(db *sql.DB, newUser User) error {
	err := checkUsername(db, newUser.Username)

	if err != nil {
		fmt.Println("user already exists")
		return errors.New("user already exists")
	}

	passwordHash, err := getPasswordHash(newUser.Password)
	if err != nil {
		return err
	}

	newAuthUser := authUser{
		Email:        newUser.Email,
		Username:     newUser.Username,
		PasswordHash: passwordHash,
	}

	err = insertUserDB(db, newAuthUser)
	if err != nil {
		return err
	}

	return nil
}

func (userService) ShowUserList(w http.ResponseWriter, r *http.Request, db *sql.DB) error {
	users, err := getUserListDB(db)

	if err != nil {
		fmt.Println("cant get userlist")
		return errors.New("cant get userlist")
	}
	ts, err := template.ParseFiles("userList.html")
	if err != nil {
		return err
	}
	fmt.Println(users[0])
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
		return fmt.Errorf("error checking username uniqueness: %s", err)
	}

	if count > 0 {
		return errors.New("user already exists")
	}

	return nil
}

func insertUserDB(db *sql.DB, data authUser) error {
	_, err := db.Exec("INSERT INTO "+tableName+" (email, username, password) VALUES ($1, $2, $3)",
		data.Email, data.Username, data.PasswordHash)
	if err != nil {
		return fmt.Errorf("error inserting user into database: %s", err)
	}

	return nil
}
