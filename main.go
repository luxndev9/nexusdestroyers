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
	"github.com/gorilla/websocket"
)

// HTML con look Byte y Sistema de Comandos
const htmlIndex = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Dashboard - Byte</title>
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css" />
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=Space+Grotesk:wght@500;600;700&display=swap" rel="stylesheet" />
    <script src="https://cdn.tailwindcss.com"></script>
    <style>
        :root { --bg: #030303; --surface: rgba(255, 255, 255, 0.03); --accent: #8b5cf6; --text: #fafafa; }
        body { background: var(--bg); color: var(--text); font-family: "Inter", sans-serif; }
        .glass-card { background: rgba(255, 255, 255, 0.02); backdrop-filter: blur(20px); border: 1px solid rgba(255,255,255,0.08); border-radius: 20px; }
        .cmd-input { background: rgba(10, 10, 10, 0.8); border: 1px solid rgba(255,255,255,0.1); transition: 0.2s; }
        .cmd-input:focus { border-color: var(--accent); box-shadow: 0 0 15px rgba(139, 92, 246, 0.2); }
        .token-area { scrollbar-width: thin; scrollbar-color: #8b5cf6 #1a1a1a; }
    </style>
</head>
<body class="p-6">
    <div class="max-w-[800px] mx-auto pt-16">
        <div class="flex items-center gap-3 mb-10">
            <div class="w-10 h-10 bg-gradient-to-br from-[#8b5cf6] to-[#7c3aed] rounded-xl flex items-center justify-center shadow-lg shadow-purple-500/20">
                <i class="fas fa-bolt text-white"></i>
            </div>
            <h1 class="text-2xl font-bold font-['Space_Grotesk']">Byte <span class="text-zinc-500">v2.1</span></h1>
        </div>

        <div class="glass-card p-8 mb-6">
            <label class="text-[10px] uppercase tracking-[0.2em] text-zinc-500 font-bold mb-4 block">Accounts (Tokens)</label>
            <textarea id="tokens" class="token-area w-full h-40 bg-black/40 border border-zinc-800 p-4 rounded-xl text-[11px] font-mono mb-6 outline-none focus:border-[#8b5cf6]" placeholder="Token 1&#10;Token 2..."></textarea>

            <label class="text-[10px] uppercase tracking-[0.2em] text-zinc-500 font-bold mb-2 block">Command Console</label>
            <div class="relative">
                <input type="text" id="cmdLine" 
                    onkeypress="if(event.key==='Enter') executeCmd()"
                    class="cmd-input w-full p-4 rounded-xl text-sm font-mono outline-none" 
                    placeholder="Escribe un comando... (ej: -help)">
                <button onclick="executeCmd()" class="absolute right-3 top-2.5 bg-[#8b5cf6] hover:bg-[#7c3aed] text-white px-4 py-1.5 rounded-lg text-xs font-bold transition-all">RUN</button>
            </div>
        </div>

        <div id="logs" class="bg-black/40 border border-zinc-900 rounded-xl p-6 h-64 overflow-y-auto font-mono text-[11px] text-zinc-400 space-y-2">
            <div class="text-zinc-600">-- Byte Terminal Ready. Type -help to see commands. --</div>
        </div>
    </div>

    <script>
        function log(msg, type='info') {
            const logs = document.getElementById('logs');
            const color = type === 'error' ? 'text-red-400' : (type === 'success' ? 'text-[#8b5cf6]' : 'text-zinc-400');
            logs.innerHTML += '<div class="' + color + '">> ' + msg + '</div>';
            logs.scrollTop = logs.scrollHeight;
        }

        async function executeCmd() {
            const input = document.getElementById('cmdLine');
            const tokens = document.getElementById('tokens').value.split('\n').filter(t => t.trim() !== "");
            const rawCmd = input.value.trim();
            
            if(!rawCmd) return;
            input.value = '';

            if(rawCmd === '-help') {
                log('Available Commands:', 'success');
                log('-help : Muestra esta lista');
                log('-vc [ServerID]:[ChannelID] : Conecta tokens al canal de voz');
                log('-bio [texto] : Cambia la biografía de las cuentas');
                log('-check : Revisa si los tokens son válidos');
                return;
            }

            if(tokens.length === 0) { log('Error: No hay tokens cargados', 'error'); return; }

            const parts = rawCmd.split(' ');
            const action = parts[0].replace('-', '');
            const value = parts.slice(1).join(' ');

            log('Procesando ' + action + ' en ' + tokens.length + ' cuentas...');

            const res = await fetch('/api/action', {
                method: 'POST',
                body: JSON.stringify({ action: action, tokens: tokens, value: value })
            });
            const result = await res.text();
            log(result, 'success');
        }
    </script>
</body>
</html>
`

type Payload struct {
	Op int         `json:"op"`
	D  interface{} `json:"d"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" { port = "8080" }

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t, _ := template.New("index").Parse(htmlIndex)
		t.Execute(w, nil)
	})

	http.HandleFunc("/api/action", handleAction)

	log.Printf("Byte Dashboard Live on port %s", port)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+port, nil))
}

func handleAction(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action string   `json:"action"`
		Tokens []string `json:"tokens"`
		Value  string   `json:"value"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	var wg sync.WaitGroup
	success := 0
	var mu sync.Mutex

	for _, t := range req.Tokens {
		wg.Add(1)
		go func(tk string) {
			defer wg.Done()
			tk = strings.TrimSpace(tk)
			
			switch req.Action {
			case "vc":
				p := strings.Split(req.Value, ":")
				if len(p) == 2 && joinVC(tk, p[0], p[1]) == nil {
					mu.Lock(); success++; mu.Unlock()
				}
			case "bio":
				if updateBio(tk, req.Value) {
					mu.Lock(); success++; mu.Unlock()
				}
			case "check":
				if checkToken(tk) {
					mu.Lock(); success++; mu.Unlock()
				}
			}
		}(t)
	}
	wg.Wait()
	fmt.Fprintf(w, "Comando [%s] terminado. Éxitos: %d", req.Action, success)
}

func updateBio(token, bio string) bool {
	p, _ := json.Marshal(map[string]string{"bio": bio})
	req, _ := http.NewRequest("PATCH", "https://discord.com/api/v9/users/@me/profile", bytes.NewBuffer(p))
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	return err == nil && resp.StatusCode == 200
}

func checkToken(token string) bool {
	req, _ := http.NewRequest("GET", "https://discord.com/api/v9/users/@me", nil)
	req.Header.Set("Authorization", token)
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	return err == nil && resp.StatusCode == 200
}

func joinVC(token, guildID, channelID string) error {
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial("wss://gateway.discord.gg/?v=9&encoding=json", nil)
	if err != nil { return err }

	conn.WriteJSON(Payload{Op: 2, D: map[string]interface{}{
		"token": token, "properties": map[string]string{"$os": "linux", "$browser": "chrome", "$device": "pc"},
	}})

	time.Sleep(1 * time.Second)
	err = conn.WriteJSON(Payload{Op: 4, D: map[string]interface{}{
		"guild_id": guildID, "channel_id": channelID, "self_mute": false, "self_deaf": false,
	}})
	
	go func() { time.Sleep(1 * time.Hour); conn.Close() }()
	return err
}
