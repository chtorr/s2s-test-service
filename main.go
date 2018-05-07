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

	http.Handle("/ping", ping())
	http.Handle("/pingdb", pingDb(db))

	http.Handle("/ping_remote", pingRemote())
	http.Handle("/pingdb_remote", pingDbRemote())

	serverPort := fmt.Sprintf(":%s", listenPort)
	err = http.ListenAndServe(serverPort, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func pingDb(db *sql.DB) http.Handler {
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

func ping() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("handing /ping for %s", serviceName)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "pong")
	})
}

func pingRemote() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		service := r.URL.Query().Get("service")
		path := "ping"

		log.Printf("%s passing request to %s/%s", serviceName, service, path)

		code, body := callRemote(service, path)
		w.WriteHeader(code)
		fmt.Fprintf(w, body)
	})
}

func pingDbRemote() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		service := r.URL.Query().Get("service")
		path := "pingdb"

		log.Printf("%s passing request to %s/%s", serviceName, service, path)

		code, body := callRemote(service, path)
		w.WriteHeader(code)
		fmt.Fprintf(w, body)
	})
}

// send an HTTP request to the egress port with the host header set to the specified value
func callRemote(service, path string) (int, string) {

	client := &http.Client{}
	u := fmt.Sprintf("http://127.0.0.1:%s/%s", egressHTTPPort, path)
	req, err := http.NewRequest("GET", u, nil)
	req.Host = service

	resp, err := client.Do(req)
	if err != nil {
		return http.StatusInternalServerError, fmt.Sprintf("%s error calling %s/%s: %v", serviceName, service, path, err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return http.StatusInternalServerError, fmt.Sprintf("%s error calling %s/%s: %v", serviceName, service, path, err)
	}

	return resp.StatusCode, fmt.Sprintf("response from %s: (%s)", service, string(body))
}
