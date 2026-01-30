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
		w.Write([]byte("OAuth Server is Running!"))
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

	// --- AQUI ESTA LA MAGIA NUEVA ---
	// 1. Enviamos el mensaje a TODOS (*) los orígenes para evitar bloqueos de seguridad.
	// 2. Imprimimos el token en pantalla y NO cerramos la ventana automáticamente.
	
	content := fmt.Sprintf(`
	<html>
	<head><title>Login Exitoso</title></head>
	<body style="font-family: sans-serif; text-align: center; padding: 20px;">
		<h2 style="color: green;">¡Conexión Exitosa!</h2>
		<p>Intentando enviarte al panel automáticamente...</p>
		
		<script>
			// Intento de comunicación agresivo
			function sendMessage() {
				window.opener.postMessage('authorization:github:success:{"token":"%s","provider":"github"}', '*');
			}
			sendMessage();
			// Intentarlo cada segundo por si la ventana principal estaba ocupada
			setInterval(sendMessage, 1000);
		</script>

		<hr>
		<h3>¿No entraste automáticamente?</h3>
		<p>Copia este código y úsalo manualmente:</p>
		<textarea style="width: 100%%; height: 100px;">%s</textarea>
	</body>
	</html>
	`, token, token)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(content))
}
