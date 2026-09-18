package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "github.com/jackc/pgx/v5/pgxpool"
    "encoding/json"
    "github.com/joho/godotenv"
)

func main() {
    err := godotenv.Load() // Reads .env into environment variables, if the file exists
    if err != nil {
        log.Println("No .env file found, relying on system environment variables") // Not fatal — Render won't have a .env file, it sets env vars directly
    }

    dbURL := os.Getenv("DATABASE_URL") // Read the connection string from the environment instead of hardcoding it
    if dbURL == "" {
        log.Fatal("DATABASE_URL environment variable is not set")
    }

    conn, err := pgxpool.New(context.Background(), dbURL)
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

    type ScoreEntry struct {
        Name  string `json:"name"`
        Score int    `json:"score"`
    }

    type SubmitResponse struct {
        Top10 []ScoreEntry `json:"top10"`
        Rank  int          `json:"rank"`
    }

    http.HandleFunc("/submit-score", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "POST") // Tell the browser which methods are allowed
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

        if r.Method == "OPTIONS" { // This is the preflight check itself — just approve it and stop here
            w.WriteHeader(http.StatusOK)
            return
        }

        var submission ScoreSubmission
        err := json.NewDecoder(r.Body).Decode(&submission)
        if err != nil {
            http.Error(w, "Invalid request body", 400)
            return
        }

        _, err = conn.Exec(context.Background(),
            "INSERT INTO scores (player_name, score) VALUES ($1, $2)", submission.Name, submission.Score)
        if err != nil {
            http.Error(w, "Insert failed", 500)
            return
        }

        // Get top 10 scores
        rows, err := conn.Query(context.Background(), "SELECT player_name, score FROM scores ORDER BY score DESC LIMIT 10")
        if err != nil {
            http.Error(w, "Query failed", 500)
            return
        }
        defer rows.Close()

        var top10 []ScoreEntry
        for rows.Next() {
            var entry ScoreEntry
            rows.Scan(&entry.Name, &entry.Score)
            top10 = append(top10, entry) // Build up the list one row at a time
        }

        // Work out this score's rank: how many existing scores beat it, plus 1
        var rank int
        err = conn.QueryRow(context.Background(),
            "SELECT COUNT(*) + 1 FROM scores WHERE score > $1", submission.Score).Scan(&rank)
        if err != nil {
            http.Error(w, "Rank query failed", 500)
            return
        }

        response := SubmitResponse{Top10: top10, Rank: rank}
        w.Header().Set("Content-Type", "application/json") // Tell the browser we're sending JSON back, not plain text
        json.NewEncoder(w).Encode(response) // Convert the Go struct into JSON and write it to the response
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

