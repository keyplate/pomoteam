package main

import (
	"encoding/json"
	"io"
	"net/http"
)

func RespondWithError(w http.ResponseWriter, code int, msg string) error {
	errMsg := map[string]string{"error": msg}
	res, err := json.Marshal(errMsg)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "appliation/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(code)
	w.Write(res)
	return nil

}

func HandleCreateTimer(hs *HubService, w http.ResponseWriter, r *http.Request) {
	alias := struct {
		Alias string `json:"alias,omitempty"`
	}{}
	defer r.Body.Close()

	dat, err := io.ReadAll(r.Body)
	if err != nil {
		RespondWithError(w, 400, "Couldn't read request body")
		return
	}

	err = json.Unmarshal(dat, &alias)
	if err != nil {
		RespondWithError(w, 400, "Couldn't parse request body")
		return
	}

	id := hs.Create()
	idBody := map[string]string{"id": id.String()}
	res, err := json.Marshal(idBody)
	if err != nil {
		RespondWithError(w, 500, "Something went wrong")
	}

	w.Header().Set("Content-Type", "appliation/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusCreated)
	w.Write(res)
}
