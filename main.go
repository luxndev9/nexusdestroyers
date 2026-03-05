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

// HTML Byte Style
const htmlIndex = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8" /><title>Dashboard - Byte</title>
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css" />
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=Space+Grotesk:wght@500;600;700&display=swap" rel="stylesheet" />
    <script src="https://cdn.tailwindcss.com"></script>
    <style>
        :root { --bg: #030303; --surface: rgba(255, 255, 255, 0.03); --accent: #8b5cf6; --text: #fafafa; }
        body { background: var(--bg); color: var(--text); font-family: "Inter", sans-serif; }
        .glass-card { background: rgba(255, 255, 255, 0.02); backdrop-filter: blur(20px); border: 1px solid rgba(255,255,255,0.08); border-radius: 20px; }
        .cmd-input { background: rgba(10, 10, 10, 0.8); border: 1px solid rgba(255,255,255,0.1); outline: none; transition: 0.2s; }
        .cmd-input:focus { border-color: var(--accent); }
    </style>
</head>
<body class="p-6">
    <div class="max-w-[800px] mx-auto pt-16">
        <div class="flex items-center gap-3 mb-10">
            <div class="w-10 h-10 bg-[#8b5cf6] rounded-xl flex items-center justify-center"><i class="fas fa-bolt text-white"></i></div>
            <h1 class="text-2xl font-bold font-['Space_Grotesk'] text-white">Byte <span class="text-zinc-600">v3.0</span></h1>
        </div>
        <div class="glass-card p-8 mb-6">
            <textarea id="tokens" class="w-full h-32 bg-black/40 border border-zinc-800 p-4 rounded-xl text-[10px] font-mono mb-4 outline-none focus:border-[#8b5cf6]" placeholder="Tokens aquí..."></textarea>
            <div class="relative">
                <input type="text" id="cmdLine" onkeypress="if(event.key==='Enter') executeCmd()" class="cmd-input w-full p-4 rounded-xl text-sm font-mono" placeholder="Ej: -vc ServerID:ChannelID o -bio Texto">
                <button onclick="executeCmd()" class="absolute right-3 top-2.5 bg-[#8b5cf6] text-white px-4 py-1.5 rounded-lg text-xs font-bold">RUN</button>
            </div>
        </div>
        <div id="logs" class="bg-black/40 border border-zinc-900 rounded-xl p-6 h-48 overflow-y-auto font-mono text-[10px] text-zinc-500 space-y-1">
            <div>-- Terminal Ready. Use -help --</div>
        </div>
    </div>
    <script>
        async function executeCmd() {
            const input = document.getElementById('cmdLine');
            const tokens = document.getElementById('tokens').value.split('\n').filter(t => t.trim() !== "");
            const raw = input.value.trim();
            if(!raw || tokens.length === 0) return;
            const logBox = document.getElementById('logs');
            logBox.innerHTML += '<div class="text-zinc-400">> Processing: ' + raw + '</div>';
            input.value = '';
            const parts = raw.split(' ');
            const res = await fetch('/api/action', {
                method: 'POST',
                body: JSON.stringify({ action: parts[0].replace('-',''), tokens: tokens, value: parts.slice(1).join(' ') })
            });
            const result = await res.text();
            logBox.innerHTML += '<div class="text-[#8b5cf6]">> ' + result + '</div>';
            logBox.scrollTop = logBox.scrollHeight;
        }
    </script>
</body>
</html>
`

func main() {
	port := os.Getenv("PORT")
	if port == "" { port = "8080" }
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t, _ := template.New("index").Parse(htmlIndex); t.Execute(w, nil)
	})
	http.HandleFunc("/api/action", handleAction)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+port, nil))
}

func handleAction(w http.ResponseWriter, r *http.Request) {
	var req struct { Action string `json:"action"`; Tokens []string `json:"tokens"`; Value string `json:"value"` }
	json.NewDecoder(r.Body).Decode(&req)
	var wg sync.WaitGroup
	success := 0
	var mu sync.Mutex

	for _, t := range req.Tokens {
		wg.Add(1)
		go func(tk string) {
			defer wg.Done()
			tk = strings.TrimSpace(tk)
			var ok bool
			if req.Action == "bio" {
				ok = discordReq("PATCH", "https://discord.com/api/v9/users/@me/profile", tk, map[string]string{"bio": req.Value})
			} else if req.Action == "vc" {
				// Simulación de VC vía REST (para no usar websockets externos)
				p := strings.Split(req.Value, ":")
				if len(p) == 2 {
					ok = discordReq("PATCH", "https://discord.com/api/v9/guilds/"+p[0]+"/members/@me", tk, map[string]string{"channel_id": p[1]})
				}
			} else {
				ok = discordReq("GET", "https://discord.com/api/v9/users/@me", tk, nil)
			}
			if ok { mu.Lock(); success++; mu.Unlock() }
		}(t)
	}
	wg.Wait()
	fmt.Fprintf(w, "Acción %s: %d éxitos", req.Action, success)
}

func discordReq(method, url, token string, body interface{}) bool {
	var b []byte
	if body != nil { b, _ = json.Marshal(body) }
	r, _ := http.NewRequest(method, url, bytes.NewBuffer(b))
	r.Header.Set("Authorization", token)
	r.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(r)
	return err == nil && (resp.StatusCode == 200 || resp.StatusCode == 204)
}
