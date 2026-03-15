package resume

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

func ParseExtraArgs(input string) ([]string, error) {
	var args []string
	var builder strings.Builder

	inSingleQuote := false
	inDoubleQuote := false
	escaped := false
	tokenActive := false

	flush := func() {
		if !tokenActive {
			return
		}
		args = append(args, builder.String())
		builder.Reset()
		tokenActive = false
	}

	for _, char := range input {
		switch {
		case escaped:
			builder.WriteRune(char)
			tokenActive = true
			escaped = false
		case inSingleQuote:
			if char == '\'' {
				inSingleQuote = false
				continue
			}
			builder.WriteRune(char)
			tokenActive = true
		case inDoubleQuote:
			switch char {
			case '"':
				inDoubleQuote = false
			case '\\':
				escaped = true
				tokenActive = true
			default:
				builder.WriteRune(char)
				tokenActive = true
			}
		default:
			switch {
			case unicode.IsSpace(char):
				flush()
			case char == '\'':
				inSingleQuote = true
				tokenActive = true
			case char == '"':
				inDoubleQuote = true
				tokenActive = true
			case char == '\\':
				escaped = true
				tokenActive = true
			default:
				builder.WriteRune(char)
				tokenActive = true
			}
		}
	}

	if escaped || inSingleQuote || inDoubleQuote {
		return nil, errors.New("unterminated quoted argument")
	}

	flush()
	return args, nil
}

func ValidateExtraArgs(args []string) error {
	for _, arg := range args {
		if arg == "resume" || arg == "--resume" || strings.HasPrefix(arg, "--resume=") {
			return fmt.Errorf("resume arguments are not allowed: %s", arg)
		}
	}
	return nil
}
