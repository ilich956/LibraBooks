package mail

import (
	"bytes"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"text/template"

	"github.com/joho/godotenv"
)

func goDotEnvVariable(key string) string {

	// load .env file
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatal("Error loading .env file", err)

	}

	return os.Getenv(key)
}

func SendConfirmationEmail(email string, link string) error {
	// Sender data.
	from := goDotEnvVariable("FROM_MAIL")
	password := goDotEnvVariable("PASSWORD_MAIL")

	smtpHost := goDotEnvVariable("SMTP_HOST")
	smtpPort := goDotEnvVariable("SMTP_PORT")

	//message := []byte("This is a test email message.")

	// Authentication.
	auth := smtp.PlainAuth("", from, password, smtpHost)

	// Receiver email address.
	to := []string{email}
	fmt.Println(to)

	t, _ := template.ParseFiles("mail-template.html")

	var body bytes.Buffer

	mimeHeaders := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body.Write([]byte(fmt.Sprintf("Subject: LibraBook \n%s\n\n", mimeHeaders)))

	t.Execute(&body, struct {
		Link string
	}{
		Link: link,
	})

	// Sending email.
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, body.Bytes())
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println("Email Sent!")
	return nil
}

func SendOTPEmail(email string, otp string) error {
	from := goDotEnvVariable("FROM_MAIL")
	password := goDotEnvVariable("PASSWORD_MAIL")

	smtpHost := goDotEnvVariable("SMTP_HOST")
	smtpPort := goDotEnvVariable("SMTP_PORT")

	//message := []byte("This is a test email message.")

	// Authentication.
	auth := smtp.PlainAuth("", from, password, smtpHost)

	// Receiver email address.
	to := []string{email}
	fmt.Println(to)

	t, _ := template.ParseFiles("otp-template.html")

	var body bytes.Buffer

	mimeHeaders := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body.Write([]byte(fmt.Sprintf("Subject: LibraBook \n%s\n\n", mimeHeaders)))

	t.Execute(&body, struct {
		OTP string
	}{
		OTP: otp,
	})

	// Sending email.
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, body.Bytes())
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println("Email Sent!")
	return nil
}

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
