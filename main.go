package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type item struct {
	Value string `json:"value"`
}

type datastore struct {
	m map[string]item
	*sync.RWMutex
}

type dictionaryHandler struct {
	store *datastore
}

type error struct {
	Error string `json:"error"`
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
		notFound(w, r)
		return
	}
}

func (h *dictionaryHandler) Get(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")

	h.store.RLock()
	v, ok := h.store.m[key]
	h.store.RUnlock()

	result := fmt.Sprintf("Item %s has been searched.", key)

	if !ok {
		notFound(w, r)
		return
	}

	jsonBytes, err := json.Marshal(v)
	if err != nil {
		internalServerError(w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)

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
	h.store.Lock()
	h.store.m[key] = i
	h.store.Unlock()

	result := fmt.Sprintf("Item %s has been created with value %s.", key, value)

	jsonBytes, err := json.Marshal(i)
	if err != nil {
		internalServerError(w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)

	log.Println(result)
}

func internalServerError(w http.ResponseWriter, r *http.Request) {
	error := error{"Internal Server Error"}
	jsonBytes, err := json.Marshal(error)
	if err != nil {
		log.Fatal("Error Happened while marshaling")
		return
	}

	w.WriteHeader(http.StatusInternalServerError)
	w.Write(jsonBytes)
}

func notFound(w http.ResponseWriter, r *http.Request) {
	error := error{"Not Found"}
	jsonBytes, err := json.Marshal(error)
	if err != nil {
		log.Fatal("Error Happened while marshaling")
		return
	}

	w.WriteHeader(http.StatusNotFound)
	w.Write(jsonBytes)
}

func keyAlreadyExist(w http.ResponseWriter) {
	error := error{"Key Already Exist"}
	jsonBytes, err := json.Marshal(error)
	if err != nil {
		log.Fatal("Error Happened while marshaling")
		return
	}

	w.WriteHeader(http.StatusConflict)
	w.Write(jsonBytes)
}

func handleRequests(dictionaryH *dictionaryHandler) {
	mux := http.NewServeMux()

	mux.Handle("/set", dictionaryH)
	mux.Handle("/get", dictionaryH)
	log.Fatal(http.ListenAndServe(":9001", mux))
}

func GetFilenameDate() string {
	// Use layout string for time format.
	const layout = "01-02-2006"
	// Place now in the string.
	t := time.Now()
	return "" + t.Format(layout) + "-db.txt"
}

func (d *datastore) backUp() {
	for range time.Tick(time.Second * 3) {
		go func() {
			fmt.Println(d.m)
			b, err := json.Marshal(d.m)
			if err != nil {
				fmt.Println("error:", err)
			}
			fmt.Println(len(b))
			// err2 := ioutil.WriteFile("/tmp/dat1", b, 0644)
			// if err2 != nil {
			// 	panic(err2)
			// }

			name := GetFilenameDate()
			f, err := os.Create("/tmp/" + name)
			if err != nil {
				panic(err)
			}
			n2, err := f.Write(b)
			if err != nil {
				panic(err)
			}
			fmt.Printf("wrote %d bytes\n", n2)
			defer f.Close()

		}()
	}
}

func main() {

	dictionaryH := &dictionaryHandler{
		store: &datastore{
			m:       map[string]item{},
			RWMutex: &sync.RWMutex{},
		},
	}

	go dictionaryH.store.backUp()

	handleRequests(dictionaryH)

}
