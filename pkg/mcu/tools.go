package mcu

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/normen/obs-mcu/gomcu"
)

func ShortenText(input string) string {
	re := regexp.MustCompile(`([^-_ ]+)[AEIOUaeiou]([^-_ ]+)`)

	input = strings.ReplaceAll(input, "Input", "In")
	input = strings.ReplaceAll(input, "Output", "Out")

	ret := re.FindAllString(input, 1)
	length := utf8.RuneCountInString(input)
	for length > 6 && ret != nil {
		input = re.ReplaceAllString(input, `$1$2`)
		ret = re.FindAllString(input, 1)
		length = utf8.RuneCountInString(input)
	}
	if length > 6 {
		input = strings.ReplaceAll(input, " ", "")
		input = strings.ReplaceAll(input, "-", "")
		input = strings.ReplaceAll(input, "_", "")
		length = utf8.RuneCountInString(input)
	}
	if length > 6 {
		if match, _ := regexp.MatchString(".*[0-9][0-9][/-_][0-9][0-9]$", input); match {
			input = input[:3] + input[length-3:]
		} else if match, _ := regexp.MatchString(".*[0-9][/-_][0-9][0-9]$", input); match {
			input = input[:3] + input[length-3:]
		} else if match, _ := regexp.MatchString(".*[0-9][/-_][0-9]$", input); match {
			input = input[:4] + input[length-2:]
		} else if match, _ := regexp.MatchString(".*[0-9][0-9]$", input); match {
			input = input[:4] + input[length-2:]
		} else if match, _ := regexp.MatchString(".*[0-9]$", input); match {
			input = input[:5] + input[length-1:]
		}
		length = utf8.RuneCountInString(input)
	}
	if length < 6 {
		input = fmt.Sprintf("%-6s", input)
	} else if length > 6 {
		input = input[0:6]
	}
	return input
}

func Db2MeterLevel(valueDB float64) gomcu.MeterLevel {
	if valueDB >= 0 {
		return gomcu.MoreThan0
	} else if valueDB > -2 {
		return gomcu.MoreThan2
	} else if valueDB > -4 {
		return gomcu.MoreThan4
	} else if valueDB > -6 {
		return gomcu.MoreThan6
	} else if valueDB > -8 {
		return (gomcu.MoreThan8)
	} else if valueDB > -10 {
		return (gomcu.MoreThan10)
	} else if valueDB > -14 {
		return (gomcu.MoreThan14)
	} else if valueDB > -20 {
		return (gomcu.MoreThan20)
	} else if valueDB > -30 {
		return (gomcu.MoreThan30)
	} else if valueDB > -40 {
		return (gomcu.MoreThan40)
	} else if valueDB > -50 {
		return (gomcu.MoreThan50)
	} else if valueDB > -60 {
		return (gomcu.MoreThan60)
	}

	return (gomcu.LessThan60)
}

func Bool2State(b bool) gomcu.State {
	if b {
		return gomcu.StateOn
	}
	return gomcu.StateOff
}

func InvertState(s gomcu.State) gomcu.State {
	if s == gomcu.StateOff {
		return gomcu.StateOn
	}
	// Led Blink --> Led Off
	return gomcu.StateOff
}
