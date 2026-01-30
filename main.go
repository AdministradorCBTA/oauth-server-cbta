package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func main() {
	// Ruta de inicio (para ver si funciona)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OAuth Server is Running (Standard Lib)!"))
	})

	// Ruta de inicio de sesión (LA QUE FALTABA)
	http.HandleFunc("/auth", authHandler)

	// Ruta de regreso de GitHub
	http.HandleFunc("/callback", callbackHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	fmt.Println("Server running on port " + port)
	http.ListenAndServe(":"+port, nil)
}

// Esta función redirige al usuario a GitHub
func authHandler(w http.ResponseWriter, r *http.Request) {
	clientID := os.Getenv("OAUTH_CLIENT_ID")
	// Redirigimos a GitHub pidiendo permiso para ver repositorios
	redirectURL := fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&scope=repo", clientID)
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// Esta función recibe el código y lo canjea por el token
func callbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "No code found", http.StatusBadRequest)
		return
	}

	clientID := os.Getenv("OAUTH_CLIENT_ID")
	clientSecret := os.Getenv("OAUTH_CLIENT_SECRET")

	requestBody, _ := json.Marshal(map[string]string{
		"client_id":     clientID,
		"client_secret": clientSecret,
		"code":          code,
	})

	req, _ := http.NewRequest("POST", "https://github.com/login/oauth/access_token", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to connect to GitHub", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	token, ok := result["access_token"].(string)
	if !ok {
		// A veces GitHub manda el error en el body
		http.Error(w, "GitHub did not return a token. Check Client ID/Secret.", http.StatusInternalServerError)
		return
	}

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
		window.opener.postMessage(
			'authorization:github:success:{"token":"%s","provider":"github"}',
			"*"
		);
		window.close();
	</script>
	`, token, token)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(content))
}
