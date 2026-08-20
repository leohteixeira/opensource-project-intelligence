package database

import (
	"errors"
	"net/url"
	"strings"
)

// redact removes any embedded credential from an error message.
//
// pgx includes the connection string in some failures; that string must never
// reach a log or an API response.
func redact(err error) error {
	if err == nil {
		return nil
	}

	message := err.Error()
	if !strings.Contains(message, "://") {
		return err
	}

	var builder strings.Builder

	for field := range strings.FieldsSeq(message) {
		if builder.Len() > 0 {
			builder.WriteByte(' ')
		}
		builder.WriteString(redactField(field))
	}

	return errors.New(builder.String())
}

func redactField(field string) string {
	if !strings.Contains(field, "://") {
		return field
	}

	parsed, err := url.Parse(field)
	if err != nil || parsed.User == nil {
		return field
	}

	parsed.User = url.User("redacted")

	return parsed.String()
}
