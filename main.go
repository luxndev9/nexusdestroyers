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
	"time"
	"golang.org/x/net/websocket"
)

const htmlIndex = `
<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"><title>Byte Dashboard</title>
<script src="https://cdn.tailwindcss.com"></script></head>
<body class="bg-[#030303] text-white p-10 font-sans">
    <div class="max-w-xl mx-auto">
        <h1 class="text-[#8b5cf6] text-2xl font-bold mb-6">Byte <span class="text-white font-light">Loader</span></h1>
        <div class="bg-zinc-900/50 p-6 rounded-2xl border border-zinc-800">
            <textarea id="t" class="w-full h-32 bg-black border border-zinc-700 p-4 rounded-xl text-xs mb-4 outline-none" placeholder="Pega el token aquí..."></textarea>
            <button onclick="s()" class="w-full bg-[#8b5cf6] py-3 rounded-xl font-bold">CONECTAR</button>
        </div>
        <p id="st" class="mt-4 text-center text-zinc-500 text-xs"></p>
    </div>
    <script>
        async function s() {
            const t = document.getElementById('t').value.split('\n').filter(x => x.trim() !== "");
            document.getElementById('st').innerText = "Conectando...";
            await fetch('/api/start', { method: 'POST', body: JSON.stringify({ tokens: t }) });
            document.getElementById('st').innerText = "ONLINE. Escribe .help en Discord";
        }
    </script>
</body>
</html>`

type Payload struct {
	Op int         `json:"op"`
	D  interface{} `json:"d"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" { port = "8080" }
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t, _ := template.New("i").Parse(htmlIndex); t.Execute(w, nil)
	})
	http.HandleFunc("/api/start", handleStart)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+port, nil))
}

func handleStart(w http.ResponseWriter, r *http.Request) {
	var req struct { Tokens []string `json:"tokens"` }
	json.NewDecoder(r.Body).Decode(&req)
	for _, t := range req.Tokens { go connect(strings.TrimSpace(t)) }
	fmt.Fprintf(w, "OK")
}

func connect(token string) {
	ws, err := websocket.Dial("wss://gateway.discord.gg/?v=9&encoding=json", "", "https://discord.com")
	if err != nil { return }

	// 1. Identify
	identify := Payload{Op: 2, D: map[string]interface{}{
		"token": token,
		"properties": map[string]string{"$os": "linux", "$browser": "chrome", "$device": "pc"},
	}}
	websocket.JSON.Send(ws, identify)

	// 2. Heartbeat Loop (Para que no se desconecte)
	go func() {
		for {
			time.Sleep(40 * time.Second)
			websocket.JSON.Send(ws, Payload{Op: 1, D: nil})
		}
	}()

	// 3. Listen
	for {
		var msg map[string]interface{}
		if err := websocket.JSON.Receive(ws, &msg); err != nil { break }

		if msg["t"] == "MESSAGE_CREATE" {
			d := msg["d"].(map[string]interface{})
			content := d["content"].(string)
			channelID := d["channel_id"].(string)
			author := d["author"].(map[string]interface{})

			// Solo responde si TU escribes el comando (evita bucles)
			if strings.HasPrefix(content, ".") {
				log.Printf("Comando detectado: %s", content)
				process(token, content, channelID, author["id"].(string))
			}
		}
	}
}

func process(token, content, channelID, authorID string) {
	args := strings.Split(content, " ")
	switch args[0] {
	case ".help":
		send(token, channelID, "✅ **Byte Selfbot Active**\n`.bio [texto]` - Cambia biografía\n`.vc [ChannelID]` - Entra a voz")
	case ".bio":
		if len(args) > 1 {
			updateBio(token, strings.Join(args[1:], " "))
			send(token, channelID, "👤 Biografía actualizada.")
		}
	}
}

func send(t, c, m string) {
	body, _ := json.Marshal(map[string]string{"content": m})
	req, _ := http.NewRequest("POST", "https://discord.com/api/v9/channels/"+c+"/messages", bytes.NewBuffer(body))
	req.Header.Set("Authorization", t)
	req.Header.Set("Content-Type", "application/json")
	(&http.Client{}).Do(req)
}

func updateBio(t, b string) {
	body, _ := json.Marshal(map[string]string{"bio": b})
	req, _ := http.NewRequest("PATCH", "https://discord.com/api/v9/users/@me/profile", bytes.NewBuffer(body))
	req.Header.Set("Authorization", t)
	req.Header.Set("Content-Type", "application/json")
	(&http.Client{}).Do(req)
}
