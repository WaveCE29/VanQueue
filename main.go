package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var Db *sql.DB

const Path = "van"
const apiBasePath = "/api"

type Van struct {
	VanID     int       `json:"idvan"`
	VanName   string    `json:"vanName"`
	VanDriver string    `json:"vanDriver"`
	VanQueue  time.Time `json:"vanQueue"`
}

func SetupDB() {
	var err error
	Db, err = sql.Open("mysql", "root:root@tcp(192.168.100.81:3304)/test")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(Db)
	Db.SetConnMaxIdleTime(time.Minute * 3)
	Db.SetMaxOpenConns(10)
	Db.SetMaxIdleConns(10)
}

func corsMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Method", "GET")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Length, Content-Type, Application/JSON")
		handler.ServeHTTP(w, r)
	})
}

func getVans() ([]Van, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	results, err := Db.QueryContext(ctx, "SELECT * FROM van")
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	defer results.Close()
	vans := make([]Van, 0)
	for results.Next() {
		var van Van
		var vanQueuestr string
		err = results.Scan(&van.VanID, &van.VanName, &van.VanDriver, &vanQueuestr)
		if err != nil {
			log.Println(err.Error())
			return nil, err
		}
		van.VanQueue, err = time.Parse("2006-01-02 15:04:05", vanQueuestr)
		if err != nil {
			log.Println(err.Error())
			return nil, err
		}

		vans = append(vans, van)
	}
	return vans, nil
}

func getVan(vanid int) (*Van, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	row := Db.QueryRowContext(ctx, "SELECT * FROM van WHERE idvan = ?", vanid)
	van := &Van{}
	var vanQueueStr string // use a string to scan the datetime
	err := row.Scan(&van.VanID, &van.VanName, &van.VanDriver, &vanQueueStr)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	// Parse the string to time.Time
	van.VanQueue, err = time.Parse("2006-01-02 15:04:05", vanQueueStr)
	if err != nil {
		return nil, err
	}

	return van, nil

}

func handlerVans(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		vansList, err := getVans()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		j, err := json.Marshal(vansList)
		if err != nil {
			log.Fatal(err)
		}
		_, err = w.Write(j)
		if err != nil {
			log.Fatal(err)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}

}

func handlerVan(w http.ResponseWriter, r *http.Request) {
	urlPathSegment := strings.Split(r.URL.Path, fmt.Sprintf("%s/", Path))
	if len(urlPathSegment[1:]) > 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	vanID, err := strconv.Atoi(urlPathSegment[len(urlPathSegment)-1])
	if err != nil {
		log.Fatal(err)
		w.WriteHeader(http.StatusNotFound)
	}
	switch r.Method {
	case http.MethodGet:
		van, err := getVan(vanID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if van == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		j, err := json.Marshal(van)
		if err != nil {
			log.Fatal(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, err = w.Write(j)
		if err != nil {
			log.Fatal(err)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func SetupRoutes(apiBasePath string) {
	vansHandler := http.HandlerFunc(handlerVans)
	vanHandler := http.HandlerFunc(handlerVan)
	http.Handle(fmt.Sprintf("%s/%s", apiBasePath, Path), corsMiddleware(vansHandler))
	http.Handle(fmt.Sprintf("%s/%s/", apiBasePath, Path), corsMiddleware(vanHandler))

}

func main() {
	SetupDB()
	SetupRoutes(apiBasePath)
	http.ListenAndServe(":5000", nil)

}
