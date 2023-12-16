package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	_ "github.com/jackc/pgx/v4/stdlib"
	log "github.com/sirupsen/logrus"
)

var db *sql.DB

const (
	logFileName = "log.log"
	serverPort  = 7070

	dbHost     = "localhost"
	dbPort     = 8080
	dbUser     = "postgres"
	dbPassword = "pass"
	dbName     = "postgres"
	dbSslMode  = "prefer"
)

func main() {
	log.SetFormatter(&log.JSONFormatter{})

	// open log file
	file, errr := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if errr == nil {
		// Set the file as the output for logging
		log.SetOutput(file)
		log.WithFields(log.Fields{
			"main": "opening log file",
		}).Info("File opened successful")
		defer file.Close()
	} else {
		log.Fatal("Failed to open the log file.", errr)
	}

	initDB()

	server := mux.NewRouter()

	// adding hendling methods

	server.HandleFunc("/ping", pingHandler).Methods("GET")

	server.HandleFunc("/shorten", shortenHandler).Methods("POST")

	server.HandleFunc("/{shortURL}", GetOriginalURLHandler).Methods("GET")

	// start the server

	http.Handle("/", server)

	port := fmt.Sprintf(":%d", serverPort)

	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.WithFields(log.Fields{
			"main": "starting server",
		}).Info("Error starting server:", err)
	}
}

func pingHandler(w http.ResponseWriter, r *http.Request) {

	log.WithFields(log.Fields{
		"func": "pingHandler",
	}).Info("Received ping request.")

	w.Write([]byte("Pong! Server is up and running."))
}

func generateShortURL(originalURL string) string {
	hasher := sha256.New()
	hasher.Write([]byte(originalURL))
	hashBytes := hasher.Sum(nil)

	shortURLHash := hex.EncodeToString(hashBytes)[:8]

	shortURL := shortURLHash // http://short.url/%s
	return shortURL
}

func initDB() {
	var err error

	credential := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", dbUser, dbPassword, dbHost, dbPort, dbName, dbSslMode)
	db, err = sql.Open("pgx", credential)
	if err != nil {
		log.WithFields(log.Fields{
			"func": "initDB",
		}).Info("Error opening sql", err)
	}

	// Ping server
	err = db.Ping()
	if err != nil {
		log.WithFields(log.Fields{
			"func": "initDB",
		}).Info("Error pinging server", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS urls (
			original_url TEXT NOT NULL,
			short_url TEXT NOT NULL
		)
	`)

	if err != nil {
		log.WithFields(log.Fields{
			"func": "initDB",
		}).Info("Error creating table", err)
	}
}

func shortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.WithFields(log.Fields{
			"func": "shortenHandler",
		}).Info("Method not allowed")
		return
	}

	originalURL := r.FormValue("url")
	if originalURL == "" {
		http.Error(w, "Missing 'url' parameter", http.StatusBadRequest)
		log.WithFields(log.Fields{
			"func": "shortenHandler",
		}).Info("Missing 'url' parameter")
		return
	}

	shortURL := generateShortURL(originalURL)

	// Save original url and shorten url in db
	// curl -X POST -d "url=https://www.example.com" http://localhost:7070/shorten
	_, err := db.Exec("INSERT INTO urls (original_url, short_url) VALUES ($1, $2)", originalURL, shortURL)
	if err != nil {
		log.WithFields(log.Fields{
			"func": "shortenHandler",
		}).Info("Error inserting into database:", err)
		http.Error(w, "Failed to save URL to database", http.StatusInternalServerError)
		return
	}

	//result := "http://short.url/" + shortURL

	w.Write([]byte(shortURL))
}

func GetOriginalURLHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	shortURL := vars["shortURL"]

	// fmt.Println("======================")
	// fmt.Println(shortURL)

	// substr := "http://short.url/"

	// if strings.Contains(shortURL, substr) {
	// 	shortURL = strings.Replace(shortURL, substr, "", -1)
	// } else {
	// 	w.Write([]byte("Not correct shortURL"))
	// 	log.WithFields(log.Fields{
	// 		"func": "GetOriginalURLHandler",
	// 	}).Info("Not correct shortURL format")
	// 	return
	// }

	// fmt.Println(shortURL)

	//curl http://localhost:7070/67d709a6
	var originalURL string
	err := db.QueryRow("SELECT original_url FROM urls WHERE short_url = $1", shortURL).Scan(&originalURL)
	if err != nil {
		http.NotFound(w, r)
		log.WithFields(log.Fields{
			"func": "GetOriginalURLHandler",
		}).Info("URL not found")
		return
	}

	log.WithFields(log.Fields{
		"func": "GetOriginalURLHandler",
	}).Info("Original URL successful found")

	// Redirect to original URL
	// w.Header().Set("Location", originalURL)
	// w.WriteHeader(http.StatusMovedPermanently)
	// w.Write([]byte("Moved Permanently"))

	// Send original url
	w.Write([]byte(originalURL))
}
