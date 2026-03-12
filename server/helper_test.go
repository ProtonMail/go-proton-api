package server

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func newMessageLiteral(from, to string) []byte {
	return fmt.Appendf(nil, "From: %v\r\nReceiver: %v\r\nSubject: %v\r\n\r\nHello World!", from, to, uuid.New())
}

func newMessageLiteralWithSubject(from, to, subject string) []byte {
	return fmt.Appendf(nil, "From: %v\r\nReceiver: %v\r\nSubject: %v\r\n\r\nHello World!", from, to, subject)
}

func newMessageLiteralWithSubjectAndSize(from, to, subject string, paddingSize int) []byte {
	padding := strings.Repeat("A", paddingSize)
	return fmt.Appendf(nil, "From: %v\r\nReceiver: %v\r\nSubject: %v\r\n\r\nHello World!Padding:%s", from, to, subject, padding)
}
