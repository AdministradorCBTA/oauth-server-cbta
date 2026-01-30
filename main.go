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
		w.Write([]byte("OAuth Server is Live!"))
	})
	http.HandleFunc("/auth", authHandler)
	http.HandleFunc("/callback", callbackHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	fmt.Println("Server running on port " + port)
	http.ListenAndServe(":"+port, nil)
}

func authHandler(w http.ResponseWriter, r *http.Request) {
	clientID := os.Getenv("OAUTH_CLIENT_ID")
	redirectURL := fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&scope=repo,user", clientID)
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

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
		http.Error(w, "GitHub did not return a token", http.StatusInternalServerError)
		return
	}

	// --- CAMBIO IMPORTANTE AQUI ---
	// Enviamos el mensaje cada 0.5 segundos y esperamos 3 segundos antes de cerrar.
	content := fmt.Sprintf(`
	<html>
	<body style="background-color: #f0f0f0; font-family: sans-serif; text-align: center; padding-top: 50px;">
		<h2>Conectando...</h2>
		<p>Por favor espera, no cierres esta ventana.</p>
		<script>
			const message = 'authorization:github:success:{"token":"%s","provider":"github"}';
			
			function sendMessage() {
				console.log("Enviando token...");
				window.opener.postMessage(message, "*");
			}

			// 1. Enviar inmediatamente
			sendMessage();

			// 2. Enviar repetidamente cada 500ms por si el navegador estaba ocupado
			const interval = setInterval(sendMessage, 500);

			// 3. Cerrar la ventana después de 3 segundos
			setTimeout(function() {
				clearInterval(interval);
				window.close();
			}, 3000);
		</script>
	</body>
	</html>
	`, token)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(content))
}
