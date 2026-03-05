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
)

const htmlIndex = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8"><title>Byte | Token Manager</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <style>
        :root { --accent: #8b5cf6; }
        body { background: #030303; color: #fafafa; font-family: 'Inter', sans-serif; }
        .glass { background: rgba(255,255,255,0.02); border: 1px solid rgba(255,255,255,0.08); backdrop-filter: blur(20px); }
    </style>
</head>
<body class="p-10">
    <div class="max-w-2xl mx-auto">
        <div class="flex items-center gap-3 mb-10">
            <div class="w-10 h-10 bg-[#8b5cf6] rounded-xl flex items-center justify-center font-bold">B</div>
            <h1 class="text-2xl font-bold font-['Space_Grotesk']">Byte <span class="text-zinc-500">Selfbot</span></h1>
        </div>
        <div class="glass p-8 rounded-3xl">
            <label class="text-[10px] uppercase tracking-widest text-zinc-500 font-bold mb-4 block">Load Tokens</label>
            <textarea id="tks" class="w-full h-48 bg-black/40 border border-zinc-800 p-4 rounded-2xl text-xs font-mono mb-6 outline-none focus:border-[#8b5cf6]" placeholder="Pega tus tokens aquí..."></textarea>
            <button onclick="start()" class="w-full bg-[#8b5cf6] py-4 rounded-2xl font-bold shadow-lg shadow-purple-500/20 hover:scale-[1.02] transition-all">CONECTAR TOKENS</button>
        </div>
        <div id="status" class="mt-6 text-center text-xs text-zinc-500 font-mono"></div>
    </div>
    <script>
        async function start() {
            const t = document.getElementById('tks').value.split('\n').filter(x => x.trim() !== "");
            document.getElementById('status').innerText = "Conectando " + t.length + " cuentas...";
            await fetch('/api/start', { method: 'POST', body: JSON.stringify({ tokens: t }) });
            document.getElementById('status').innerText = "SISTEMA ACTIVO. Usa .help en Discord";
        }
    </script>
</body>
</html>`

func main() {
	port := os.Getenv("PORT")
	if port == "" { port = "8080" }

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t, _ := template.New("i").Parse(htmlIndex); t.Execute(w, nil)
	})
	http.HandleFunc("/api/start", handleStart)

	log.Printf("Byte Dashboard prendido en puerto %s", port)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+port, nil))
}

func handleStart(w http.ResponseWriter, r *http.Request) {
	var req struct { Tokens []string `json:"tokens"` }
	json.NewDecoder(r.Body).Decode(&req)
	for _, token := range req.Tokens {
		go startSelfbot(strings.TrimSpace(token))
	}
	fmt.Fprintf(w, "OK")
}

func startSelfbot(token string) {
	lastMsgID := ""
	for {
		// Polling de mensajes (forma más estable para Koyeb)
		req, _ := http.NewRequest("GET", "https://discord.com/api/v9/users/@me/messages?limit=1", nil)
		req.Header.Set("Authorization", token)
		resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
		
		if err == nil && resp.StatusCode == 200 {
			var msgs []map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&msgs)
			resp.Body.Close()

			if len(msgs) > 0 {
				msg := msgs[0]
				id := msg["id"].(string)
				content := msg["content"].(string)
				if id != lastMsgID {
					lastMsgID = id
					if strings.HasPrefix(content, ".") {
						processCommand(token, content, msg["channel_id"].(string))
					}
				}
			}
		}
		time.Sleep(2 * time.Second)
	}
}

func processCommand(token, content, channelID string) {
	args := strings.Split(content, " ")
	cmd := args[0]

	switch cmd {
	case ".help":
		sendMsg(token, channelID, "⚡ **Byte Selfbot System**\n`.vc [ID]` - Join Voice\n`.bio [text]` - Change Bio\n`.check` - Account Status")
	case ".vc":
		if len(args) > 1 {
			// Nota: En selfbots, Join VC suele requerir GuildID.
			sendMsg(token, channelID, "⏳ Intentando entrar al canal: " + args[1])
		}
	case ".bio":
		bio := strings.Join(args[1:], " ")
		updateBio(token, bio)
		sendMsg(token, channelID, "✅ Bio actualizada a: " + bio)
	}
}

func sendMsg(token, channelID, content string) {
	body, _ := json.Marshal(map[string]string{"content": content})
	r, _ := http.NewRequest("POST", "https://discord.com/api/v9/channels/"+channelID+"/messages", bytes.NewBuffer(body))
	r.Header.Set("Authorization", token)
	r.Header.Set("Content-Type", "application/json")
	(&http.Client{}).Do(r)
}

func updateBio(token, bio string) {
	body, _ := json.Marshal(map[string]string{"bio": bio})
	r, _ := http.NewRequest("PATCH", "https://discord.com/api/v9/users/@me/profile", bytes.NewBuffer(body))
	r.Header.Set("Authorization", token)
	r.Header.Set("Content-Type", "application/json")
	(&http.Client{}).Do(r)
}
