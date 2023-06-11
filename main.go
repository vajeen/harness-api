package main

import (
	"context"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("index.html") // index.html for static index page
	if err != nil {
		log.Fatal(err)
	}
	w.Header().Set("Content-Type", "text/html") // set header
	tmpl.Execute(w, "")                         // serve the template
}

func getJokeByID(client *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)                                // get param value
		reply, err := client.Get(vars["id"]).Result()      // get value
		w.Header().Set("Content-Type", "application/json") // set header
		if err == redis.Nil {                              // if no results for key, return error json
			var resp = NoJoke{"500", "Internal Server Error", "Joke not availale"}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				panic(err)
			}
		} else { // return joke
			var resp = Joke{vars["id"], reply}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				panic(err)
			}
		}
	}

}

func getAllJokes(client *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json") //set header
		keys, err := client.Do("KEYS", "*").Result()       // get all keys
		var jokes Jokes
		if err != nil {
			HandleError(err)
		} else {
			for _, k := range keys.([]interface{}) { // iterate through keys
				reply, err := client.Get(k.(string)).Result() // get value for key
				HandleError(err)
				var joke = Joke{k.(string), reply}
				jokes = append(jokes, joke)
			}
			if err := json.NewEncoder(w).Encode(jokes); err != nil { // write http response
				panic(err)
			}
		}
	}
}

// Get a random joke
func getRandJoke(client *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json") // set header to json
		keys, err := client.RandomKey().Result()           // get a random key
		HandleError(err)
		reply, err := client.Get(keys).Result() // get value
		if err != nil {
			HandleError(err)
		} else {
			var joke = Joke{keys, reply}
			if err := json.NewEncoder(w).Encode(joke); err != nil { // write http response
				panic(err)
			}
		}
	}
}

func main() {
	// Redis server connection params
	var (
		host     = getEnv("REDIS_HOST", "localhost")
		port     = string(getEnv("REDIS_PORT", "6379"))
		password = getEnv("REDIS_PASSWORD", "")
	)

	// Define new redis client
	client := redis.NewClient(&redis.Options{
		Addr:     host + ":" + port,
		Password: password,
		DB:       0,
	})

	// Check connection errors
	_, err := client.Ping().Result()
	if err != nil {
		log.Fatal(err)
	}

	r := mux.NewRouter()

	// Define routes
	r.HandleFunc("/", indexHandler)
	r.HandleFunc("/list", getAllJokes(client))      // list all jokes
	r.HandleFunc("/joke/{id}", getJokeByID(client)) // get joe by ID
	r.HandleFunc("/rand", getRandJoke(client))      // get a random joke

	srv := &http.Server{
		Handler:      r,
		Addr:         ":8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Start Server
	go func() {
		log.Println("Starting Server")
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	// Graceful Shutdown
	waitForShutdown(srv)
}

func waitForShutdown(srv *http.Server) {
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Block until we receive our signal.
	<-interruptChan

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	srv.Shutdown(ctx)

	log.Println("Shutting down")
	os.Exit(0)
}

// getEnv wrapper
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// Error handler
func HandleError(err error) {
	if err != nil {
		panic(err)
	}
}
