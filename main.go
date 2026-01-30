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

	// Preparamos el comando exacto para la consola
	consoleCommand := fmt.Sprintf(`localStorage.setItem("netlify-cms-user", JSON.stringify({"token":"%s","backend":"github"})); location.reload();`, token)

	content := fmt.Sprintf(`
	<html>
	<head>
		<title>Autenticación GitHub</title>
		<style>
			body { font-family: sans-serif; padding: 20px; text-align: center; background: #f4f4f4; }
			.container { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); max-width: 600px; margin: 0 auto; }
			textarea { width: 100%%; height: 80px; margin-top: 10px; font-family: monospace; font-size: 12px; padding: 10px; border: 1px solid #ccc; border-radius: 4px; }
			h2 { color: #2ea44f; }
			.step { margin: 15px 0; text-align: left; }
			strong { color: #333; }
		</style>
	</head>
	<body>
		<div class="container">
			<h2>✅ ¡Conexión con GitHub Exitosa!</h2>
			<p>Si la ventana principal no cambió automáticamente, el navegador bloqueó la conexión.</p>
			<hr>
			
			<div class="step">
				<strong>PASO 1:</strong> Copia todo el código de este recuadro:
				<textarea onclick="this.select()">%s</textarea>
			</div>

			<div class="step">
				<strong>PASO 2:</strong> Ve a la pestaña del Administrador (donde está el botón de Login).
			</div>

			<div class="step">
				<strong>PASO 3:</strong> Presiona <code>F12</code>, ve a la pestaña <strong>Console</strong> (Consola), pega el código y dale <strong>ENTER</strong>.
			</div>
		</div>

		<script>
			// Intento automático (por si acaso funciona)
			const message = 'authorization:github:success:{"token":"%s","provider":"github"}';
			window.opener.postMessage(message, "*");
		</script>
	</body>
	</html>
	`, consoleCommand, token)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(content))
}
