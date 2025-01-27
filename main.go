package main

import (
    "database/sql"
    "encoding/json"
    "html/template"
    "log"
    "net/http"
    "os"

    _ "github.com/lib/pq" // PostgreSQL driver for NeonDB
    "github.com/joho/godotenv" // Load environment variables from .env file
)

type Post struct {
    ID      int    `json:"id"`
    Title   string `json:"title"`
    Content string `json:"content"`
}

var (
    db       *sql.DB
    tmpl     = template.Must(template.ParseGlob("templates/*.html"))
    dbConfig string
)

func main() {
    var err error

    // Load environment variables from .env file
    if err := godotenv.Load(); err != nil {
        log.Fatalf("Error loading .env file: %v", err)
    }

    // Get database URL from the environment
    dbConfig = os.Getenv("DB_URL")
    if dbConfig == "" {
        log.Fatal("DB_URL is not set in the environment variables")
    }

    // Initialize the database connection
    db, err = sql.Open("postgres", dbConfig)
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    defer db.Close()

    // Ensure the database is reachable
    if err = db.Ping(); err != nil {
        log.Fatalf("Cannot ping the database: %v", err)
    }

    // Set up routes
    http.HandleFunc("/", homeHandler)
    http.HandleFunc("/post/new", newPostHandler)
    http.HandleFunc("/post/create", createPostHandler)
    http.HandleFunc("/post/view", viewPostHandler)

    // API routes
    http.HandleFunc("/api/posts", apiGetPostsHandler)
    http.HandleFunc("/api/post", apiCreatePostHandler)

    // Start the server
    log.Println("Starting server on :8080...")
    if err := http.ListenAndServe(":8080", nil); err != nil {
        log.Fatalf("Server failed to start: %v", err)
    }
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
    rows, err := db.Query("SELECT id, title, content FROM posts")
    if err != nil {
        http.Error(w, "Failed to fetch posts", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var posts []Post
    for rows.Next() {
        var post Post
        if err := rows.Scan(&post.ID, &post.Title, &post.Content); err != nil {
            http.Error(w, "Error scanning posts", http.StatusInternalServerError)
            return
        }
        posts = append(posts, post)
    }

    tmpl.ExecuteTemplate(w, "home.html", posts)
}

func newPostHandler(w http.ResponseWriter, r *http.Request) {
    tmpl.ExecuteTemplate(w, "new.html", nil)
}

func createPostHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
        return
    }

    title := r.FormValue("title")
    content := r.FormValue("content")

    _, err := db.Exec("INSERT INTO posts (title, content) VALUES ($1, $2)", title, content)
    if err != nil {
        http.Error(w, "Failed to create post", http.StatusInternalServerError)
        return
    }

    http.Redirect(w, r, "/", http.StatusSeeOther)
}

func viewPostHandler(w http.ResponseWriter, r *http.Request) {
    id := r.URL.Query().Get("id")

    var post Post
    if err := db.QueryRow("SELECT id, title, content FROM posts WHERE id = $1", id).Scan(&post.ID, &post.Title, &post.Content); err != nil {
        http.Error(w, "Post not found", http.StatusNotFound)
        return
    }

    tmpl.ExecuteTemplate(w, "view.html", post)
}

// API Handler to fetch all posts
func apiGetPostsHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
        return
    }

    rows, err := db.Query("SELECT id, title, content FROM posts")
    if err != nil {
        http.Error(w, "Failed to fetch posts", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var posts []Post
    for rows.Next() {
        var post Post
        if err := rows.Scan(&post.ID, &post.Title, &post.Content); err != nil {
            http.Error(w, "Error scanning posts", http.StatusInternalServerError)
            return
        }
        posts = append(posts, post)
    }

    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(posts); err != nil {
        http.Error(w, "Failed to encode posts", http.StatusInternalServerError)
    }
}

// API Handler to create a new post
func apiCreatePostHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
        return
    }

    var post Post
    decoder := json.NewDecoder(r.Body)
    if err := decoder.Decode(&post); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    _, err := db.Exec("INSERT INTO posts (title, content) VALUES ($1, $2)", post.Title, post.Content)
    if err != nil {
        http.Error(w, "Failed to create post", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusCreated)
    if err := json.NewEncoder(w).Encode(post); err != nil {
        http.Error(w, "Failed to encode created post", http.StatusInternalServerError)
    }
}
