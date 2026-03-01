package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

var ready atomic.Bool

const maxN = 10000

var indexTemplate = template.Must(template.New("index").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Fibonacci Calculator</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:system-ui,sans-serif;background:#0f172a;color:#e2e8f0;min-height:100vh;display:flex;align-items:center;justify-content:center}
.card{background:#1e293b;border-radius:16px;padding:2.5rem;max-width:480px;width:100%;box-shadow:0 25px 50px rgba(0,0,0,.4)}
h1{font-size:1.75rem;margin-bottom:.5rem;background:linear-gradient(135deg,#818cf8,#c084fc);-webkit-background-clip:text;-webkit-text-fill-color:transparent}
.sub{color:#94a3b8;margin-bottom:2rem;font-size:.9rem}
label{display:block;color:#94a3b8;font-size:.85rem;margin-bottom:.5rem}
input{width:100%;padding:.75rem 1rem;border-radius:8px;border:1px solid #334155;background:#0f172a;color:#e2e8f0;font-size:1rem;outline:none}
input:focus{border-color:#818cf8}
button{width:100%;margin-top:1rem;padding:.75rem;border-radius:8px;border:none;background:linear-gradient(135deg,#6366f1,#8b5cf6);color:#fff;font-size:1rem;font-weight:600;cursor:pointer}
button:hover{opacity:.9}button:disabled{opacity:.5;cursor:not-allowed}
.result{margin-top:1.5rem;padding:1rem;background:#0f172a;border-radius:8px;border:1px solid #334155;word-break:break-all;font-family:monospace;font-size:.85rem;max-height:200px;overflow-y:auto}
.result .lbl{color:#94a3b8;font-size:.75rem;text-transform:uppercase;letter-spacing:.05em;margin-bottom:.25rem}
.result .val{color:#a5b4fc}
.meta{margin-top:1.5rem;display:flex;gap:1rem;color:#475569;font-size:.75rem}
.err{color:#f87171}.tm{color:#34d399;font-size:.8rem;margin-top:.5rem}
</style>
</head>
<body>
<div class="card">
  <h1>Fibonacci Calculator</h1>
  <p class="sub">Knative Serverless Function</p>
  <label for="n">Enter n (0 - {{.MaxN}})</label>
  <input type="number" id="n" min="0" max="{{.MaxN}}" value="10" autofocus>
  <button id="btn" onclick="compute()">Calculate</button>
  <div id="result" class="result" style="display:none">
    <div class="lbl">F(<span id="rn"></span>)</div>
    <div class="val" id="rv"></div>
    <div class="tm" id="rt"></div>
  </div>
  <div id="error" class="err" style="display:none"></div>
  <div class="meta"><span>v{{.Version}}</span><span>{{.Commit}}</span></div>
</div>
<script>
async function compute(){
  const n=document.getElementById('n').value,btn=document.getElementById('btn'),
  res=document.getElementById('result'),err=document.getElementById('error');
  btn.disabled=true;btn.textContent='Computing...';err.style.display='none';res.style.display='none';
  try{const r=await fetch('/api/fib',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({n:parseInt(n)})});
  const d=await r.json();if(!r.ok){err.textContent=d.error;err.style.display='block';return}
  document.getElementById('rn').textContent=d.n;document.getElementById('rv').textContent=d.result;
  document.getElementById('rt').textContent='Computed in '+d.duration;res.style.display='block';
  }catch(e){err.textContent=e.message;err.style.display='block'}
  finally{btn.disabled=false;btn.textContent='Calculate'}}
document.getElementById('n').addEventListener('keydown',e=>{if(e.key==='Enter')compute()});
</script>
</body>
</html>`))

type fibRequest struct {
	N int `json:"n"`
}

type fibResponse struct {
	N        int    `json:"n"`
	Result   string `json:"result"`
	Digits   int    `json:"digits"`
	Duration string `json:"duration"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type indexData struct {
	Version string
	Commit  string
	MaxN    int
}

func fibonacci(n int) *big.Int {
	if n <= 0 {
		return big.NewInt(0)
	}
	if n == 1 {
		return big.NewInt(1)
	}
	a, b := big.NewInt(0), big.NewInt(1)
	for i := 2; i <= n; i++ {
		a.Add(a, b)
		a, b = b, a
	}
	return b
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	indexTemplate.Execute(w, indexData{
		Version: Version,
		Commit:  shortCommit(Commit),
		MaxN:    maxN,
	})
}

func handleFib(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "POST required"})
		return
	}
	var req fibRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON"})
		return
	}
	if req.N < 0 || req.N > maxN {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: fmt.Sprintf("n must be between 0 and %d", maxN),
		})
		return
	}
	start := time.Now()
	result := fibonacci(req.N)
	duration := time.Since(start)
	resultStr := result.String()
	slog.Info("fibonacci computed", "n", req.N, "digits", len(resultStr), "duration", duration.String())
	writeJSON(w, http.StatusOK, fibResponse{
		N:        req.N,
		Result:   resultStr,
		Digits:   len(resultStr),
		Duration: duration.String(),
	})
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok","version":"%s"}`, Version)
}

func handleReady(w http.ResponseWriter, _ *http.Request) {
	if !ready.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, `{"status":"not ready"}`)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status":"ready"}`)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func shortCommit(s string) string {
	if len(s) > 7 {
		return s[:7]
	}
	return s
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/api/fib", handleFib)
	mux.HandleFunc("/healthz", handleHealth)
	mux.HandleFunc("/readyz", handleReady)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	ready.Store(true)

	slog.Info("starting server", "port", port, "version", Version, "commit", shortCommit(Commit))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")

	ready.Store(false)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
	slog.Info("server stopped")
}
