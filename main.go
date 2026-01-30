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

	// Preparamos el comando exacto para copiar y pegar
	consoleCommand := fmt.Sprintf(`localStorage.setItem("netlify-cms-user", JSON.stringify({"token":"%s","backend":"github"})); location.reload();`, token)

	content := fmt.Sprintf(`
	<html>
	<head>
		<title>Acceso Generado</title>
		<style>
			body { font-family: sans-serif; padding: 20px; text-align: center; background: #f0f2f5; }
			.card { background: white; padding: 20px; border-radius: 10px; box-shadow: 0 4px 6px rgba(0,0,0,0.1); max-width: 600px; margin: 20px auto; }
			textarea { width: 100%%; height: 100px; margin-top: 10px; padding: 10px; border: 1px solid #ccc; border-radius: 5px; font-family: monospace; font-size: 14px; color: #333; }
			h2 { color: #2ea44f; }
			.instrucciones { text-align: left; margin-top: 20px; font-size: 15px; line-height: 1.6; }
			code { background: #eee; padding: 2px 5px; border-radius: 3px; }
		</style>
	</head>
	<body>
		<div class="card">
			<h2>✅ ¡Token Recibido con Éxito!</h2>
			<p>Tu navegador bloqueó la conexión automática, pero ya tenemos tu llave.</p>
			
			<div class="instrucciones">
				<strong>Paso 1:</strong> Copia TODO el código del siguiente cuadro:
				<textarea onclick="this.select()">%s</textarea>
				
				<br><br>
				<strong>Paso 2:</strong> Vuelve a la pestaña del administrador (donde está el botón de Login).
				<br>
				<strong>Paso 3:</strong> Presiona <code>F12</code>, ve a la pestaña <strong>Console</strong> (Consola).
				<br>
				<strong>Paso 4:</strong> Pega el código y presiona <strong>ENTER</strong>.
			</div>
		</div>
	</body>
	</html>
	`, consoleCommand, token)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(content))
}
