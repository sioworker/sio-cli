package sio

import (
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var (
	ErrLogin = errors.New("bad login")  // wrong user/pass
	Err2FA   = errors.New("needs 2fa")  // two_factor wants a token step
	ErrSess  = errors.New("no session") // not logged in or expired
	csrfRe   = regexp.MustCompile(`name="csrfmiddlewaretoken" value="([^"]+)"`)
	okRe     = regexp.MustCompile(`^((?:INI_)?OK)\d+$`)
	tfaRe    = regexp.MustCompile(`current_step"[^>]*value="token"|value="token"[^>]*current_step`)
	nofollow = &http.Client{Timeout: cl.Timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
)

func cookie(res *http.Response, n string) string {
	for _, c := range res.Cookies() {
		if c.Name == n {
			return c.Value
		}
	}
	return ""
}

func (h *Host) Login(user, pass string) error { // web login like the site, keeps only the sessionid
	res, err := nofollow.Get(h.URL + "/login/")
	if err != nil {
		return err
	}
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	m, ct := csrfRe.FindSubmatch(b), cookie(res, "csrftoken")
	if m == nil || ct == "" {
		return fmt.Errorf("%d: no login form on %s/login/", res.StatusCode, h.URL)
	}
	f := url.Values{"csrfmiddlewaretoken": {string(m[1])}, "login_view-current_step": {"auth"}, "auth-username": {user}, "auth-password": {pass}}
	req, _ := http.NewRequest("POST", h.URL+"/login/", strings.NewReader(f.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", h.URL+"/login/") // django csrf wants it on https
	req.AddCookie(&http.Cookie{Name: "csrftoken", Value: ct})
	res, err = nofollow.Do(req)
	if err != nil {
		return err
	}
	b, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if s := cookie(res, "sessionid"); s != "" && res.StatusCode/100 == 3 {
		h.Session = s
		return nil
	}
	if tfaRe.Match(b) {
		return Err2FA
	}
	return ErrLogin
}

func (h *Host) SubWeb(ct, id string) (Sub, error) { // read one row off /c/ct/submissions/, Status "?" if not there yet
	if h.Session == "" {
		return Sub{}, ErrSess
	}
	req, _ := http.NewRequest("GET", h.URL+"/c/"+ct+"/submissions/", nil)
	req.AddCookie(&http.Cookie{Name: "sessionid", Value: h.Session})
	res, err := nofollow.Do(req)
	if err != nil {
		return Sub{}, err
	}
	defer res.Body.Close()
	if res.StatusCode/100 == 3 { // bounced to /login/, session expired
		return Sub{}, ErrSess
	}
	b, _ := io.ReadAll(res.Body)
	s := Sub{Status: "?"}
	st := regexp.MustCompile(`id="submission` + regexp.QuoteMeta(id) + `-status"\s*class="([^"]*)"`).FindSubmatch(b)
	if st == nil {
		return s, nil
	}
	s.Status = ""                                                                   // row there but no submission--X = hidden
	if m := regexp.MustCompile(`submission--(\S+)`).FindSubmatch(st[1]); m != nil { // display_type, ? while judging
		s.Status = okRe.ReplaceAllString(html.UnescapeString(string(m[1])), "$1") // OK100 -> OK, the num is a score bucket for css
	}
	if m := regexp.MustCompile(`id="submission` + regexp.QuoteMeta(id) + `-score"[^>]*>\s*([^<]*?)\s*<`).FindSubmatch(b); m != nil && len(m[1]) > 0 {
		var n int
		if _, err := fmt.Sscan(string(m[1]), &n); err == nil {
			s.Score = &n
		}
	}
	return s, nil
}
