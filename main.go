package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type FormFields struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"https://whisperly.com", "http://localhost:3000"}, // Use this to allow specific origin hosts
	}))

	client, err := makeMongoConnection()
	if err != nil {
		panic(err)
	}
	defer client.Disconnect(context.Background())
	collection := client.Database("whisperly-waiting-list").Collection("form-fields")
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World!"))
	})
	r.Post("/submit", func(w http.ResponseWriter, r *http.Request) {
		form_fields := FormFields{}

		err := json.NewDecoder(r.Body).Decode(&form_fields)
		if err != nil {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}
		if form_fields.Name == "" || form_fields.Email == "" {
			http.Error(w, "Name and Email are required", http.StatusBadRequest)
			return
		}
		var existing FormFields
		err = collection.FindOne(context.Background(), map[string]string{"email": form_fields.Email}).Decode(&existing)
		if err == nil {
			http.Error(w, "Email already exists", http.StatusConflict)
			return
		}

		_, err = collection.InsertOne(context.Background(), form_fields)
		if err != nil {
			http.Error(w, "Failed to save data", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("Form submitted successfully"))
	})
	http.ListenAndServe(":8080", r)
}

// make a mongodb connection
func makeMongoConnection() (*mongo.Client, error) {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		return nil, errors.New("MONGO_URI environment variable is not set")
	}
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		return nil, err
	}
	err = client.Ping(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	return client, nil
}
