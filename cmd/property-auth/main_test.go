package main

import "testing"

func TestStoreLoginDecision(t *testing.T) {
	s := &Store{users: map[string]string{}}
	if err := s.signup("tenant@example.com", "pw", "A Tenant"); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, email, password string
		want                  bool
	}{{"match", "tenant@example.com", "pw", true}, {"wrong password", "tenant@example.com", "bad", false}, {"unknown", "other@example.com", "pw", false}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := s.login(tc.email, tc.password); got != tc.want {
				t.Fatalf("login=%v want %v", got, tc.want)
			}
		})
	}
}
