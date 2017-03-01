package email

import "fmt"

func GetEmailActivationMessage(locale string, url string) *Message {
	return &Message{
		Subject: "Confirm your registration",
		Body:    fmt.Sprintf("Thank you for creating an account!<br/>Please confirm your email address by clicking on the link below:<br/><br/><a href=\"%s\">Click here to confirm</a>", url),
	}
}

func GetEmailChangeEmailAddressMessage(locale string, url string) *Message {
	return &Message{
		Subject: "Confirm your email address",
		Body:    fmt.Sprintf("You've requested an email change.<br/>Please confirm your email address by clicking on the link below:<br/><br/><a href=\"%s\">Click here to confirm</a>", url),
	}
}

func GetEmailChangePasswordMessage(locale string, url string) *Message {
	return &Message{
		Subject: "Password Change",
		Body:    fmt.Sprintf("You've requested to change your password.<br/>Please use the link below:<br/><br/><a href=\"%s\">Click here to confirm</a>", url),
	}
}
