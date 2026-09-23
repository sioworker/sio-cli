package sio

import (
	"encoding/json"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const quotesURL = "https://github.com/mudroljub/programming-quotes-api/raw/refs/heads/master/data/quotes.json"

type Quote struct {
	Text   string `json:"text"`
	Author string `json:"author"`
}

func (c *Cfg) QuotesOn() bool { return c.Quotes == nil || *c.Quotes } // def on

func RandQuote() (Quote, bool) { // cached a week in the user cache dir
	d, _ := os.UserCacheDir()
	p := filepath.Join(d, "sio", "quotes.json")
	if st, err := os.Stat(p); err != nil || time.Since(st.ModTime()) > 7*24*time.Hour {
		b, err := fetchQuotes()
		os.MkdirAll(filepath.Dir(p), 0700)
		if err == nil {
			os.WriteFile(p, b, 0600)
		} else if st == nil {
			os.WriteFile(p, []byte("[]"), 0600) // offline w/ no cache, retry in a day not every run
			os.Chtimes(p, time.Now(), time.Now().Add(-6*24*time.Hour))
		} else {
			os.Chtimes(p, time.Now(), time.Now()) // keep the stale one a week more
		}
	}
	b, err := os.ReadFile(p)
	var qs []Quote
	if err != nil || json.Unmarshal(b, &qs) != nil || len(qs) == 0 {
		return Quote{}, false
	}
	return qs[rand.IntN(len(qs))], true
}

func fetchQuotes() ([]byte, error) {
	res, err := (&http.Client{Timeout: 3 * time.Second}).Get(quotesURL) // short, runs after every cmd
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, ErrHTML
	}
	b, err := io.ReadAll(res.Body)
	var qs []Quote
	if err == nil && json.Unmarshal(b, &qs) != nil {
		return nil, ErrHTML
	}
	return b, err
}
