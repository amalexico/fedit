package main

import (
"log"
"net/http"
)

func main() {
http.HandleFunc("/health", healthEndpoint)
http.HandleFunc("/api/users", usersEndpoint)
http.HandleFunc("/api/orders", ordersEndpoint)
// TODO: add products endpoint here
log.Println("server starting on :8080")
log.Fatal(http.ListenAndServe(":8080", nil))
}

func healthEndpoint(w http.ResponseWriter, r *http.Request) {
w.Write([]byte("ok"))
}

func usersEndpoint(w http.ResponseWriter, r *http.Request) {
w.Write([]byte("users"))
}

func ordersEndpoint(w http.ResponseWriter, r *http.Request) {
w.Write([]byte("orders"))
}
