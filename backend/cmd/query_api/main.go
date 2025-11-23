package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/honeynet/ochi/backend/handlers"
	"github.com/honeynet/ochi/backend/repos"
	"github.com/honeynet/ochi/backend/services"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sqlx.Connect("sqlite3", "test.db")
	if err != nil {
		log.Fatal(err)
	}

	queryRepo, err := repos.NewQueryRepo(db)
	if err != nil {
		log.Fatal(err)
	}

	queryService := services.NewQueryService(queryRepo)
	queryHandler := handlers.NewQueryHandler(queryService)

	r := mux.NewRouter()
	r.HandleFunc("/queries", queryHandler.CreateQueryHandler).Methods("POST")
	r.HandleFunc("/queries/{id}", queryHandler.GetQueryByIDHandler).Methods("GET")
	r.HandleFunc("/queries/{id}", queryHandler.UpdateQueryHandler).Methods("PUT")
	r.HandleFunc("/queries/{id}", queryHandler.DeleteQueryHandler).Methods("DELETE")
	r.HandleFunc("/queries/owner/{owner_id}", queryHandler.FindQueriesByOwnerHandler).Methods("GET")

	log.Println("Server running at :8080")
	http.ListenAndServe(":8080", r)
}
