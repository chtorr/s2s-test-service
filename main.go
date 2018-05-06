package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

var serviceName string

var (
	listenPort         = "8080"
	egressHTTPPort     = "9000"
	egressPostgresPort = "9100"
)

const (
	configKeyServiceName        = "SERVICE_NAME"
	configKeyServicePort        = "SERVICE_PORT"
	configKeyEgressHTTPPort     = "EGRESS_HTTP_PORT"
	configKeyEgressPostgresPort = "EGRESS_POSTGRES_PORT"
)

func main() {

	// mandatory service name
	serviceName = os.Getenv(configKeyServiceName)
	if serviceName == "" {
		log.Fatalf("%s env var must be set", configKeyServiceName)
	}

	// override listen port for this service
	port := os.Getenv(configKeyServicePort)
	if port != "" {
		listenPort = port
	}

	// override egress http port
	port = os.Getenv(configKeyEgressHTTPPort)
	if port != "" {
		egressHTTPPort = port
	}

	// override egress postgres port
	port = os.Getenv(configKeyEgressPostgresPort)
	if port != "" {
		egressPostgresPort = port
	}

	conninfo := fmt.Sprintf("postgres://postgres:postgres@127.0.0.1:%s/postgres?sslmode=disable", egressPostgresPort)
	fmt.Println(conninfo)
	db, err := sql.Open("postgres", conninfo)
	if err != nil {
		log.Fatalf("Failed opening db: %v", err)
	}

	log.Printf("Starting server for %s...", serviceName)

	http.HandleFunc("/ping", ping)
	http.Handle("/pingdb", pingdb(db))

	serverPort := fmt.Sprintf(":%s", listenPort)
	err = http.ListenAndServe(serverPort, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func pingdb(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := db.Exec("SELECT 1")
		if err != nil {
			http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "pong")
	})
}

func ping(w http.ResponseWriter, r *http.Request) {
	service := r.URL.Query().Get("service")
	if service != "" {
		returnRemote(w, r)
		return
	}

	returnLocal(w, r)
	return
}

func returnLocal(w http.ResponseWriter, r *http.Request) {
	log.Printf("handing /ping for %s", serviceName)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "pong")
}

// send an HTTP request to the egress port with the host header set to the specified value
func returnRemote(w http.ResponseWriter, r *http.Request) {
	service := r.URL.Query().Get("service")
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "ping"
	}
	log.Printf("%s passing request to %s/%s", serviceName, service, path)

	client := &http.Client{}
	u := fmt.Sprintf("http://127.0.0.1:%s/%s", egressHTTPPort, path)
	req, err := http.NewRequest("GET", u, nil)
	req.Host = service

	resp, err := client.Do(req)
	if err != nil {
		msg := fmt.Sprintf("%s error pinging %s: %v", serviceName, service, err)
		log.Println(msg)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, msg)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		msg := fmt.Sprintf("%s error pinging %s: %v", serviceName, service, err)
		log.Println(msg)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, msg)
		return
	}

	w.WriteHeader(resp.StatusCode)
	fmt.Fprintf(w, "response from %s: (%s)", service, string(body))
	return
}
