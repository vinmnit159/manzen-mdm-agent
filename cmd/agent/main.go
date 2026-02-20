package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/vinmnit159/manzen-mdm-agent/internal/checkin"
	"github.com/vinmnit159/manzen-mdm-agent/internal/collector"
	"github.com/vinmnit159/manzen-mdm-agent/internal/enroll"
)

const version = "0.1.0"

func main() {
	var (
		enrollCmd    = flag.NewFlagSet("enroll", flag.ExitOnError)
		enrollToken  = enrollCmd.String("token", "", "One-time enrollment token (required)")
		enrollServer = enrollCmd.String("server", "https://ismsbackend.bitcoingames1346.com", "ISMS backend URL")

		daemonCmd      = flag.NewFlagSet("daemon", flag.ExitOnError)
		daemonServer   = daemonCmd.String("server", "https://ismsbackend.bitcoingames1346.com", "ISMS backend URL")
		daemonInterval = daemonCmd.Duration("interval", 15*time.Minute, "Check-in interval (e.g. 15m, 1h)")

		checkinCmd    = flag.NewFlagSet("checkin", flag.ExitOnError)
		checkinServer = checkinCmd.String("server", "https://ismsbackend.bitcoingames1346.com", "ISMS backend URL")
	)

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "version":
		fmt.Printf("manzen-mdm-agent v%s\n", version)

	case "enroll":
		enrollCmd.Parse(os.Args[2:])
		if *enrollToken == "" {
			log.Fatal("--token is required for enrollment")
		}
		if err := enroll.Run(*enrollServer, *enrollToken); err != nil {
			log.Fatalf("Enrollment failed: %v", err)
		}

	case "checkin":
		checkinCmd.Parse(os.Args[2:])
		cfg, err := enroll.LoadConfig()
		if err != nil {
			log.Fatalf("Device not enrolled. Run 'manzen-agent enroll --token <TOKEN>' first.\n%v", err)
		}
		posture, err := collector.Collect()
		if err != nil {
			log.Fatalf("Failed to collect posture: %v", err)
		}
		if err := checkin.Run(*checkinServer, cfg, posture); err != nil {
			log.Fatalf("Check-in failed: %v", err)
		}
		fmt.Println("Check-in successful.")

	case "daemon":
		daemonCmd.Parse(os.Args[2:])
		cfg, err := enroll.LoadConfig()
		if err != nil {
			log.Fatalf("Device not enrolled. Run 'manzen-agent enroll --token <TOKEN>' first.\n%v", err)
		}
		log.Printf("Starting daemon — check-in every %s → %s", *daemonInterval, *daemonServer)
		for {
			posture, err := collector.Collect()
			if err != nil {
				log.Printf("Collect error: %v", err)
			} else if err := checkin.Run(*daemonServer, cfg, posture); err != nil {
				log.Printf("Check-in error: %v", err)
			} else {
				log.Println("Check-in OK")
			}
			time.Sleep(*daemonInterval)
		}

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Manzen MDM Agent

Usage:
  manzen-agent enroll  --token <ONE_TIME_TOKEN> [--server <URL>]
  manzen-agent checkin [--server <URL>]
  manzen-agent daemon  [--server <URL>] [--interval 15m]
  manzen-agent version`)
}
