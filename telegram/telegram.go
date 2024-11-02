package telegram

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"alertgo/config"
)

const (
    cTelegramAPIBase            = "https://api.telegram.org/bot%s"
    cTelegramSendMessage        = cTelegramAPIBase + "/sendMessage"
    cTelegramSendPhoto          = cTelegramAPIBase + "/sendPhoto"
    cTelegramEditMessage        = cTelegramAPIBase + "/editMessageText"
    cTelegramEditMessageCaption = cTelegramAPIBase + "/editMessageCaption"
    cTelegramEditMessageMedia   = cTelegramAPIBase + "/editMessageMedia"
)

func EditTelegramMessageMedia(messageID string, photo, caption string, initCfg config.InitConfig) error {
    telegramAPI := fmt.Sprintf(cTelegramEditMessageMedia, initCfg.TelegramBotToken)
    
    mediaJSON, err := json.Marshal(map[string]string{
        "type":  "photo",
        "media": photo,
        "caption": caption,
        "parse_mode": "HTML",
    })
    if err != nil {
        return err
    }

    response, err := http.PostForm(
        telegramAPI,
        url.Values{
            "chat_id":    {initCfg.TelegramChatID},
            "message_id": {messageID},
            "media":      {string(mediaJSON)},
        })
    
    if err != nil {
        return fmt.Errorf("error editing message media: %v", err)
    }
    defer response.Body.Close()

    if response.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(response.Body)
        return fmt.Errorf("unexpected status code: %d, response: %s", response.StatusCode, string(body))
    }

    return nil
}

func SendTelegramMessageWithPhoto(message, photoURL string, initCfg config.InitConfig) (string, error) {
    telegramAPI := fmt.Sprintf(cTelegramSendPhoto, initCfg.TelegramBotToken)
    
    response, err := http.PostForm(
        telegramAPI,
        url.Values{
            "chat_id": {initCfg.TelegramChatID},
            "caption": {message},  // Use caption instead of text for photos
            "photo": {photoURL},
            "parse_mode": {"HTML"},
            "allow_sending_without_reply": {"true"},
            "protect_content": {"false"},
        })
    
    if err != nil {
        return "", fmt.Errorf("error sending telegram message with photo: %v", err)
    }
    defer response.Body.Close()

    if response.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(response.Body)
        return "", fmt.Errorf("unexpected status code: %d, response: %s", response.StatusCode, string(body))
    }

    var result struct {
        Ok     bool `json:"ok"`
        Result struct {
            MessageID int `json:"message_id"`
        } `json:"result"`
    }

    body, err := io.ReadAll(response.Body)
    if err != nil {
        return "", fmt.Errorf("error reading response body: %v", err)
    }

    if err := json.Unmarshal(body, &result); err != nil {
        return "", fmt.Errorf("error parsing response: %v", err)
    }

    return fmt.Sprintf("%d", result.Result.MessageID), nil
}

func EditTelegramMessageWithPhoto(messageID, caption string, initCfg config.InitConfig) error {
    telegramAPI := fmt.Sprintf(cTelegramEditMessageCaption, initCfg.TelegramBotToken)
    
    response, err := http.PostForm(
        telegramAPI,
        url.Values{
            "chat_id":     {initCfg.TelegramChatID},
            "message_id":  {messageID},
            "caption":     {caption},
            "parse_mode":  {"HTML"},
        })
    
    if err != nil {
        return fmt.Errorf("error editing telegram message caption: %v", err)
    }
    defer response.Body.Close()

    if response.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(response.Body)
        return fmt.Errorf("unexpected status code: %d, response: %s", response.StatusCode, string(body))
    }

    return nil
}