package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

var (
	githubOauthConfig *oauth2.Config
)

func init() {
	githubOauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("OAUTH_CLIENT_ID"),
		ClientSecret: os.Getenv("OAUTH_CLIENT_SECRET"),
		Endpoint:     github.Endpoint,
	}
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/auth", authHandler)
	http.HandleFunc("/callback", callbackHandler)
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	fmt.Println("Server running on port " + port)
	http.ListenAndServe(":"+port, nil)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OAuth Server is Running!"))
}

func authHandler(w http.ResponseWriter, r *http.Request) {
	url := githubOauthConfig.AuthCodeURL("state", oauth2.AccessTypeOnline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	token, err := githubOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, "Error exchanging code for token", http.StatusInternalServerError)
		return
	}

	// Respuesta para Decap CMS
	content := fmt.Sprintf(`
	<script>
		const receiveMessage = (message) => {
			window.opener.postMessage(
				'authorization:github:success:{"token":"%s","provider":"github"}',
				message.origin
			);
			window.close();
		};
		window.addEventListener("message", receiveMessage, false);
		// Enviar mensaje inmediatamente por si acaso
		window.opener.postMessage(
			'authorization:github:success:{"token":"%s","provider":"github"}',
			"*"
		);
		window.close();
	</script>
	`, token.AccessToken, token.AccessToken)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(content))
}
