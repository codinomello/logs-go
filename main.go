package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type LogEntry struct {
	Message   string    `bson:"message"`
	Timestamp time.Time `bson:"timestamp"`
}

var client *mongo.Client

func init() {
	// Conecta ao MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Verifica a conexão
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}

	fmt.Println("Connected to MongoDB!")
}

func main() {
	// Handler para salvar logs
	http.HandleFunc("/log", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Lê o corpo da requisição (o log)
		logMessage := r.FormValue("message")
		if logMessage == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "Log message is required")
			return
		}

		// Cria uma entrada de log
		logEntry := LogEntry{
			Message:   logMessage,
			Timestamp: time.Now(),
		}

		// Insere o log no MongoDB
		collection := client.Database("logsdb").Collection("logs")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := collection.InsertOne(ctx, logEntry)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, "Failed to save log to MongoDB")
			return
		}

		// Salva o log em uma coleção MongoDB
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Log received and saved to MongoDB")
	})

	http.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Consulta os logs no MongoDB
		collection := client.Database("logsdb").Collection("logs")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cursor, err := collection.Find(ctx, bson.M{})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, "Failed to fetch logs from MongoDB")
			return
		}
		defer cursor.Close(ctx)

		var logs []LogEntry
		if err := cursor.All(ctx, &logs); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, "Failed to decode logs")
			return
		}

		// Retorna os logs como JSON
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		for _, logEntry := range logs {
			fmt.Fprintf(w, "[%s] %s\n", logEntry.Timestamp.Format("2006-01-02 15:04:05"), logEntry.Message)
		}
	})

	fmt.Println("Server started at :8080")
	http.ListenAndServe(":8080", nil)
}
