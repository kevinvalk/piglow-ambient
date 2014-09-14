package main

import (
	"github.com/cpucycle/astrotime"
	"github.com/wjessop/go-piglow"
	"time"
	"strings"
	"strconv"
	"errors"
	"unicode"
	"math"
	"log"
	"io/ioutil"
	"os"
	"fmt"
	"flag"
)

const HELP_HEADER = "PiGlow Ambient, version 0.1.0\n"
const LATITUDE = float64(51.82796)
const LONGITUDE = float64(5.86830)
const TRANSITION_SPEED = "60m"
const MAX_POWER = 255

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

var pidPath string
var logPath string

func main() {
	// Adjust command line help text
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, HELP_HEADER)
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n  -h,--help: this help\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n")
	}

	// Command line arguments
	flag.StringVar(&pidPath, "pidfile", "", "name of the PID file")
	flag.StringVar(&logPath, "logfile", "-", "log to a specified file, - for stdout")
	flag.Parse()

	// Write pid file
	if pidPath != "" {
		if err := ioutil.WriteFile(pidPath, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
			log.Fatalf("error creating PID file: %v", err)
		}
		defer os.Remove(pidPath) // Remove when we exit
	}

	// Setup logging
	if logPath != "-" {
		logFile, err := os.OpenFile(logPath, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0640)
		if err != nil {
			log.Fatalf("error opening file: %v", err)
		}
		defer logFile.Close()
		log.SetOutput(logFile)
	}

	// Initialize transition speed
	transitionTime, err := getTransitionSpeed(TRANSITION_SPEED)
	if err != nil {
		log.Fatal(err)
	}
	if transitionTime <= 0 {
		log.Fatal("Need to have a transition period that is greater then zero!")
	}

	// Do the initial calculations
	transitionDuration := time.Duration(transitionTime) * time.Second
	sleepDuration := time.Duration((float64(transitionTime)/float64(MAX_POWER)*0.9) * 1000000000)  // Dynamic calculate sleep time to optimize CPU usage while maintaining smooth transitions when the transition period is very small
	if sleepDuration > time.Second {
		sleepDuration = time.Second
	}
	fadeInTime := astrotime.NextSunset(time.Now(), LATITUDE, LONGITUDE).Add(-transitionDuration/2)
	fadeOutTime := astrotime.NextSunrise(time.Now(), LATITUDE, LONGITUDE).Add(-transitionDuration/2)

	// Setup PiGlow
	p, err := piglow.NewPiglow()
	if err != nil {
		log.Fatal("Could not create a PiGlow object: ", err)
	}
	p.SetAll(0)
	if err = p.Apply(); err != nil {
		log.Fatal("Could not set PiGlow: ", err)
	}

	// Announce some basic information
	log.Printf("Transition time in seconds: %d, Sleep duration: %f", transitionTime, sleepDuration.Seconds())
	log.Printf("The next fadeIn  is %02d:%02d:%02d on %d/%d/%d", fadeInTime.Hour(), fadeInTime.Minute(), fadeInTime.Second(), fadeInTime.Month(), fadeInTime.Day(), fadeInTime.Year())
	log.Printf("The next fadeOut is %02d:%02d:%02d on %d/%d/%d", fadeOutTime.Hour(), fadeOutTime.Minute(), fadeOutTime.Second(), fadeOutTime.Month(), fadeOutTime.Day(), fadeOutTime.Year())

	var power int
	for {
		// FadeIn
		if elapsed := time.Now().Sub(fadeInTime); elapsed > 0 {
			// Calculate brightness with maximum of 255
			power = int(math.Ceil((MAX_POWER/float64(transitionTime))*elapsed.Seconds())) % 256

			// Set the new brightness
			p.SetAll(uint8(power))
			if err = p.Apply(); err != nil {
				log.Fatal("Could not set PiGlow: ", err)
			}

			// If we have complete our fadeIn calculate next fadeIn
			if power >= 255 {
				fadeInTime = astrotime.NextSunset(time.Now(), LATITUDE, LONGITUDE).Add(-transitionDuration/2)
				log.Printf("The next fadeIn  is %02d:%02d:%02d on %d/%d/%d", fadeInTime.Hour(), fadeInTime.Minute(), fadeInTime.Second(), fadeInTime.Month(), fadeInTime.Day(), fadeInTime.Year())
			}
		}

		// FadeOut
		if elapsed := time.Now().Sub(fadeOutTime); elapsed > 0 {
			// Calculate brightness with minimum of zero
			power = 255-int(math.Floor((MAX_POWER/float64(transitionTime))*elapsed.Seconds()))
			if power < 0 {
				power = 0
			}

			// Set the new brightness
			p.SetAll(uint8(power))
			if err = p.Apply(); err != nil {
				log.Fatal("Could not set PiGlow: ", err)
			}

			// If we have complete our fadeIn calculate next fadeIn
			if power <= 0 {
				fadeOutTime = astrotime.NextSunrise(time.Now(), LATITUDE, LONGITUDE).Add(-transitionDuration/2)
				log.Printf("The next fadeOut is %02d:%02d:%02d on %d/%d/%d", fadeOutTime.Hour(), fadeOutTime.Minute(), fadeOutTime.Second(), fadeOutTime.Month(), fadeOutTime.Day(), fadeOutTime.Year())
			}
		}

		// Sleep
		time.Sleep(sleepDuration)
	}
}
