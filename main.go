package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

var VERSION = "dev"

type Server struct {
	Host  string
	Port  int
	State string
}

func (s *Server) Serve() {
	r := mux.NewRouter().StrictSlash(true)
	address := fmt.Sprintf("%s:%d", s.Host, s.Port)

	r.HandleFunc("/status", s.Status).Methods("GET")
	r.HandleFunc("/jobs", s.Run).Methods("POST")
	r.HandleFunc("/stop", s.Stop).Methods("POST")
	r.HandleFunc("/jobs/{job_id}/log", s.Logs).Methods("GET")

	fmt.Printf("Agent %s listening on https://%s\n", VERSION, address)

	loggedRouter := handlers.LoggingHandler(os.Stdout, r)

	log.Fatal(http.ListenAndServeTLS(address, "server.crt", "server.key", loggedRouter))
}

func (s *Server) Status(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(400)
	m := make(map[string]interface{})

	m["state"] = s.State
	m["version"] = VERSION

	jsonString, _ := json.Marshal(m)

	fmt.Fprintf(w, string(jsonString))
}

func (s *Server) Logs(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain")

	// not yet supported
	// start_from = params[:start_from].to_i || 0

	logfile, err := os.Open("/tmp/job_log.json")

	if err != nil {
		w.WriteHeader(404)
		return
	}
	defer logfile.Close()

	io.Copy(w, logfile)
}

func (s *Server) Run(w http.ResponseWriter, r *http.Request) {
	if s.State != "waiting for job" {
		w.WriteHeader(422)
		fmt.Fprintf(w, `{"message": "a job is already running"}`)
		return
	}

	s.State = "received-job"

	jobRequest := JobRequest{}

	err := json.NewDecoder(r.Body).Decode(&jobRequest)

	if err != nil {
		fmt.Fprintf(w, `{"message": "%s"}`, err)
		return
	}

	job := Job{Request: jobRequest}

	s.State = "job-started"

	go job.Run()
}

func (s *Server) Stop(w http.ResponseWriter, r *http.Request) {
	s.unsuported(w)
}

func (s *Server) unsuported(w http.ResponseWriter) {
	w.WriteHeader(400)
	fmt.Fprintf(w, `{"message": "not supported"}`)
}

func main() {
	action := os.Args[1]

	switch action {
	case "serve":
		server := Server{Host: "0.0.0.0", Port: 8000, State: "waiting for job"}
		server.Serve()

	case "run":
		job, err := NewJobFromYaml(os.Args[2])

		if err != nil {
			panic(err)
		}

		job.Run()

	case "version":
		fmt.Println(VERSION)
	}
}
