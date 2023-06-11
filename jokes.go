package main

type Joke struct {
	Id   string `json:"id"`
	Joke string `json:"joke"`
}

type Jokes []Joke

type NoJoke struct {
	Status  string `json:"status"`
	Error   string `json:"error"`
	Message string `json:"message"`
}