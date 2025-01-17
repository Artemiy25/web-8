package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "postgres"
	dbname   = "sandbox"
)

type DatabaseProvider struct {
	db *sql.DB
}

type Counter struct {
	ID    int `json:"id"`
	Value int `json:"value"`
}

func main() {

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Successfully connected to the database!")

	dbProvider := &DatabaseProvider{db: db}

	err = dbProvider.initializeCounter()
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/count", func(w http.ResponseWriter, r *http.Request) {
		countHandler(w, r, dbProvider)
	})

	err = http.ListenAndServe(":3333", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func countHandler(w http.ResponseWriter, r *http.Request, dbProvider *DatabaseProvider) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	switch r.Method {
	case http.MethodGet:

		counter, err := dbProvider.GetCounter()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if counter == nil {
			http.Error(w, "Counter not found", http.StatusNotFound)
			return
		}
		if err := json.NewEncoder(w).Encode(counter); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

	case http.MethodPost:

		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		countStr := r.FormValue("count")
		count, err := strconv.Atoi(countStr)
		if err != nil {
			http.Error(w, "Count must be a number", http.StatusBadRequest)
			return
		}

		err = dbProvider.IncreaseCounter(count)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Counter increased by %d", count)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (dp *DatabaseProvider) GetCounter() (*Counter, error) {
	query := "SELECT id, value FROM counter LIMIT 1"
	row := dp.db.QueryRow(query)

	var counter Counter
	err := row.Scan(&counter.ID, &counter.Value)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &counter, nil
}

func (dp *DatabaseProvider) IncreaseCounter(value int) error {
	query := "UPDATE counter SET value = value + $1 WHERE id = 1"
	_, err := dp.db.Exec(query, value)
	return err
}

func (dp *DatabaseProvider) initializeCounter() error {
	var count Counter

	query := "SELECT id FROM counter LIMIT 1"
	err := dp.db.QueryRow(query).Scan(&count.ID)
	if err == sql.ErrNoRows {

		insertQuery := "INSERT INTO counter (value) VALUES ($1)"
		_, err := dp.db.Exec(insertQuery, 0)
		if err != nil {
			return err
		}
	}

	return nil
}