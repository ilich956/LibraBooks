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
)

var (
	port       = goDotEnvVariable("PORT")
	connStr    = goDotEnvVariable("CONN_STR")
	driverName = goDotEnvVariable("DRIVERNAME")
	tableName  = goDotEnvVariable("TABLENAME")
)

type DisplayUser struct {
	ID          int
	Email       string
	Username    string
	IsActivated bool
	IsAdmin     bool
}

type User struct {
	Email    string
	Username string
	Password string
}

type authUser struct {
	ID           int
	Email        string
	Username     string
	PasswordHash string
	Confirmation string
	Token        string
	IsActivated  bool
	IsAdmin      bool
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

func (userService) CreateUser(db *sql.DB, newUser User, token string) error {
	err := checkUsername(db, newUser.Email)
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
		Token:        token,
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

	// err = mail.SendConfirmationEmail(newAuthUser.Email, fullLink)
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

func (userService) DeleteUser(db *sql.DB, userID string) error {
	_, err := db.Exec("DELETE FROM user_table WHERE id = $1", userID)
	if err != nil {
		// Handle the error
		return fmt.Errorf("error deleting user: %s", err)
	}

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

func SaveToken(db *sql.DB, email string, token string) error {
	_, err := db.Exec("UPDATE "+tableName+" SET token = $1 WHERE email = $2", token, email)
	if err != nil {
		log.WithError(err).Error("Error saving token in database")
		return fmt.Errorf("error saving token in database: %s", err)
	}
	return nil
}

func (userService) AuthenticateUser(db *sql.DB, username string, password string, token string) error {
	var storedPasswordHash string
	var storedOTP *string
	err := db.QueryRow("SELECT password, otp FROM "+tableName+" WHERE email = $1", username).Scan(&storedPasswordHash, &storedOTP) //
	if err != nil {
		if err == sql.ErrNoRows {
			log.WithError(err).Warn("User not found")
			return errors.New("user not found")
		}
		log.WithError(err).Error("Error retrieving user password hash from database")
		return fmt.Errorf("error retrieving user password hash: %s", err)
	}

	if storedOTP != nil && password == *storedOTP {
		// Clear OTP from database
		_, err := db.Exec("UPDATE "+tableName+" SET otp = NULL WHERE email = $1", username)
		if err != nil {
			log.WithError(err).Error("Error clearing OTP in database")
			return fmt.Errorf("error clearing OTP in database: %s", err)
		}
		return nil // Successfully authenticated with OTP
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedPasswordHash), []byte(password))
	if err != nil {
		return errors.New("incorrect password")
	}

	if storedOTP != nil {
		_, err = db.Exec("UPDATE "+tableName+" SET otp = NULL WHERE email = $1", username)
		if err != nil {
			log.WithError(err).Error("Error clearing OTP in database")
			return fmt.Errorf("error clearing OTP in database: %s", err)
		}
	}

	err = SaveToken(db, username, token)
	if err != nil {
		// Handle error
		log.WithError(err).Error("Error saving token")
		return err
	}

	return nil
}

func (userService) ChangePassword(db *sql.DB, email string, password string, newpassword string) error {
	var storedPasswordHash string
	err := db.QueryRow("SELECT password FROM "+tableName+" WHERE email = $1", email).Scan(&storedPasswordHash)
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

	// Hash the new password
	newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(newpassword), bcrypt.DefaultCost)
	if err != nil {
		log.WithError(err).Error("Error hashing new password")
		return fmt.Errorf("error hashing new password: %s", err)
	}

	// Update the password in the database
	_, err = db.Exec("UPDATE "+tableName+" SET password = $1 WHERE email = $2", newPasswordHash, email)
	if err != nil {
		log.WithError(err).Error("Error updating password in database")
		return fmt.Errorf("error updating password in database: %s", err)
	}

	return nil
}

func (userService) OTPservice(db *sql.DB, email string) error {
	otp := uuid.New()

	_, err := db.Exec("UPDATE "+tableName+" SET otp = $1 WHERE email = $2", otp, email)
	if err != nil {
		log.WithError(err).Error("Error updating password in database")
		return fmt.Errorf("error updating password in database: %s", err)
	}

	// err = mail.SendOTPEmail(email, otp.String())
	if err != nil {
		log.WithError(err).Error("error sending confirmation email")
		return errors.New("error sending confirmation email")
	}

	return nil
}

func (userService) ShowUserList(w http.ResponseWriter, r *http.Request, db *sql.DB) error {
	cookie, err := r.Cookie("token")
	if err != nil {
		return errors.New("token not found in cookies")
	}
	token := cookie.Value

	var isAdmin bool
	err = db.QueryRow("SELECT isadmin FROM user_table WHERE token = $1", token).Scan(&isAdmin)
	if err != nil {
		return errors.New("error checking user admin status")
	}

	if !isAdmin {
		http.Error(w, "Access denied: Only admins can view user list", http.StatusUnauthorized)
		return nil
	}

	users, err := getUserListDB(db)
	if err != nil {
		log.WithError(err).Error("Error getting user list from database")
		return errors.New("failed to retrieve user list from the database")
	}

	ts, err := template.ParseFiles("userList.html")
	if err != nil {
		log.WithError(err).Error("Error parsing user list template")
		return errors.New("failed to parse HTML template")
	}

	err = ts.Execute(w, users)
	if err != nil {
		log.WithError(err).Error("Error executing HTML template")
		return errors.New("failed to render user list template")
	}

	log.Info("User list displayed successfully")
	return nil
}

func getUserListDB(db *sql.DB) ([]authUser, error) {
	// rows, err := db.Query(`SELECT id, email, username, isactivated, isadmin FROM user_table`)
	rows, err := db.Query(`SELECT id, email, isactivated, isadmin FROM user_table LIMIT 10000`)

	if err != nil {
		return nil, fmt.Errorf("error querying database: %s", err)
	}
	defer rows.Close()

	var users []authUser

	for rows.Next() {
		var u authUser
		// err := rows.Scan(&u.ID, &u.Email, &u.Username, &u.IsActivated, &u.IsAdmin)
		err := rows.Scan(&u.ID, &u.Email, &u.IsActivated, &u.IsAdmin)

		if err != nil {
			return nil, fmt.Errorf("error scanning row: %s", err)
		}
		users = append(users, u)
	}
	return users, nil
}

func checkUsername(db *sql.DB, email string) error {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM " + tableName + " WHERE email = '" + email + "'").Scan(&count)
	if err != nil {
		log.WithError(err).Error("Error checking email uniqueness")
		return fmt.Errorf("error checking email uniqueness: %s", err)
	}

	if count > 0 {
		log.Warn("User already exists")
		return errors.New("user already exists")
	}

	return nil
}

func insertUserDB(db *sql.DB, data authUser) error {
	_, err := db.Exec("INSERT INTO "+tableName+" (email, username, password, confirmation, token) VALUES ($1, $2, $3, $4, $5)",
		data.Email, data.Username, data.PasswordHash, data.Confirmation, data.Token)
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
