package mail

import (
	"bytes"
	"database/sql"
	"fmt"
	"net/smtp"
	"os"
	"sync"
	"text/template"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func goDotEnvVariable(key string) string {

	// load .env file
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatal("Error loading .env file", err)

	}

	return os.Getenv(key)
}

func SendEmailAll(db *sql.DB, numGoroutines int) error {
	// Measure start time
	startTime := time.Now()

	// Query emails from the database
	rows, err := db.Query("SELECT email FROM user_table LIMIT 1000")
	if err != nil {
		return err
	}
	defer rows.Close()

	// Store emails
	var emails []string
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			return err
		}
		emails = append(emails, email)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	segmentSize := len(emails) / numGoroutines

	for i := 0; i < numGoroutines; i++ {
		start := i * segmentSize
		end := start + segmentSize
		if i == numGoroutines-1 {
			end = len(emails)
		}

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for j := start; j < end; j++ {
				SendConfirmationEmail(emails[j])
			}
		}(start, end)
	}

	wg.Wait()

	// Measure end time and calculate duration
	duration := time.Since(startTime)
	fmt.Printf("Emails sent in %v\n", duration)

	return nil
}

func SendConfirmationEmail(email string) error {
	// Sender data.
	from := goDotEnvVariable("FROM_MAIL")
	password := goDotEnvVariable("PASSWORD_MAIL")

	smtpHost := goDotEnvVariable("SMTP_HOST")
	smtpPort := goDotEnvVariable("SMTP_PORT")

	// Authentication.
	auth := smtp.PlainAuth("", from, password, smtpHost)

	// Receiver email address.
	to := []string{email}

	t, _ := template.ParseFiles("mail-template.html")

	var body bytes.Buffer

	mimeHeaders := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body.Write([]byte(fmt.Sprintf("Subject: LibraBook \n%s\n\n", mimeHeaders)))

	t.Execute(&body, struct {
		Message string
	}{
		Message: "Hello",
	})

	// Sending email.
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, body.Bytes())
	if err != nil {
		return err
	}
	fmt.Println("Email Sent to:", email)
	return nil
}

// func SendConfirmationEmail(email string, link string) error {
// 	// Sender data.
// 	from := goDotEnvVariable("FROM_MAIL")
// 	password := goDotEnvVariable("PASSWORD_MAIL")

// 	smtpHost := goDotEnvVariable("SMTP_HOST")
// 	smtpPort := goDotEnvVariable("SMTP_PORT")

// 	//message := []byte("This is a test email message.")

// 	// Authentication.
// 	auth := smtp.PlainAuth("", from, password, smtpHost)

// 	// Receiver email address.
// 	to := []string{email}
// 	fmt.Println(to)

// 	t, _ := template.ParseFiles("mail-template.html")

// 	var body bytes.Buffer

// 	mimeHeaders := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
// 	body.Write([]byte(fmt.Sprintf("Subject: LibraBook \n%s\n\n", mimeHeaders)))

// 	t.Execute(&body, struct {
// 		Link string
// 	}{
// 		Link: link,
// 	})

// 	// Sending email.
// 	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, body.Bytes())
// 	if err != nil {
// 		fmt.Println(err)
// 		return err
// 	}
// 	fmt.Println("Email Sent!")
// 	return nil
// }

// func SendEmailAll(db *sql.DB, numGoroutines int) error {
// 	// Measure start time
// 	startTime := time.Now()

// 	// Query emails from the database
// 	rows, err := db.Query("SELECT email, link FROM users_table LIMIT 5")
// 	if err != nil {
// 		return err
// 	}
// 	defer rows.Close()

// 	// Store emails and links
// 	var emails []string
// 	var links []string
// 	for rows.Next() {
// 		var email, link string
// 		if err := rows.Scan(&email, &link); err != nil {
// 			return err
// 		}
// 		emails = append(emails, email)
// 		links = append(links, link)
// 	}
// 	if err := rows.Err(); err != nil {
// 		return err
// 	}

// 	var wg sync.WaitGroup
// 	segmentSize := len(emails) / numGoroutines

// 	for i := 0; i < numGoroutines; i++ {
// 		start := i * segmentSize
// 		end := start + segmentSize
// 		if i == numGoroutines-1 {
// 			end = len(emails)
// 		}

// 		wg.Add(1)
// 		go func(start, end int) {
// 			defer wg.Done()
// 			for j := start; j < end; j++ {
// 				SendConfirmationEmail(emails[j], links[j])
// 			}
// 		}(start, end)
// 	}

// 	wg.Wait()

// 	// Measure end time and calculate duration
// 	duration := time.Since(startTime)
// 	fmt.Printf("Emails sent in %v\n", duration)

// 	return nil
// }

func SendEmail(email, text string) error {
	from := goDotEnvVariable("FROM_MAIL")
	password := goDotEnvVariable("PASSWORD_MAIL")

	smtpHost := goDotEnvVariable("SMTP_HOST")
	smtpPort := goDotEnvVariable("SMTP_PORT")

	auth := smtp.PlainAuth("", from, password, smtpHost)

	to := []string{email}

	body := "To: " + email + "\r\n" +
		"Subject: LibraBook\r\n" +
		"\r\n" +
		text

	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, []byte(body))
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println("Email Sent!")
	return nil
}

// func SendEmailAll(db *sql.DB, numGoroutines int) error {
// 	startTime := time.Now()

// 	// Fetch all user emails from the database
// 	rows, err := db.Query("SELECT id, email FROM user_table LIMIT 5")
// 	if err != nil {
// 		return err
// 	}
// 	defer rows.Close()

// 	// Channel to communicate errors from goroutines
// 	errCh := make(chan error)

// 	// Channel to signal completion of all goroutines
// 	doneCh := make(chan struct{})

// 	// Start goroutines
// 	for i := 0; i < numGoroutines; i++ {
// 		go func() {
// 			for rows.Next() {
// 				var id int
// 				var email string
// 				if err := rows.Scan(&id, &email); err != nil {
// 					errCh <- err
// 					return
// 				}
// 				// Send email in a separate goroutine
// 				go func(email string) {
// 					if err := SendEmail(email, "Your email content here"); err != nil {
// 						errCh <- err
// 						return
// 					}
// 					fmt.Println("Email sent to:", email)
// 				}(email)
// 			}
// 		}()
// 	}

// 	// Wait for all goroutines to finish
// 	go func() {
// 		for i := 0; i < numGoroutines; i++ {
// 			<-doneCh
// 		}
// 		close(errCh)
// 	}()

// 	// Wait for errors or completion
// 	for err := range errCh {
// 		if err != nil {
// 			// Handle or log the error
// 			log.Error("Error sending email:", err)
// 			return err
// 		}
// 	}

// 	duration := time.Since(startTime)
// 	fmt.Println("All emails sent. Time taken:", duration)

// 	return nil
// }

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
