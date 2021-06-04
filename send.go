package suckmail

import (
	"errors"
	"net"
	"net/smtp"
	"strings"
)

func Send(conn net.Conn, host, username, password string, message *MailMessage) error {
	if len(message.errors) > 0 {
		str := strings.Builder{}
		for _, err := range message.errors {
			str.WriteString(err.Error() + "\r\n")
		}
		return errors.New("Errors on build email message:\r\n" + str.String())
	}
	rawmessagedata, err := message.build()
	if err != nil {
		return err
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return nil
	}
	defer client.Close()

	auth := smtp.PlainAuth("", username, password, host)
	if err = client.Auth(auth); err != nil {
		return err
	}

	// To && From
	if err = client.Mail(message.fromEmail); err != nil {
		return err
	}

	if err = client.Rcpt(message.recieverEmail); err != nil {
		return err
	}

	// Data
	w, err := client.Data()
	if err != nil {
		return err
	}

	_, err = w.Write(rawmessagedata)
	if err != nil {
		return err
	}

	if err = w.Close(); err != nil {
		return err
	}

	return client.Quit()
}
