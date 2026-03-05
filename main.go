package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"sync"
	"time"
)

// Datos para el HTML
type PageData struct {
	Username string
	Avatar   string
}

// Lo que recibe el servidor desde el botón de la web
type ActionRequest struct {
	Action string   `json:"action"`
	Tokens []string `json:"tokens"`
	Value  string   `json:"value"`
}

func main() {
	// Configuración de Koyeb
	port := os.Getenv("PORT")
	if port == "" { port = "8080" }

	// 1. Ruta para ver la web
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Aquí idealmente verificarías si está logueado
		data := PageData{
			Username: "FringeUser", 
			Avatar:   "https://cdn.discordapp.com/embed/avatars/0.png",
		}
		tmpl, _ := template.ParseFiles("templates/index.html")
		tmpl.Execute(w, data)
	})

	// 2. LA MAGIA DE LOS BOTONES: Esta ruta recibe lo que mandas desde el JS del Index
	http.HandleFunc("/api/action", handleAction)

	fmt.Printf("🚀 Dashboard encendido en puerto %s\n", port)
	http.ListenAndServe(":"+port, nil)
}

func handleAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" { return }

	var req ActionRequest
	json.NewDecoder(r.Body).Decode(&req)

	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	// Ejecutar la acción para cada token
	for _, token := range req.Tokens {
		wg.Add(1)
		go func(t string) {
			defer wg.Done()
			
			// Si la acción es cambiar BIO
			if req.Action == "bio" {
				payload, _ := json.Marshal(map[string]string{"bio": req.Value})
				discordReq, _ := http.NewRequest("PATCH", "https://discord.com/api/v9/users/@me/profile", bytes.NewBuffer(payload))
				
				discordReq.Header.Set("Authorization", t)
				discordReq.Header.Set("Content-Type", "application/json")
				discordReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

				client := &http.Client{Timeout: 5 * time.Second}
				resp, err := client.Do(discordReq)
				
				if err == nil && resp.StatusCode == 200 {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
				if resp != nil { resp.Body.Close() }
			}
		}(token)
	}

	wg.Wait()
	fmt.Fprintf(w, "Se actualizaron %d cuentas correctamente.", successCount)
}