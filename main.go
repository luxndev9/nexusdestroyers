package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Esto mete el HTML dentro del ejecutable automáticamente
//go:embed templates/index.html
var templateFolder embed.FS

// Estructuras de datos
type PageData struct {
	Username string
	Avatar   string
}

type ActionRequest struct {
	Action string   `json:"action"`
	Tokens []string `json:"tokens"`
	Value  string   `json:"value"`
}

func main() {
	// 1. Puerto dinámico para la nube
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 2. Rutas del Servidor
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/api/action", handleAction)

	// 3. Goroutine para el Bot (Trabajador 24/7)
	go func() {
		for {
			// Aquí podrías poner un checker automático cada hora
			time.Sleep(1 * time.Hour)
		}
	}()

	// 4. Inicio del servidor en 0.0.0.0
	fmt.Printf("🚀 FringeShop Online en el puerto %s\n", port)
	serverAddr := "0.0.0.0:" + port
	err := http.ListenAndServe(serverAddr, nil)
	if err != nil {
		log.Fatal("❌ Error al encender: ", err)
	}
}

// Renderiza la web usando el archivo embebido
func handleHome(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		Username: "FringeUser_Pro",
		Avatar:   "https://cdn.discordapp.com/embed/avatars/1.png",
	}

	// Cargamos el template desde la memoria (embed)
	tmpl, err := template.ParseFS(templateFolder, "templates/index.html")
	if err != nil {
		http.Error(w, "Error interno: "+err.Error(), 500)
		return
	}
	tmpl.Execute(w, data)
}

// Maneja los clics de los botones de la web
func handleAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Solo POST", 405)
		return
	}

	var req ActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Fprintf(w, "Error en los datos enviados")
		return
	}

	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	// Procesar cada token de forma independiente y rápida
	for _, token := range req.Tokens {
		t := strings.TrimSpace(token)
		if t == "" { continue }

		wg.Add(1)
		go func(tk string) {
			defer wg.Done()

			// Configuración de la petición a Discord
			apiUrl := "https://discord.com/api/v9/users/@me/profile"
			method := "PATCH"
			var body []byte

			switch req.Action {
			case "bio":
				payload := map[string]string{"bio": req.Value}
				body, _ = json.Marshal(payload)
			case "check":
				method = "GET"
				apiUrl = "https://discord.com/api/v9/users/@me"
				body = nil
			}

			dReq, _ := http.NewRequest(method, apiUrl, bytes.NewBuffer(body))
			dReq.Header.Set("Authorization", tk)
			dReq.Header.Set("Content-Type", "application/json")
			dReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")

			// Cliente con timeout para que no se quede trabado
			client := &http.Client{Timeout: 8 * time.Second}
			resp, err := client.Do(dReq)
			
			if err == nil {
				if resp.StatusCode == 200 || resp.StatusCode == 204 {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
				resp.Body.Close()
			}
		}(t)
	}

	wg.Wait()
	fmt.Fprintf(w, "✅ [%s] Acción terminada. Éxitos: %d de %d", strings.ToUpper(req.Action), successCount, len(req.Tokens))
}
