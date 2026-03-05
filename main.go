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

const htmlIndex = `
<!DOCTYPE html>
<html>
<head><title>FringeShop PRO</title><script src="https://cdn.tailwindcss.com"></script></head>
<body class="bg-[#36393f] text-white p-10 font-sans">
    <div class="max-w-4xl mx-auto">
        <h1 class="text-3xl font-bold mb-6">🛸 Nexus Destroyers Dashboard</h1>
        <div class="grid grid-cols-2 gap-4 mb-6">
            <textarea id="tokens" placeholder="Tokens aquí..." class="bg-[#202225] p-4 rounded h-40 outline-none"></textarea>
            <textarea id="proxies" placeholder="Proxies (opcional)..." class="bg-[#202225] p-4 rounded h-40 outline-none"></textarea>
        </div>
        <div class="flex gap-4 mb-6">
            <input type="text" id="val" class="bg-[#202225] p-2 rounded flex-grow" placeholder="BIO / Nombre">
            <button onclick="run('bio')" class="bg-indigo-500 px-6 py-2 rounded font-bold">Cambiar BIO</button>
            <button onclick="run('check')" class="bg-gray-600 px-6 py-2 rounded font-bold">Check Status</button>
        </div>
        <div id="logs" class="bg-black p-4 rounded h-40 overflow-y-auto font-mono text-xs text-green-400">
            [SYSTEM] Esperando tokens...
        </div>
    </div>
    <script>
        async function run(type) {
            const tokens = document.getElementById('tokens').value.split('\n').filter(t => t.trim() !== "");
            const val = document.getElementById('val').value;
            const logBox = document.getElementById('logs');
            logBox.innerHTML += "<br>> Ejecutando " + type + "...";
            const res = await fetch('/api/action', { method: 'POST', body: JSON.stringify({ action: type, tokens: tokens, value: val }) });
            const text = await res.text();
            logBox.innerHTML += "<br>> " + text;
        }
    </script>
</body>
</html>
`

func main() {
	port := os.Getenv("PORT")
	if port == "" { port = "8080" }

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t, _ := template.New("index").Parse(htmlIndex)
		t.Execute(w, nil)
	})

	http.HandleFunc("/api/action", handleAction)

	fmt.Printf("🚀 App prendida en 0.0.0.0:%s\n", port)
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
			url := "https://discord.com/api/v9/users/@me/profile"
			method := "PATCH"
			var body []byte
			if req.Action == "bio" {
				body, _ = json.Marshal(map[string]string{"bio": req.Value})
			} else {
				method = "GET"
				url = "https://discord.com/api/v9/users/@me"
			}
			request, _ := http.NewRequest(method, url, bytes.NewBuffer(body))
			request.Header.Set("Authorization", strings.TrimSpace(tk))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("User-Agent", "Mozilla/5.0")
			client := &http.Client{Timeout: 5 * time.Second}
			resp, _ := client.Do(request)
			if resp != nil {
				if resp.StatusCode == 200 || resp.StatusCode == 204 {
					mu.Lock()
					success++
					mu.Unlock()
				}
				resp.Body.Close()
			}
		}(t)
	}
	wg.Wait()
	fmt.Fprintf(w, "Completado: %d éxitos.", success)
}
