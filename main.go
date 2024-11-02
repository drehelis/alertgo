package main

import (
	"log"
	"net/http"
	"time"

	"alertgo/alerts"
	"alertgo/config"
	"alertgo/types"
)

func monitorAlerts(client *http.Client, initCfg config.InitConfig) chan error {
    errCh := make(chan error)
    ticker := time.NewTicker(initCfg.PollInterval)
    seenAlerts := make(map[string]*types.MessageState)

    go func() {
        // Fetch on start because why wait? :P
        alrt, err := alerts.FetchAlerts(client, initCfg)
        if err != nil {
            errCh <- err
        } else {
            alerts.ProcessAlerts(alrt, seenAlerts, initCfg)
        }

        // Fetch with regular interval
        for range ticker.C {
            alrt, err := alerts.FetchAlerts(client, initCfg)
            if err != nil {
                errCh <- err
                continue
            }
            alerts.ProcessAlerts(alrt, seenAlerts, initCfg)
        }
    }()

    return errCh
}


func main() {
    cfg, err := config.LoadConfig()
    if err != nil {
        panic(err)
    }
	
	httpClient := &http.Client{
		Timeout: time.Second * 10,
	}

	log.Println("Starting alert monitoring with Telegram notifications...")
	log.Printf("Alerts filter applied for %s\n", cfg.TargetLocationFilter)
	log.Printf("Poll interval is every %v\n", cfg.PollInterval)

	// Start monitoring in a goroutine
	run := monitorAlerts(httpClient, *cfg)

	// Handle errors from the monitoring goroutine
	for err := range run {
		log.Printf("Error fetching alerts: %v\n", err)
	}
}

