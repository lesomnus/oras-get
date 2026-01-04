package og

import (
	"bufio"
	"fmt"
	"net/http"

	"github.com/lesomnus/oras-get/og/upstream"
)

func serveTagList(w http.ResponseWriter, r *http.Request, repo upstream.Repository) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if repo.Upstream.Redirect {
		url := fmt.Sprintf("%s://%s/v2/%s/tags/list", repo.Upstream.Scheme, repo.Reference.Domain(), repo.Reference.Repo())
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
		return
	}

	bw := bufio.NewWriter(w)
	defer bw.Flush()

	if err := repo.Tags(r.Context(), "", func(tags []string) error {
		for _, tag := range tags {
			if _, err := fmt.Fprintf(bw, "%s\n", tag); err != nil {
				return err
			}
		}
		return bw.Flush()
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
