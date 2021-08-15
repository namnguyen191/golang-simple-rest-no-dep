package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Fish struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Environment string `json:"environment,omitempty"`
	MaxLength   int    `json:"max_length,omitempty"`
}

type fishesHandler struct {
	sync.Mutex
	db map[string]Fish
}

func newFishesHander() *fishesHandler {
	return &fishesHandler{
		db: map[string]Fish{},
	}
}

func (h *fishesHandler) getAllFishes(w http.ResponseWriter, r *http.Request) {
	var fishes []Fish

	h.Lock()
	for _, fish := range h.db {
		fishes = append(fishes, fish)
	}
	h.Unlock()

	jsonBytes, err := json.Marshal(fishes)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	w.Header().Add("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func (h *fishesHandler) getRandomCoaster(w http.ResponseWriter, r *http.Request) {
	ids := make([]string, len(h.db))

	h.Lock()
	i := 0
	for id := range h.db {
		ids[i] = id
		i++
	}
	h.Unlock()

	var target string
	if len(ids) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if len(ids) == 1 {
		target = ids[0]
	} else {
		rand.Seed(time.Now().UnixNano())
		target = ids[rand.Intn(len(ids))]
	}

	w.Header().Add("location", fmt.Sprintf("/fishes/%s", target))
	w.WriteHeader(http.StatusFound)
}

func (h *fishesHandler) getFish(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.String(), "/")

	if len(parts) != 3 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if parts[2] == "random" {
		h.getRandomCoaster(w, r)
		return
	}

	h.Lock()
	defer h.Unlock()
	foundFish, ok := h.db[parts[2]]

	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	jsonBytes, err := json.Marshal(foundFish)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Add("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func (h *fishesHandler) addNewFish(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	ct := r.Header.Get("content-type")
	if ct != "application/json" {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		w.Write([]byte(fmt.Sprintf("need content-type 'application/json' but got '%s'", ct)))
		return
	}

	var fish Fish
	err = json.Unmarshal(bodyBytes, &fish)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
	}

	fish.ID = fmt.Sprintf("%d", time.Now().UnixNano())

	h.Lock()
	defer h.Unlock()

	h.db[fish.ID] = fish

}

func (h *fishesHandler) fishes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		{
			h.getAllFishes(w, r)
			return
		}
	case "POST":
		{
			h.addNewFish(w, r)
			return
		}
	default:
		{
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte("method not allowed"))
			return
		}
	}
}

type adminPortal struct {
	password string
}

func newAdminPortal() *adminPortal {
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		panic("required env variable ADMIN_PASSWORD not set")
	}

	return &adminPortal{password: password}
}

func (a *adminPortal) handler(w http.ResponseWriter, r *http.Request) {
	user, pass, ok := r.BasicAuth()
	if !ok || user != "admin" || pass != a.password {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("You do not have the right permission"))
		return
	}

	w.Write([]byte("<html><h1>Super secret admin portal </h1></html>"))
}

func main() {
	admin := newAdminPortal()

	fishesHandler := newFishesHander()

	http.HandleFunc("/admin", admin.handler)

	http.HandleFunc("/fishes", fishesHandler.fishes)
	http.HandleFunc("/fishes/", fishesHandler.getFish)

	err := http.ListenAndServe(":8080", nil)

	if err != nil {
		panic(err)
	}
}
