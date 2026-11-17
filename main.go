// gd-demo-8-mesh is step 8 of the git-deploy demo: mesh exposures. One small
// application in two roles. Every instance answers /orders and /admin, and
// says which workload called it. With ORDERS_URL set, its page also calls
// that orders service and shows what each call got back; with EXTERNAL_URL
// set, it also calls that address outside the cluster.
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// version is bumped on each demo commit to make the rolling update visible.
const version = "v1"

func main() {
	log.Printf("gd-demo-8-mesh %s starting on :8080", version)
	ordersURL := strings.TrimSuffix(os.Getenv("ORDERS_URL"), "/")
	externalURL := os.Getenv("EXTERNAL_URL")
	client := &http.Client{Timeout: 3 * time.Second}

	http.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			printf(w, "order #1003 created, called by %s\n", caller(r))
			return
		}
		printf(w, "orders #1001, #1002, called by %s\n", caller(r))
	})

	http.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		printf(w, "admin area, called by %s\n", caller(r))
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		hostname, _ := os.Hostname()
		printf(w, "Hello from gd-demo-8-mesh %s\n", version)
		printf(w, "\npod:     %s\n", hostname)
		printf(w, "served:  %s\n", time.Now().Format(time.RFC3339))

		if ordersURL == "" {
			printf(w, "\nNo ORDERS_URL: this instance only serves /orders and /admin.\n")
		} else {
			printf(w, "\nCalls to %s:\n\n", ordersURL)
			for _, c := range []struct{ method, path string }{
				{http.MethodGet, "/orders"},
				{http.MethodPost, "/orders"},
				{http.MethodGet, "/admin"},
			} {
				status, body := call(client, c.method, ordersURL+c.path)
				printf(w, "  %-4s %-7s  %-3s  %s\n", c.method, c.path, status, body)
			}
		}

		if externalURL != "" {
			printf(w, "\nCall outside the cluster:\n\n")
			status, body := call(client, http.MethodGet, externalURL)
			// A web page often fits on one long line: keep this one short.
			if r := []rune(body); len(r) > 60 {
				body = string(r[:60]) + "…"
			}
			printf(w, "  %-4s %-7s  %-3s  %s\n", http.MethodGet, externalURL, status, body)
		}

		printf(w, "\nStep 8 of the git-deploy demo: mesh exposures.\n")
		printf(w, "Who may call orders, and how, is declared on orders with\n")
		printf(w, "`git-deploy expose consumer add`: no code change needed here.\n")
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}

// caller is the identity of the calling workload. In the mesh, the sidecar
// verifies the caller's certificate and appends its SPIFFE ID to the
// X-Forwarded-Client-Cert header; the last element is the one it added.
func caller(r *http.Request) string {
	xfcc := r.Header.Get("X-Forwarded-Client-Cert")
	if xfcc == "" {
		return "an unknown caller (no mesh identity)"
	}
	elements := strings.Split(xfcc, ",")
	for _, field := range strings.Split(elements[len(elements)-1], ";") {
		if uri, ok := strings.CutPrefix(field, "URI="); ok {
			return uri
		}
	}
	return "an unknown caller (no mesh identity)"
}

// call makes one request to the orders service and returns its status and the
// first line of its body, or why there was no answer.
func call(client *http.Client, method, url string) (string, string) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return "---", err.Error()
	}
	resp, err := client.Do(req)
	if err != nil {
		return "---", "no answer: " + err.Error()
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	line, _, _ := strings.Cut(strings.TrimSpace(string(body)), "\n")
	return fmt.Sprint(resp.StatusCode), line
}

// printf writes one line of the page, discarding the write error on purpose: a
// client hanging up mid-response is this server's normal weather, and there is
// nothing left to send it.
func printf(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}
