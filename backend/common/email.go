package common

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/smtp"
	"strings"
)

func SendEmail(subject string, receiver string, content string) error {
	encodedSubject := fmt.Sprintf("=?UTF-8?B?%s?=", base64.StdEncoding.EncodeToString([]byte(subject)))
	mail := []byte(fmt.Sprintf("To: %s\r\n"+
		"From: %s<%s>\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n\r\n%s\r\n",
		receiver, GetSystemName(), GetSMTPAccount(), encodedSubject, content))
	auth := smtp.PlainAuth("", GetSMTPAccount(), GetSMTPToken(), GetSMTPServer())
	addr := fmt.Sprintf("%s:%d", GetSMTPServer(), SMTPPort)
	to := strings.Split(receiver, ";")
	var err error
	if SMTPPort == 465 {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         GetSMTPServer(),
		}
		conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", GetSMTPServer(), SMTPPort), tlsConfig)
		if err != nil {
			return err
		}
		client, err := smtp.NewClient(conn, GetSMTPServer())
		if err != nil {
			return err
		}
		defer client.Close()
		if err = client.Auth(auth); err != nil {
			return err
		}
		if err = client.Mail(GetSMTPAccount()); err != nil {
			return err
		}
		receiverEmails := strings.Split(receiver, ";")
		for _, receiver := range receiverEmails {
			if err = client.Rcpt(receiver); err != nil {
				return err
			}
		}
		w, err := client.Data()
		if err != nil {
			return err
		}
		_, err = w.Write(mail)
		if err != nil {
			return err
		}
		err = w.Close()
		if err != nil {
			return err
		}
	} else {
		err = smtp.SendMail(addr, auth, GetSMTPAccount(), to, mail)
	}
	return err
}
