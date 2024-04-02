package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log/slog"
	"net/http"
	"text/template"

	"github.com/rs/cors"
)

func serveHTTP() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {

		if req.URL.Path != "/" {
			http.NotFound(w, req)
		}

	})
	mux.HandleFunc("/getjobs", GetJobs)
	mux.HandleFunc("/schedule", ScheduleJob)

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(mux)

	slog.Info("Web server started on port 8080")
	http.ListenAndServe(":8080", handler)
	return nil
}

func GetJobs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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
	RenderJobsTmpl(w, jobs, ctx)
}

func RenderJobsTmpl(w http.ResponseWriter, jobs []byte, ctx context.Context) {
	// set up var for data, from the Job struct
	var data []Job
	// unmarshal the jobs into the data var
	err := json.Unmarshal(jobs, &data)
	catchHTTPerr(err, w)
	// render the template for joblist.html
	slog.Debug("%s: Rendering joblist.html", ctx.Value(KeyServerAddr))
	tmpl, err := template.ParseFiles("joblist.html")
	catchHTTPerr(err, w)

	err = tmpl.Execute(w, data)
	catchHTTPerr(err, w)
}

// Function to schedule a job, reading in data from a template generated using the schedule.tmpl file and rendered by html/template
func ScheduleJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tmpl := template.Must(template.ParseFiles("schedule.html"))
	tmpl.Execute(w, nil)
	// parse the body of the request

	// mux.Body should look like {"name": "job name", "time": "time to run the job", "url": "file to download"}
	// create a dummy io.reader for testing
	// testreader := strings.NewReader(`{"name": "Foobar Job", "time": "1970-01-01 13:37:00", "url": "https://blog.badgerops.net/content/images/2020/03/badger.png"}`)
	// first, convert the URL encoded string to a byte array
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		slog.Error("%s: Error reading body: %s", ctx.Value(KeyServerAddr), err)
	}
	// then, convert the byte array to an io.Reader
	r.Body = ioutil.NopCloser(bytes.NewReader(body))
	// finally, parse the io.Reader
	job, err := ParseJob(r.Body)
	if err != nil {
		slog.Error("%s: Error: %s", ctx.Value(KeyServerAddr), err)
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

func renderScheduleTmpl() {
	// render the schedule.tmpl file

}
