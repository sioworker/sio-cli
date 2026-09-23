package sio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (h *Host) do(req *http.Request) ([]byte, error) {
	if h.Token != "" {
		req.Header.Set("Authorization", "Token "+h.Token)
	}
	req.Header.Set("Accept", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode/100 != 2 {
		var e struct{ Detail string }
		if json.Unmarshal(b, &e) == nil && e.Detail != "" {
			return nil, fmt.Errorf("%d: %s", res.StatusCode, e.Detail)
		}
		return nil, fmt.Errorf("%d: %s", res.StatusCode, strings.TrimSpace(string(b)))
	}
	return b, nil
}

func (h *Host) get(p string, v any) error {
	req, _ := http.NewRequest("GET", h.URL+p, nil)
	b, err := h.do(req)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

type Prob struct {
	Short string `json:"short_name"`
	Name  string `json:"full_name"`
	Left  *int   `json:"submissions_left"`
	Res   *struct {
		Score  any    `json:"score"`
		Status string `json:"status"`
	} `json:"user_result"`
}

type Sub struct {
	ID     int       `json:"id"`
	Date   time.Time `json:"date"`
	Score  *int      `json:"score"`
	Status string    `json:"status"`
	Prob   string    `json:"-"`
}

func (h *Host) Probs(ct string) ([]Prob, error) {
	var ps []Prob
	return ps, h.get("/api/c/"+ct+"/problem_list/", &ps)
}

func (h *Host) Subs(ct, prob string) ([]Sub, bool, error) { // last 20 only, bool=truncated
	var r struct {
		Subs  []Sub `json:"submissions"`
		Trunc bool  `json:"is_truncated_to_20"`
	}
	err := h.get("/api/c/"+ct+"/problem_submission_list/"+prob+"/", &r)
	for i := range r.Subs {
		r.Subs[i].Prob = prob
	}
	return r.Subs, r.Trunc, err
}

func (h *Host) Ping() (string, error) {
	req, _ := http.NewRequest("GET", h.URL+"/api/auth_ping", nil)
	b, err := h.do(req)
	return strings.TrimPrefix(strings.Trim(string(b), "\"\n"), "pong "), err
}

func (h *Host) Submit(contest, prob, file string) (string, error) {
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, _ := w.CreateFormFile("file", filepath.Base(file))
	io.Copy(fw, f)
	w.Close()
	req, _ := http.NewRequest("POST", h.URL+"/api/c/"+contest+"/submit/"+prob, &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	b, err := h.do(req)
	return strings.TrimSpace(string(b)), err
}
