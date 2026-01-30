package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OAuth Server is Running (Standard Lib)!"))
	})

	http.HandleFunc("/callback", callbackHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	fmt.Println("Server running on port " + port)
	http.ListenAndServe(":"+port, nil)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Obtener el código que nos manda GitHub
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "No code found", http.StatusBadRequest)
		return
	}

	// 2. Preparar los datos para canjear el código por el token
	clientID := os.Getenv("OAUTH_CLIENT_ID")
	clientSecret := os.Getenv("OAUTH_CLIENT_SECRET")

	requestBody, _ := json.Marshal(map[string]string{
		"client_id":     clientID,
		"client_secret": clientSecret,
		"code":          code,
	})

	// 3. Hacer la petición a GitHub manualmente
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

	// 4. Leer la respuesta
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	token, ok := result["access_token"].(string)
	if !ok {
		http.Error(w, "GitHub did not return a token", http.StatusInternalServerError)
		return
	}

	// 5. Enviar el script mágico a la ventana
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
