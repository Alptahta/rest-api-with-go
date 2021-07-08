package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

var backupRating *int

const errMarshaling string = "Error Happened while marshaling"

type item struct {
	Value string `json:"value"`
}

type datastore struct {
	m map[string]item
}

type dictionaryHandler struct {
	store *datastore
}

func (h *dictionaryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	log.Println(r.URL.Path)
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/get":
		h.Get(w, r)
		return
	case r.Method == http.MethodPost && r.URL.Path == "/set":
		h.Create(w, r)
		return
	default:
		notFound(w)
		return
	}
}

func (h *dictionaryHandler) Get(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")

	v, ok := h.store.m[key]

	result := fmt.Sprintf("Item %s has been searched.", key)

	if !ok {
		notFound(w)
		return
	}

	jsonBytes, err := json.Marshal(v)
	if err != nil {
		internalServerError(w)
		return
	}
	w.WriteHeader(http.StatusOK)
	if _, err = w.Write(jsonBytes); err != nil {
		log.Fatal(err)
	}

	log.Println(result)
}

func (h *dictionaryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var i item

	key := r.URL.Query().Get("key")
	value := r.URL.Query().Get("value")

	if _, ok := h.store.m[key]; ok {
		keyAlreadyExist(w)
		return
	}

	i.Value = value
	h.store.m[key] = i

	result := fmt.Sprintf("Item %s has been created with value %s.", key, value)

	jsonBytes, err := json.Marshal(i)
	if err != nil {
		internalServerError(w)
		return
	}
	w.WriteHeader(http.StatusOK)
	if _, err = w.Write(jsonBytes); err != nil {
		log.Fatal(err)
	}

	log.Println(result)
}

func internalServerError(w http.ResponseWriter) {
	error := "Internal Server Error"
	jsonBytes, err := json.Marshal(error)
	if err != nil {
		log.Fatal(errMarshaling)
		return
	}

	w.WriteHeader(http.StatusInternalServerError)
	if _, err = w.Write(jsonBytes); err != nil {
		log.Fatal(err)
	}
}

func notFound(w http.ResponseWriter) {
	error := "Not Found"
	jsonBytes, err := json.Marshal(error)
	if err != nil {
		log.Fatal(errMarshaling)
		return
	}

	w.WriteHeader(http.StatusNotFound)
	if _, err = w.Write(jsonBytes); err != nil {
		log.Fatal(err)
	}
}

func keyAlreadyExist(w http.ResponseWriter) {
	error := "Key Already Exist"
	jsonBytes, err := json.Marshal(error)
	if err != nil {
		log.Fatal(errMarshaling)
		return
	}

	w.WriteHeader(http.StatusConflict)
	if _, err = w.Write(jsonBytes); err != nil {
		log.Fatal(err)
	}
}

func handleRequests(dictionaryH *dictionaryHandler) {
	mux := http.NewServeMux()

	mux.Handle("/set", dictionaryH)
	mux.Handle("/get", dictionaryH)
	log.Fatal(http.ListenAndServe(":9001", mux))
}

func setDateString() string {
	// Use layout string for time format.
	const layout = "01-02-2006 2:3:5"
	// Place now in the string.
	t := time.Now()
	return "" + t.Format(layout) + "-db.txt"
}

//Back-up data to file
func backUp(input <-chan map[string]item) {

	for v := range input {
		b, err := json.Marshal(v)
		if err != nil {
			fmt.Println("error:", err)
		}

		name := setDateString()
		f, err := os.Create("/tmp/" + name)
		if err != nil {
			panic(err)
		}

		n2, err := f.Write(b)
		if err != nil {
			panic(err)
		}

		fmt.Printf("wrote %d bytes\n", n2)
		log.Printf("File created under /tmp with name %s", name)
		f.Close()

	}

}

func main() {
	const InitialBackUpRate int = 5
	backupRating = flag.Int("backUp", InitialBackUpRate, "How many seconds should pass for after last backup to create new backup file. Default is 5")
	flag.Parse()

	dictionaryH := &dictionaryHandler{
		store: &datastore{
			m: map[string]item{},
		},
	}

	go handleRequests(dictionaryH)

	ch := make(chan map[string]item)
	go backUp(ch)

	for range time.Tick(time.Second * time.Duration(*backupRating)) {
		ch <- dictionaryH.store.m
	}

}
