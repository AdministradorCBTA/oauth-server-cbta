package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func main() {
	// Ruta para verificar si está vivo
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OAuth Server is Live & Ready!"))
	})

	// Ruta 1: Iniciar Login
	http.HandleFunc("/auth", authHandler)

	// Ruta 2: Recibir respuesta de GitHub
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
	// Pedimos acceso
	redirectURL := fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&scope=repo,user", clientID)
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Obtener el código
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "No code found", http.StatusBadRequest)
		return
	}

	// 2. Canjear código por token
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

	// 3. SCRIPT AUTOMÁTICO (La parte importante)
	// Envía el mensaje y se cierra.
	content := fmt.Sprintf(`
	<html>
	<body>
	<p>Autenticando... espera un momento.</p>
	<script>
		const message = 'authorization:github:success:{"token":"%s","provider":"github"}';
		
		// Enviar a la ventana que nos abrió (el CMS)
		window.opener.postMessage(message, "*");
		
		// Cerrar esta ventana inmediatamente
		window.close();
	</script>
	</body>
	</html>
	`, token)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(content))
}
