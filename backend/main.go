package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "github.com/jackc/pgx/v5/pgxpool"
    "encoding/json"
)

func main() {
    conn, err := pgxpool.New(context.Background(), "postgres://postgres:yourpassword@localhost:5432/postgres")
    if err != nil {
        log.Fatal("Unable to connect to database:", err)
    }
    defer conn.Close()

    err = conn.Ping(context.Background()) // Actually attempt a real round-trip to the database
    if err != nil {
        log.Fatal("Unable to reach database:", err) // This is where a wrong password or stopped container would actually get caught
    }
    fmt.Println("Successfully connected to database")

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        fmt.Fprintf(w, "Hello, go to '/leaderboard' for more.")
    })

    type ScoreSubmission struct {
        Name  string `json:"name"`  // Matches the "name" field in the incoming JSON
        Score int    `json:"score"` // Matches the "score" field
    }

    http.HandleFunc("/submit-score", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*") // Same CORS permission your other endpoint needs

        var submission ScoreSubmission
        err := json.NewDecoder(r.Body).Decode(&submission) // Read the request body and fill in the struct above
        if err != nil {
            http.Error(w, "Invalid request body", 400)
            return
        }

        _, err = conn.Exec(context.Background(),
            "INSERT INTO scores (player_name, score) VALUES ($1, $2)", submission.Name, submission.Score)
        // Exec runs SQL that doesn't return rows (INSERT/UPDATE/DELETE), unlike Query
        if err != nil {
            http.Error(w, "Insert failed", 500)
            return
        }

        fmt.Fprint(w, "Score saved")
    })

    http.HandleFunc("/leaderboard", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        rows, err := conn.Query(context.Background(), "SELECT player_name, score FROM scores ORDER BY score DESC")
        // Query() runs the SQL and gives back a set of result rows to loop through
        if err != nil {
            http.Error(w, "Query failed", 500) // Send back an HTTP 500 error if the query itself fails
            return
        }
        defer rows.Close() // Always release the result set once you're done reading it

        for rows.Next() { // Step through each row one at a time
            var name string
            var score int
            rows.Scan(&name, &score) // Copy this row's columns into your Go variables
            fmt.Fprintf(w, "%s: %d\n", name, score) // Write each one into the response
        }
    })

    fmt.Println("Server listening on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
// http://localhost:8080

