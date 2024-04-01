package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"unicode/utf8"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/cors"
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
	mux.HandleFunc("/schedule", ScheduleJob)

	// Set up the CORS middleware
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},                      // Allow all origins
		AllowedMethods: []string{"GET", "POST", "OPTIONS"}, // Allow these HTTP methods
		AllowedHeaders: []string{"*"},                      // Allow all headers
	})

	// Wrap the mux with the cors handler
	handler := c.Handler(mux)
	// Start the server
	http.ListenAndServe(":8080", handler)
	fmt.Println("Web server started on port 8080")
}

func GetJobs(w http.ResponseWriter, r *http.Request) {
	// Get the list of jobs
	jobs, err := GetScheduledJobs()
	if err != nil {
		log.Println(err)
		return
	}
	// Write the response
	// Set the content type to JSON and write the response
	w.Header().Set("Content-Type", "application/json")
	w.Write(jobs)
}

func ScheduleJob(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Parse the request body
	// mux.Body should look like {"name": "job name", "time": "time to run the job", "url": "file to download"}
	// create a dummy io.reader for testing
	//testreader := strings.NewReader(`{"name": "Foobar Job", "time": "1970-01-01 13:37:00", "url": "https://blog.badgerops.net/content/images/2020/03/badger.png"}`)
	// first, convert the URL encoded string to a byte array
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
	}
	// then, convert the byte array to an io.Reader
	r.Body = ioutil.NopCloser(bytes.NewReader(body))
	// finally, parse the io.Reader
	job, err := ParseJob(r.Body)
	if err != nil {
		log.Println(err)
		// raise error to the user so they can correct the job
		w.Write([]byte(fmt.Sprintf("Error: %s", err)))
	}

	// check to see if the job already exists using CheckDupJobs, otherwise save the new job to the DB

	existingJob, err := CheckDupJobs(job)
	if err != nil {
		fmt.Println(err)
	}
	if existingJob != nil {
		// Write the response
		w.Write([]byte(fmt.Sprintf("Existing job found: %s", existingJob)))
	}
	if existingJob == nil {
		SavetoDB(job)
		// Write the response
		w.Write([]byte(fmt.Sprintf("Job %s scheduled", job.Name)))
	}
}

func GetScheduledJobs() ([]byte, error) {
	// Get the list of jobs
	jobs := GetJobsFromDB()

	// Convert the jobs to a JSON response
	jobsJSON, err := json.Marshal(jobs)
	if err != nil {
		return nil, err
	}

	return jobsJSON, nil
}

func ParseJob(body io.Reader) (Job, error) {
	// Parse the JSON body
	job := Job{}
	err := json.NewDecoder(body).Decode(&job)
	if err != nil {
		err := fmt.Errorf("Error parsing JSON: %s", err)
		log.Println(err)
		return job, err
	}
	// validate the job name is UTF-8
	if !utf8.ValidString(job.Name) {
		err := fmt.Errorf("Invalid UTF-8 string: %s", job.Name)
		log.Println(err)
		return job, err
	}
	// validate the URL is valid
	_, err = url.ParseRequestURI(job.URL)
	if err != nil {
		err := fmt.Errorf("Invalid URL: %s", job.URL)
		log.Println(err)
		return job, err
	}
	return job, nil
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

// check for duplicate jobs
func CheckDupJobs(job Job) (*Job, error) {
	jobs := GetJobsFromDB()

	// Search for the job in the list of jobs
	for _, j := range jobs {
		if j.Name == job.Name {
			return &job, fmt.Errorf("%s", job.Name)
		}
	}

	return nil, nil
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
