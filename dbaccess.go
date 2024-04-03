package main

import (
	"database/sql"
	"log"
)

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
