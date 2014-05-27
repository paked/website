package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"code.google.com/p/go.crypto/bcrypt"
	"github.com/gorilla/mux"
	"github.com/hackedu/backend/database"
	"github.com/hackedu/backend/model"
)

// Authenticate checks the provided user information against the information
// in the database. If it all checks out, then a JWT is generated and
// returned.
func Authenticate(w http.ResponseWriter, r *http.Request,
	u *model.User) *AppError {
	defer r.Body.Close()

	var requestUser model.RequestUser
	err := json.NewDecoder(r.Body).Decode(&requestUser)
	if err != nil {
		return &AppError{err, "bad request", http.StatusBadRequest}
	}

	userFromDB, err := database.GetUserByEmail(requestUser.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			return &AppError{err, "user not found", http.StatusNotFound}
		}

		return &AppError{err, "error retrieving user",
			http.StatusInternalServerError}
	}

	err = userFromDB.ComparePassword(requestUser.Password)
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return &AppError{err, "invalid password", http.StatusBadRequest}
	} else if err != nil {
		return &AppError{err, "error checking password",
			http.StatusInternalServerError}
	}

	token, err := model.NewToken(userFromDB)
	if err != nil {
		return &AppError{err, "problem creating jwt token",
			http.StatusInternalServerError}
	}

	return renderJSON(w, token, http.StatusOK)
}

// CreateUser creates a new user from JSON in the request body.
func CreateUser(w http.ResponseWriter, r *http.Request,
	u *model.User) *AppError {
	if u == nil || u.Type != model.UserAdmin {
		err := errors.New("not authorized")
		return &AppError{err, err.Error(), http.StatusUnauthorized}
	}

	defer r.Body.Close()
	user, err := model.NewUser(r.Body)
	if err != nil {
		return &AppError{err, err.Error(), http.StatusBadRequest}
	}

	err = database.SaveUser(user)
	if err != nil {
		if err == model.ErrInvalidEmail {
			return &AppError{err, err.Error(), http.StatusBadRequest}
		}
		return &AppError{err, "error saving user", http.StatusInternalServerError}
	}

	return renderJSON(w, user, http.StatusOK)
}

// GetUser gets the user specified by ID in the url.
func GetUser(w http.ResponseWriter, r *http.Request, u *model.User) *AppError {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		return &AppError{err, "invalid id", http.StatusBadRequest}
	}

	if id == u.ID {
		return renderJSON(w, u, http.StatusOK)
	}

	return &AppError{err, "unauthorized", http.StatusBadRequest}
}

// GetCurrentUser gets the current authenticated user.
func GetCurrentUser(w http.ResponseWriter, r *http.Request,
	u *model.User) *AppError {
	if u == nil {
		return &AppError{errors.New("user not authorized"), "not authorized",
			http.StatusUnauthorized}
	}

	return renderJSON(w, u, http.StatusOK)
}
