package main

import (
    "fmt"
    "net/http"
)

func main() {
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprint(w, "hello")
    })

    fmt.Println("Server listening on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}