package main

import (
	"html/template"
	"log"
	"net/http"
)

type PageData struct {
	Title   string
	Header  string
	Content string
}

func RenderTemplate(w http.ResponseWriter, data PageData) {
	tmpl, err := template.ParseFiles("template.html")
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
