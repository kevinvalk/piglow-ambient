package main

import (
	"strings"
	"strconv"
	"errors"
	"unicode"
	"fmt"
)

const MAX_POWER = 255

type Config struct {
	Settings struct {
		TransitionSpeed string
		Latitude float64
		Longitude float64
		PingIp string
	}
}

func getTransitionSpeed(str string) (int, error) {
	if len(str) <= 0 {
		return -1, errors.New("No transition time given")
	}

	speed := strings.Replace(strings.ToLower(strings.TrimSpace(str)), " ", "", -1)
	timeType := speed[len(speed)-1:len(speed)]

	if !unicode.IsLetter([]rune(timeType)[0]) {
		timeType = "s"
		speed += "s"
	}

	timeSpeed, e := strconv.Atoi(speed[:len(speed)-1])

	if e != nil {
		return -1, e
	}

	switch timeType {
		case "s":
		case "m":
			timeSpeed *= 60
		case "h":
			timeSpeed *= 60 * 60
		default:
			return -1, fmt.Errorf("Time type `%s` given, but is not supported", timeType)
	}

	return timeSpeed, nil
}
