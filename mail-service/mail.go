package mail

import (
	"bytes"
	"fmt"
	"net/smtp"
	"text/template"
)

func SendConfirmationEmail(email string, link string) error {
	// Sender data.
	from := "librabook12@gmail.com"
	password := "tpua wetq aqkp ossq"

	// smtp server configuration.
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

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
