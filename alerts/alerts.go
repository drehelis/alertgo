package alerts

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"strings"

	"alertgo/config"
	"alertgo/maps"
	"alertgo/telegram"
	"alertgo/types"
)

var userAgents = []string {
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36 Edg/130.0.0.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:131.0) Gecko/20100101 Firefox/131.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0.1 Safari/605.1.15",
}

func FetchAlerts(client *http.Client, initCfg config.InitConfig) ([]types.ThreatAlert, error) {
    req, err := http.NewRequest(http.MethodGet, initCfg.AlertsEndpoint, nil)
    if err != nil {
        return nil, err
    }

    randomUserAgent := userAgents[rand.Intn(len(userAgents))]
    req.Header.Set("User-Agent", randomUserAgent)

    res, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer res.Body.Close()

    body, err := io.ReadAll(res.Body)
    if err != nil {
        return nil, err
    }

    // Stupid empty json utf8 byte order mark removal
    cleanBody := strings.TrimSpace(strings.Replace(string(body), "\ufeff", "", -1))
    if cleanBody == "" || cleanBody == "\r\n" {
        return []types.ThreatAlert{}, nil
    }

    // log.Printf("Raw data: %q\n", cleanBody)
    var alert types.ThreatAlert
    if err := json.Unmarshal([]byte(cleanBody), &alert); err != nil {
        var alerts []types.ThreatAlert
        if err := json.Unmarshal([]byte(cleanBody), &alerts); err != nil {
            return nil, fmt.Errorf("unable to parse value: %q, error: %s", string(body), err.Error())
        }
        return alerts, nil
    }

    return []types.ThreatAlert{alert}, nil
}

func getAlertEmoji(title string) string {
    switch title {
    case "ירי רקטות וטילים":
        return "🚀"
	case "חדירת כלי טיס עוין":
		return "✈︎"
    default:
        return "🚨"
    }
}

func formatAlertMessage(emoji, title string, locations []string, desc string) string {
    return fmt.Sprintf(
        "🔴 " + 
        "התרעה מסוג <b>%s</b> " +
        "%s\n" +
        "באזור: %s\n" +
        "%s",
        title,
        emoji,
        strings.Join(locations, ", "),
        desc,
    )
}

func ProcessAlerts(alerts []types.ThreatAlert, seenAlerts map[string]*types.MessageState, initCfg config.InitConfig) {
    if len(alerts) == 0 {
        log.Println("No alerts to process")
        return
    }

    sort.Slice(alerts, func(i, j int) bool {
        return alerts[i].ID > alerts[j].ID
    })

    latestAlert := alerts[0]
    log.Printf("Processing latest alert ID: %s with locations: %v", latestAlert.ID, latestAlert.Data)

    // Find existing state
    var existingState *types.MessageState
    var existingMessageID string

    for id, state := range seenAlerts {
        if isLocationSubset(state.Locations, latestAlert.Data) || 
           isLocationSubset(latestAlert.Data, state.Locations) {
            existingState = state
            existingMessageID = state.MessageID
            log.Printf("Found existing alert with ID: %s matching location criteria", id)
            break
        }
    }

    if existingState != nil {
        // Merge new locations with existing ones
        allLocations := existingState.Locations
        for _, loc := range latestAlert.Data {
            found := false
            for _, existingLoc := range allLocations {
                if existingLoc == loc {
                    found = true
                    break
                }
            }
            if !found {
                allLocations = append(allLocations, loc)
            }
        }

        // Generate message and map with all locations
        emoji := getAlertEmoji(latestAlert.Title)
        newMessage := formatAlertMessage(emoji, latestAlert.Title, allLocations, latestAlert.Desc)
        newMapURL := maps.GenerateMapURL(allLocations, initCfg.GoogleMapsAPIKey)
        
        mapChanged := existingState.MapURL != newMapURL
        contentChanged := existingState.Content != newMessage
        
        if !mapChanged && !contentChanged {
            log.Printf("No changes detected - skipping update")
            return
        }

        if mapChanged {
            // Update map with all locations
            log.Printf("Updating map with all locations")
            err := telegram.EditTelegramMessageMedia(existingMessageID, newMapURL, newMessage, initCfg)
            if err != nil {
                log.Printf("Failed to update map: %v", err)
                return
            }
        } else if contentChanged {
            // Update caption
            err := telegram.EditTelegramMessageWithPhoto(existingMessageID, newMessage, initCfg)
            if err != nil {
                log.Printf("Failed to update caption: %v", err)
                return
            }
        }

        // Update state keeping all locations
        delete(seenAlerts, existingState.ID)
        existingState.ID = latestAlert.ID
        existingState.Locations = allLocations
        existingState.Content = newMessage
        existingState.MapURL = newMapURL
        seenAlerts[latestAlert.ID] = existingState
    } else {
        // Create new message
        emoji := getAlertEmoji(latestAlert.Title)
        newMessage := formatAlertMessage(emoji, latestAlert.Title, latestAlert.Data, latestAlert.Desc)
        newMapURL := maps.GenerateMapURL(latestAlert.Data, initCfg.GoogleMapsAPIKey)

        log.Printf("Creating new message with map")
        messageID, err := telegram.SendTelegramMessageWithPhoto(newMessage, newMapURL, initCfg)
        if err != nil {
            log.Printf("Failed to send new message: %v", err)
            return
        }

        seenAlerts[latestAlert.ID] = &types.MessageState{
            ID:        latestAlert.ID,
            MessageID: messageID,
            Locations: latestAlert.Data,
            Content:   newMessage,
            HasPhoto:  true,
            MapURL:    newMapURL,
        }
    }
}

func isLocationSubset(locations1, locations2 []string) bool {
    // Create map of locations2 for O(1) lookup
    loc2Map := make(map[string]bool)
    for _, loc := range locations2 {
        loc2Map[loc] = true
    }

    // Check if each location in locations1 exists in locations2
    for _, loc := range locations1 {
        if !loc2Map[loc] {
            return false
        }
    }
    
    return true
}