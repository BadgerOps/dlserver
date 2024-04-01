package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"text/template"
	"unicode/utf8"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/cors"
)

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
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		// The "/" pattern matches everything, so we need to check
		// that we're at the root here.
		if req.URL.Path != "/" {
			http.NotFound(w, req)
			return
		}
		data := PageData{
			Title:   "My Page Title",
			Header:  "Welcome to my website!",
			Content: "This is some content.",
		}

		RenderTemplate(w, data)
	})
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
	slog.Info("Web server started on port 8080")
	http.ListenAndServe(":8080", handler)
}

func GetJobs(w http.ResponseWriter, r *http.Request) {
	// Get the list of jobs in json format
	jobs, err := GetScheduledJobs()
	if err != nil {
		slog.Error("Error: %s", err)
		return
	}

	if err != nil {
		slog.Error("Error: %s", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	RenderJobsTmpl(w, jobs)
}

func RenderJobsTmpl(w http.ResponseWriter, jobs []byte) {
	// set up var for data, from the Job struct
	var data []Job
	// unmarshal the jobs into the data var
	err := json.Unmarshal(jobs, &data)
	catchHTTPerr(err, w)
	// render the template for joblist.html
	slog.Debug("Rendering joblist.html")
	tmpl, err := template.ParseFiles("joblist.html")
	catchHTTPerr(err, w)

	err = tmpl.Execute(w, data)
	catchHTTPerr(err, w)
}

func ScheduleJob(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// mux.Body should look like {"name": "job name", "time": "time to run the job", "url": "file to download"}
	// create a dummy io.reader for testing
	// testreader := strings.NewReader(`{"name": "Foobar Job", "time": "1970-01-01 13:37:00", "url": "https://blog.badgerops.net/content/images/2020/03/badger.png"}`)
	// first, convert the URL encoded string to a byte array
	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error: %s", err)
	}
	// then, convert the byte array to an io.Reader
	r.Body = io.NopCloser(bytes.NewReader(body))
	// finally, parse the io.Reader
	job, err := ParseJob(r.Body)
	if err != nil {
		slog.Error("Error: %s", err)
		// raise error to the user so they can correct the job
		w.Write([]byte(fmt.Sprintf("Error: %s", err)))
		// then break out of the function instead of continuing
		return
	}

	// check to see if the job already exists using CheckDupJobs, otherwise save the new job to the DB
	existingJob, err := CheckDupJobs(job)
	catch(err)
	if existingJob != nil {
		w.Write([]byte(fmt.Sprintf("Existing job found: %s", existingJob)))
	}
	if existingJob == nil {
		SavetoDB(job)
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
