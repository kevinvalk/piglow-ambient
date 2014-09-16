package main

import (
	"github.com/cpucycle/astrotime"
	"github.com/wjessop/go-piglow"
	"code.google.com/p/gcfg"
	"time"
	"strconv"
	"math"
	"log"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"fmt"
	"flag"
)
const VERSION = "0.2.1"

var isRunning bool
var isTesting bool
var pidPath string
var logPath string
var cfgPath string
var cfg Config

func initFlags(){
	// Adjust command line help text
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "PiGlow Ambient, version %s\n", VERSION)
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n  -h,--help: this help\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n")
	}

	// Command line arguments
	flag.BoolVar(&isTesting, "test", false, "if enabled it tests the program and dies")
	flag.StringVar(&pidPath, "pidfile", "", "name of the PID file")
	flag.StringVar(&logPath, "logfile", "-", "log to a specified file, - for stdout")
	flag.StringVar(&cfgPath, "cfgfile", "/etc/piglow-ambient.gcfg", "configuration file")
	flag.Parse()
}

func initSignal() {
	ChannelInterrupt := make(chan os.Signal, 1)
	signal.Notify(ChannelInterrupt, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)

	go func(){
		<- ChannelInterrupt
		log.Printf("Goodbye!")
		isRunning = false
	}()

	ChannelReload := make(chan os.Signal, 1)
	signal.Notify(ChannelReload, syscall.SIGHUP)

	go func(){
		for isRunning {
			<- ChannelReload
			log.Printf("[TODO] We should reload config file!")
		}
	}()
}


func main() {
	// Do initializing
	isRunning = true
	initFlags()
	initSignal()

	// Setup logging
	if logPath != "-" {
		logFile, err := os.OpenFile(logPath, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0640)
		if err != nil {
			log.Fatalf("error opening file: %v", err)
		}
		defer logFile.Close()
		log.SetOutput(logFile)
	}

	if logPath != "-" {
		log.Printf("--------------------------------------------------------")
	}
	log.Printf("Welcome to PiGlow Ambient version %s", VERSION)

	// Write pid file
	if pidPath != "" {
		if err := ioutil.WriteFile(pidPath, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
			log.Fatalf("error creating PID file: %v", err)
		}
		defer os.Remove(pidPath) // Remove when we exit
	}

	// Read configuration file
	err := gcfg.ReadFileInto(&cfg, cfgPath)
	if err != nil {
		log.Fatalf("Failed to parse gcfg data: %s", err)
	}

	// Initialize transition speed
	transitionTime, err := getTransitionSpeed(cfg.Settings.TransitionSpeed)
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
	fadeInTime := astrotime.NextSunset(time.Now(), cfg.Settings.Latitude, -cfg.Settings.Longitude).Add(-transitionDuration/2)
	fadeOutTime := astrotime.NextSunrise(time.Now(), cfg.Settings.Latitude, -cfg.Settings.Longitude).Add(-transitionDuration/2)

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
	log.Printf("Transition time in seconds: %d, Sleep duration: %.04f", transitionTime, sleepDuration.Seconds())
	log.Printf("Latitude: %f, Longitude: %f", cfg.Settings.Latitude, cfg.Settings.Longitude)
	log.Printf("The next fadeIn  is %02d:%02d:%02d on %d/%d/%d", fadeInTime.Hour(), fadeInTime.Minute(), fadeInTime.Second(), fadeInTime.Month(), fadeInTime.Day(), fadeInTime.Year())
	log.Printf("The next fadeOut is %02d:%02d:%02d on %d/%d/%d", fadeOutTime.Hour(), fadeOutTime.Minute(), fadeOutTime.Second(), fadeOutTime.Month(), fadeOutTime.Day(), fadeOutTime.Year())

	// isTesting
	if isTesting {
		os.Exit(0)
	}

	// Main loop
	var power int
	for isRunning {
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
				fadeInTime = astrotime.NextSunset(time.Now(), cfg.Settings.Latitude, -cfg.Settings.Longitude).Add(-transitionDuration/2)
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
				fadeOutTime = astrotime.NextSunrise(time.Now(), cfg.Settings.Latitude, -cfg.Settings.Longitude).Add(-transitionDuration/2)
				log.Printf("The next fadeOut is %02d:%02d:%02d on %d/%d/%d", fadeOutTime.Hour(), fadeOutTime.Minute(), fadeOutTime.Second(), fadeOutTime.Month(), fadeOutTime.Day(), fadeOutTime.Year())
			}
		}

		// Sleep
		time.Sleep(sleepDuration)
	}
}
