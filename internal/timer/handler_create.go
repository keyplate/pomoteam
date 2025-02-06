package timer

import (
	"encoding/json"
	"io"
	"net/http"
)

func HandleCreateTimer(hs *HubService, w http.ResponseWriter, r *http.Request) {
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
