package main

import (
	"fmt"
	"github.com/sendgrid/sendgrid-go"
	"os"
)

func sendVerificationEmail(email, token string) {
	sg := sendgrid.NewSendGridClientWithApiKey(os.Getenv("SENDGRID_API_TOKEN"))
	message := sendgrid.NewMail()
	message.AddTo(email)
	message.SetSubject("Please Verify your Email for EC2 Stopwatch")
	message.SetHTML(fmt.Sprintf("Please click the following link to verify your account. After you verify your email, you will recieve another email containing your Stopwatch API Token.<br><br><a href='%s/verify/%s'>%s/verify/%s</a>", os.Getenv("STOPWATCH_URL"), token, os.Getenv("STOPWATCH_URL"), token))
	message.SetFrom(os.Getenv("EMAIL_FROM_ADDRESS"))
	r := sg.Send(message)
	if r != nil {
		fmt.Printf("Error sending email: '%s'", r)
		return
	}
}

func sendTokenEmail(email, token string) {
	sg := sendgrid.NewSendGridClientWithApiKey(os.Getenv("SENDGRID_API_TOKEN"))
	message := sendgrid.NewMail()
	message.AddTo(email)
	message.SetSubject("Your EC2 Stopwatch API Token")
	message.SetHTML(fmt.Sprintf("Your API Token is %s<br>Keep it secret. Keep it safe.", token))
	message.SetFrom(os.Getenv("EMAIL_FROM_ADDRESS"))
	r := sg.Send(message)
	if r != nil {
		fmt.Printf("Error sending email: '%s'", r)
		return
	}
}
