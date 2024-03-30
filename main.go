package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

/*
This program creates a simple threaded webserver with RESTful API endpoints.
The first endpoint is a simple GET request that returns a JSON response of currently scheduled jobs
The second endpoint is a POST that allows the user to schedule a new job
*/

type Job struct {
	Name string `json:"name"`
	Time string `json:"time"`
	URL  string `json:"url"`
}

func main() {
	// http.Handle("/jobs", http.HandlerFunc(GetJobs))

	// http.HandleFunc("/bar", func(w http.ResponseWriter, r *http.Request) {
	// 	fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	// })

	// log.Fatal(http.ListenAndServe(":8080", nil))
	//http.ListenAndServe(":8080", nil)

	// Create a new router
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		// The "/" pattern matches everything, so we need to check
		// that we're at the root here.
		if req.URL.Path != "/" {
			http.NotFound(w, req)
			return
		}
		fmt.Fprintf(w, "Welcome to the home page!")
	})
	fmt.Println("starting up mux for /jobs")
	mux.HandleFunc("/getjobs", GetJobs)
	mux.HandleFunc("/shedule", ScheduleJob)

	// Start the server
	http.ListenAndServe(":8080", mux)
	fmt.Println("Web server started on port 8080")
}

func NewRouter() {
	panic("unimplemented")
}

func GetJobs(w http.ResponseWriter, r *http.Request) {
	// Get the list of jobs
	jobs := GetScheduledJobs()

	// Write the response
	w.Write(jobs)
}

func ScheduleJob(w http.ResponseWriter, r *http.Request) {
	// Parse the request body
	// mux.Body should look like {"name": "job name", "time": "time to run the job", "url": "file to download"}
	// create a dummy io.reader for testing
	testreader := strings.NewReader(`{"name": "job name", "time": "time to run the job", "url": "https://blog.badgerops.net/content/images/2020/03/badgemux.png"}`)
	job := ParseJob(testreader)

	// Schedule the job
	SavetoDB(job)

	// Write the response1
	w.Write([]byte("Job scheduled"))
}

func GetScheduledJobs() []byte {
	// Get the list of jobs
	jobs := GetJobsFromDB()

	// Convert the jobs to a JSON response
	jobsJSON, _ := json.Marshal(jobs)

	return jobsJSON
}

func ParseJob(body io.Reader) Job {
	// Parse the JSON body
	job := Job{}
	json.NewDecoder(body).Decode(&job)

	return job
}

// function to interact with local sqlite database
func GetJobsFromDB() []Job {
	// Connect to the database
	db := ConnectDB()

	// Get the list of jobs
	jobs := QueryJobs(db)

	// Close the database connection
	db.Close()

	return jobs
}

func SavetoDB(job Job) {
	// Connect to the database
	db := ConnectDB()

	// Insert the job into the database
	log.Printf("Saving %s to database", job.Name)
	_, err := db.Exec("INSERT INTO jobs (name, time, url) VALUES (?, ?, ?)", job.Name, job.Time, job.URL)
	if err != nil {
		log.Fatal(err)
	}

	// Close the database connection
	db.Close()
}

func ConnectDB() *sql.DB {
	// Open the database
	db, err := sql.Open("sqlite3", "./jobs.db")
	if err != nil {
		log.Fatal(err)
	}

	// Create the jobs table if it doesn't exist
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS jobs (id INTEGER PRIMARY KEY, name TEXT, time TEXT, url TEXT)")
	if err != nil {
		log.Fatal(err)
	}

	return db
}

func QueryJobs(db *sql.DB) []Job {
	// Query the database for the list of jobs
	rows, err := db.Query("SELECT name, time, url FROM jobs")
	if err != nil {
		log.Fatal(err)
	}

	// Create a slice to hold the jobs
	jobs := []Job{}

	// Iterate over the rows and add the jobs to the slice
	for rows.Next() {
		job := Job{}
		rows.Scan(&job.Name, &job.Time, &job.URL)
		jobs = append(jobs, job)
	}

	return jobs
}
