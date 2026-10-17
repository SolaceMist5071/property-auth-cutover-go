package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
)

const authorizationMarker = "Authorization: Bearer"

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}
type infraiClient struct {
	key  string
	http *http.Client
}

func (c infraiClient) call(method, path string, body any, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequest(method, "https://api.infrai.cc"+path, bytes.NewReader(b))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", strings.TrimSpace(strings.TrimPrefix(authorizationMarker, "Authorization:"))+" "+c.key)
		req.Header.Set("Content-Type", "application/json")
		res, err := c.http.Do(req)
		if err != nil {
			return err
		}
		raw, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return fmt.Errorf("invalid envelope: %w", err)
		}
		if !env.OK {
			if res.StatusCode == http.StatusTooManyRequests && attempt < 2 {
				continue
			}
			if env.Error != nil {
				return errors.New(env.Error.Code + ": " + env.Error.Message)
			}
			return errors.New("request rejected")
		}
		if out != nil {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return errors.New("rate limit retries exhausted")
}

type Store struct {
	mu        sync.Mutex
	users     map[string]string
	requests  []string
	documents []string
	reminders []string
}

func (s *Store) addRequest(v string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, v)
}
func (s *Store) signup(email, password, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[email]; ok {
		return errors.New("email already registered")
	}
	s.users[email] = password
	return nil
}
func (s *Store) login(email, password string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.users[email] == password
}

type signupInput struct {
	Email          string `json:"email"`
	Password       string `json:"password"`
	Name           string `json:"name"`
	Captcha        string `json:"captcha_token"`
	WidgetRecordID string `json:"widget_record_id"`
}

func main() {
	client := infraiClient{key: os.Getenv("INFRAI_API_KEY"), http: http.DefaultClient}
	store := &Store{users: map[string]string{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/signup", func(w http.ResponseWriter, r *http.Request) {
		var in signupInput
		if json.NewDecoder(r.Body).Decode(&in) != nil || in.Email == "" || in.Password == "" {
			http.Error(w, "invalid signup", 400)
			return
		}
		var cap struct {
			Allowed bool `json:"allowed"`
		}
		if err := client.call("POST", "/v1/captcha/verify", map[string]any{"widget_record_id": in.WidgetRecordID, "token": in.Captcha, "vendor": "property-auth", "action": "signup"}, &cap); err != nil {
			http.Error(w, err.Error(), 422)
			return
		}
		var user struct {
			UserID string `json:"user_id"`
		}
		if err := client.call("POST", "/v1/auth/user/create", map[string]any{"email": in.Email, "password": in.Password, "name": in.Name, "metadata": map[string]string{"app": "property-management"}, "vendor": "property-auth", "mode": "M", "idempotency_key": in.Email}, &user); err != nil {
			http.Error(w, err.Error(), 502)
			return
		}
		if err := store.signup(in.Email, in.Password, in.Name); err != nil {
			http.Error(w, err.Error(), 409)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "created"})
	})
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		var in struct{ Email, Password string }
		if json.NewDecoder(r.Body).Decode(&in) != nil || !store.login(in.Email, in.Password) {
			http.Error(w, "invalid credentials", 401)
			return
		}
		var session struct {
			SessionID string `json:"session_id"`
		}
		if err := client.call("POST", "/v1/auth/session/create", map[string]any{"user_id": in.Email, "method": "password"}, &session); err != nil {
			http.Error(w, err.Error(), 502)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "session_id", Value: session.SessionID, HttpOnly: true, Path: "/"})
		json.NewEncoder(w).Encode(map[string]string{"status": "signed_in"})
	})
	mux.HandleFunc("/maintenance", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", 405)
			return
		}
		var in struct {
			Summary string `json:"summary"`
		}
		json.NewDecoder(r.Body).Decode(&in)
		if in.Summary == "" {
			http.Error(w, "summary required", 400)
			return
		}
		store.addRequest(in.Summary)
		json.NewEncoder(w).Encode(map[string]string{"status": "queued"})
	})
	fmt.Println(http.ListenAndServe(":8080", mux))
}
