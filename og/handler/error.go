package handler

import "net/http"

func ManifestParseFailed(w http.ResponseWriter, err error) {
	http.Error(w, "parse manifest: "+err.Error(), http.StatusBadGateway)
}
