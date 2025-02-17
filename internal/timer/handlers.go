package timer

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/google/uuid"
)

func HandleCreateHub(hs *HubService, w http.ResponseWriter, r *http.Request) {
	alias := struct {
		Alias string `json:"alias,omitempty"`
	}{}
	defer r.Body.Close()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	dat, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Couldn't read request body", 400)
		return
	}

	err = json.Unmarshal(dat, &alias)
	if err != nil {
		http.Error(w, "Couldn't parse request body", 400)
		return
	}

	id := hs.create()
	idBody := map[string]string{"id": id.String()}
	res, err := json.Marshal(idBody)
	if err != nil {
		http.Error(w, "Something went wrong", 500)
	}

	w.Header().Set("Content-Type", "appliation/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(res)
}

func HandleCheckHub(hs *HubService, w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    id, err := uuid.Parse(r.PathValue("hubId"))
    if err != nil {
        http.Error(w, "Bad request", http.StatusBadRequest)
        return
    }

    _, err = hs.get(id)
    if err != nil {
        http.Error(w, "Not found", http.StatusNotFound)
        return
    }

    w.WriteHeader(http.StatusOK)
}
