package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
)

var service_name string

var (
	listenPort     = 8080
	egressHTTPPort = 9000
)

const (
	configKeyServiceName = "SERVICE_NAME"
	configKeyServicePort = "SERVICE_PORT"
	configKeyEgressPort  = "EGRESS_HTTP_PORT"
)

func main() {

	// mandatory service name
	service_name = os.Getenv(configKeyServiceName)
	if service_name == "" {
		log.Fatalf("%s env var must be set", configKeyServiceName)
	}

	// override listen port for this service
	service_port := os.Getenv(configKeyServicePort)
	if service_port != "" {
		port, err := strconv.Atoi(service_port)
		if err != nil {
			log.Fatalf("failed converting %s to int: %v", configKeyServicePort, err)
		}
		listenPort = port
	}

	// override egress port
	egress_port := os.Getenv(configKeyEgressPort)
	if egress_port != "" {
		port, err := strconv.Atoi(egress_port)
		if err != nil {
			log.Fatalf("failed converting %s to int: %v", configKeyEgressPort, err)
		}
		listenPort = port
	}

	log.Printf("Starting server for %s...", service_name)

	http.HandleFunc("/ping", ping)

	err := http.ListenAndServe(strconv.Itoa(listenPort), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
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
	log.Printf("handing /ping for %s", service_name)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "pong")
}

// send an HTTP request to the egress port with the host header set to the specified value
func returnRemote(w http.ResponseWriter, r *http.Request) {
	service := r.URL.Query().Get("service")
	log.Printf("%s passing request to %s", service_name, service)

	client := &http.Client{}
	u := fmt.Sprintf("http://127.0.0.1:%d/ping", egressHTTPPort)
	req, err := http.NewRequest("GET", u, nil)
	req.Host = service

	resp, err := client.Do(req)
	if err != nil {
		msg := fmt.Sprintf("%s error pinging %s: %v", service_name, service, err)
		log.Println(msg)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, msg)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		msg := fmt.Sprintf("%s error pinging %s: %v", service_name, service, err)
		log.Println(msg)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, msg)
		return
	}

	w.WriteHeader(resp.StatusCode)
	w.Write(body)
	return
}
