package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"

	"github.com/shayanderson/jem"
)

/*
Simple REST API example using JEM for entity management and validation:
- Get users `curl localhost:8080/users`
- Create users: `curl -XPOST -d'[{"name":"Bob","age":20}]' localhost:8080/users`
- Update single user: `curl -XPATCH -d'{"id":"[ID]","age":22}' localhost:8080/users`
*/

// User represents a user
type User struct {
	ID   string `json:"id"   validate:"id,required,len=5"`
	Name string `json:"name" validate:"required"`
	Age  int    `json:"age"  validate:"required,min=0,max=130"`
}

// main runs the HTTP server
func main() {
	userStore := newUserStore()
	// seed user store
	if _, err := userStore.Create(User{Name: "John Doe", Age: 30}); err != nil {
		panic("failed to create user: " + err.Error())
	}
	if _, err := userStore.Create(User{Name: "Jane Smith", Age: 25}); err != nil {
		panic("failed to create user: " + err.Error())
	}
	userHandler := newUserHandler(userStore)

	http.HandleFunc("GET /users", userHandler.get)
	http.HandleFunc("POST /users", userHandler.post)
	http.HandleFunc("PATCH /users", userHandler.patch)

	fmt.Println("starting server on :8080")
	err := http.ListenAndServe(":8080", nil) //#nosec G114
	if err != nil {
		fmt.Println("failed to start server on :8080:", err)
		os.Exit(1)
	}
}

// UserStore manages user data
type UserStore struct {
	users map[string]User
}

// newUserStore creates a new user store
func newUserStore() *UserStore {
	return &UserStore{
		users: make(map[string]User),
	}
}

// All returns all users in the store
func (s *UserStore) All() []User {
	var users []User
	for _, user := range s.users {
		users = append(users, user)
	}
	return users
}

// Create adds a new user to the store, not thread-safe
func (s *UserStore) Create(user User) (string, error) {
	user.ID = "u-" + strconv.Itoa(rand.Intn(999-100)+100) //#nosec G404
	if _, exists := s.users[user.ID]; exists {
		return "", fmt.Errorf("user with id %s already exists", user.ID)
	}
	s.users[user.ID] = user
	return user.ID, nil
}

// Update modifies an existing user in the store, not thread-safe
func (s *UserStore) Update(user User, m map[string]any) (*User, error) {
	u, ok := s.users[user.ID]
	if !ok {
		return nil, fmt.Errorf("user with id %s does not exist", user.ID)
	}
	if _, ok = m["name"]; ok {
		u.Name = user.Name
	}
	if _, ok = m["age"]; ok {
		u.Age = user.Age
	}
	s.users[user.ID] = u
	return &u, nil
}

// UserHandler handles user-related HTTP requests
type UserHandler struct {
	factory *jem.Factory[User, string]
	store   *UserStore
}

// newUserHandler creates a new user handler
func newUserHandler(store *UserStore) *UserHandler {
	return &UserHandler{factory: jem.New[User, string](), store: store}
}

// get handles GET requests for users
func (u *UserHandler) get(w http.ResponseWriter, r *http.Request) {
	users := u.store.All()
	writeJSON(w, http.StatusOK, users)
}

// patch handles PATCH requests for users
func (u *UserHandler) patch(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	res, err := u.factory.MakePartial(body)
	if err != nil {
		writeJSON(
			w,
			http.StatusUnprocessableEntity,
			map[string]string{"error": "validation failed: " + err.Error()},
		)
		return
	}
	v, err := u.store.Update(*res.Value, res.Map)
	if err != nil {
		writeJSON(
			w,
			http.StatusConflict,
			map[string]string{"error": "failed to update user: " + err.Error()},
		)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

// post handles POST requests for users
func (u *UserHandler) post(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	res, err := u.factory.ReadMany(r.Body)
	if err != nil {
		if errors.Is(err, jem.ErrRead) {
			http.Error(w, "failed to read request body: "+err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(
			w,
			http.StatusUnprocessableEntity,
			map[string]string{"error": "validation failed: " + err.Error()},
		)
		return
	}

	ids := []string{}
	for _, item := range res {
		id, err := u.store.Create(*item.Value)
		if err != nil {
			writeJSON(
				w,
				http.StatusConflict,
				map[string]string{"error": "failed to create user: " + err.Error()},
			)
			return
		}
		ids = append(ids, id)
	}
	writeJSON(w, http.StatusCreated, ids)
}

// writeJSON writes a JSON response with the given status code and payload
func writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
