package main

import (
	"bytes"
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

// Estructuras
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
	// 1. Configuracion de Red para Koyeb
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 2. Trabajador en Segundo Plano (Bot 24/7)
	go func() {
		for {
			// Aquí puedes poner lógica de auto-check o lo que quieras
			// fmt.Println("[BOT] Sistema activo en segundo plano...")
			time.Sleep(1 * time.Hour)
		}
	}()

	// 3. Rutas del Servidor Web
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/api/action", handleAction)

	// 4. Encendido (0.0.0.0 es clave para evitar el error de headers)
	fmt.Printf("🌐 FringeShop Panel en http://0.0.0.0:%s\n", port)
	err := http.ListenAndServe("0.0.0.0:"+port, nil)
	if err != nil {
		log.Fatal("❌ Error al iniciar servidor: ", err)
	}
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		Username: "FringeUser", // Aquí irá el nombre de Discord después
		Avatar:   "https://cdn.discordapp.com/embed/avatars/0.png",
	}
	
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "Error cargando template: "+err.Error(), 500)
		return
	}
	tmpl.Execute(w, data)
}

func handleAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var req ActionRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		fmt.Fprintf(w, "Error en datos")
		return
	}

	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for _, token := range req.Tokens {
		t := strings.TrimSpace(token)
		if t == "" { continue }

		wg.Add(1)
		go func(tk string) {
			defer wg.Done()

			// Lógica según la acción
			apiUrl := "https://discord.com/api/v9/users/@me/profile"
			method := "PATCH"
			var body []byte

			if req.Action == "bio" {
				payload := map[string]string{"bio": req.Value}
				body, _ = json.Marshal(payload)
			} else if req.Action == "check" {
				method = "GET"
				apiUrl = "https://discord.com/api/v9/users/@me"
				body = nil
			}

			dReq, _ := http.NewRequest(method, apiUrl, bytes.NewBuffer(body))
			dReq.Header.Set("Authorization", tk)
			dReq.Header.Set("Content-Type", "application/json")
			dReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

			client := &http.Client{Timeout: 7 * time.Second}
			resp, err := client.Do(dReq)
			
			if err == nil && (resp.StatusCode == 200 || resp.StatusCode == 204) {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
			if resp != nil { resp.Body.Close() }
		}(t)
	}

	wg.Wait()
	fmt.Fprintf(w, "✅ Acción [%s] completada en %d cuentas.", req.Action, successCount)
}
