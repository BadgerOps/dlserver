package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"unicode/utf8"

	_ "github.com/mattn/go-sqlite3"
)

const KeyServerAddr = "serverAddr"

func catch(err error) {
	if err != nil {
		slog.Error("Application error: %s", err)
		//panic(err)
	}
}

func catchHTTPerr(err error, w http.ResponseWriter) {
	if err != nil {
		slog.Error("HTTP Error: %s", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// do we need this duplicate struct? maybe kill it.
type JobPageData struct {
	Jobs []Job
}

type Job struct {
	Name string `json:"name"`
	Time string `json:"time"`
	URL  string `json:"url"`
}

func main() {
	// The "/" pattern matches everything, so we need to check
	// that we're at the root here.
	// currently unused
	// data := PageData{
	// 	Title:   "My Page Title",
	// 	Header:  "Welcome to my website!",
	// 	Content: "This is some content.",
	// }
	// RenderTemplate(w, data)
	// Set up the CORS middleware
	// Allow all origins
	// Allow these HTTP methods
	// Allow all headers
	// Wrap the mux with the cors handler
	// Start the server
	err := serveHTTP()
	catchHTTPerr(err, nil)
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
	// TODO: fix this up - its kind of a mess and doesn't do what I expected it to... but it _does_ run.
	job := Job{}
	err := json.NewDecoder(body).Decode(&job)
	if err != nil {
		err := fmt.Errorf("Error parsing JSON: %s", err)
		slog.Error("Error: %s", err)
		return job, err
	}
	// validate the job name is not empty
	if utf8.RuneCountInString(job.Name) == 0 {
		err = errors.Join(fmt.Errorf("Job name cannot be empty"))
		slog.Error("Error: %s", err)
	}
	// validate the job name is not invalid characters
	if !utf8.ValidString(job.Name) {
		err = errors.Join(fmt.Errorf("Invalid characters in job name"))
		slog.Error("Error: %s", err)
	}
	// validate the URL is valid
	_, err = url.ParseRequestURI(job.URL)
	if err != nil {
		err = errors.Join(fmt.Errorf("Invalid URL: %s", job.URL))
		slog.Error("Error: %s", err)
	}
	return job, err
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
