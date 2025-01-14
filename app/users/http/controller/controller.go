package controller

import (
	"encoding/json"
	cmiddleware "hexagony/app/shared/http/middleware"
	"hexagony/app/users/domain"
	"hexagony/lib/clog"
	"hexagony/lib/crypto"
	"hexagony/lib/rest"
	"hexagony/lib/validation"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type UserHandler struct {
	userUseCase domain.UserUseCase
}

func NewUserHandler(c *chi.Mux, as domain.UserUseCase) {
	handler := UserHandler{userUseCase: as}

	c.Route("/user", func(r chi.Router) {
		r.Use(cmiddleware.AuthMiddleware)

		r.Get("/", handler.FindAll)
		r.Get("/{uuid}", handler.FindByID)
		r.Post("/", handler.Add)
		r.Put("/{uuid}", handler.Update)
		r.Delete("/{uuid}", handler.Delete)
	})
}

type createUserRequest struct {
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required"`
	Password string `json:"password" validate:"required,gte=8"`
}

type updateUserRequest struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required"`
}

// FindAll godoc
// @Summary      List of users
// @Description  lists all users
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string  true  "Insert your access token"  default(Bearer <Add access token here>)
// @Success      200            {object}  []domain.User
// @Failure      500            {object}  rest.Message
// @Router       /user [get]
func (u *UserHandler) FindAll(w http.ResponseWriter, r *http.Request) {
	users, err := u.userUseCase.FindAll(r.Context())
	if err != nil {
		clog.Error(err, domain.ErrFindAll.Error())
		rest.DecodeError(w, r, domain.ErrFindAll, http.StatusInternalServerError)
		return
	}

	rest.JSON(w, http.StatusOK, &users)
}

// FindByID godoc
// @Summary      List an user
// @Description  lists an user by uuid
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string  true  "Insert your access token"  default(Bearer <Add access token here>)
// @Param        uuid           path      string  true  "user uuid"
// @Success      200            {object}  domain.User
// @Failure      422            {object}  rest.Message
// @Failure      500            {object}  rest.Message
// @Router       /user/{uuid} [get]
func (u *UserHandler) FindByID(w http.ResponseWriter, r *http.Request) {
	uuid, err := uuid.Parse(chi.URLParam(r, "uuid"))
	if err != nil {
		clog.Error(err, domain.ErrUUIDParse.Error())
		rest.DecodeError(w, r, domain.ErrUUIDParse, http.StatusInternalServerError)
		return
	}

	user, err := u.userUseCase.FindByID(r.Context(), uuid)
	if err != nil {
		clog.Error(err, domain.ErrFindByID.Error())
		rest.DecodeError(w, r, domain.ErrFindByID, http.StatusUnprocessableEntity)
		return
	}

	rest.JSON(w, http.StatusOK, user)
}

// Add godoc
// @Summary      Add an user
// @Description  add a new user
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string             true  "Insert your access token"  default(Bearer <Add access token here>)
// @Param        payload        body      createUserRequest  true  "add a new user"
// @Success      201            {object}  rest.Message
// @Failure      400            {object}  rest.Message
// @Failure      422            {object}  rest.Message
// @Failure      500            {object}  rest.Message
// @Router       /user [post]
func (u *UserHandler) Add(w http.ResponseWriter, r *http.Request) {
	var payload createUserRequest

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		clog.Error(err, domain.ErrAdd.Error())
		rest.DecodeError(w, r, domain.ErrAdd, http.StatusInternalServerError)
		return
	}

	validation := validation.New()

	if err := validation.BindStruct(r.Context(), payload); err != nil {
		validation.DecodeError(w, err)
		return
	}

	bcrypt := crypto.New()

	hashPass, err := bcrypt.HashPassword(payload.Password, 10)
	if err != nil {
		clog.Error(err, domain.ErrHashPassword.Error())
		rest.DecodeError(w, r, domain.ErrHashPassword, http.StatusUnprocessableEntity)
		return
	}

	user := domain.User{
		UUID:      uuid.New(),
		Name:      payload.Name,
		Email:     payload.Email,
		Password:  hashPass,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = u.userUseCase.Add(r.Context(), &user)
	if err != nil {
		clog.Error(err, domain.ErrAdd.Error())
		rest.DecodeError(w, r, domain.ErrAdd, http.StatusUnprocessableEntity)
		return
	}

	rest.JSON(w, http.StatusCreated, &rest.Message{Message: "Created"})
}

// Update godoc
// @Summary      Update an user
// @Description  update an user by uuid
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string             true  "Insert your access token"  default(Bearer <Add access token here>)
// @Param        uuid           path      string             true  "user uuid"
// @Param        payload        body      updateUserRequest  true  "update an user by uuid"
// @Success      200            {object}  rest.Message
// @Failure      400            {object}  rest.Message
// @Failure      422            {object}  rest.Message
// @Failure      500            {object}  rest.Message
// @Router       /user/{uuid} [put]
func (u *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	uuid, err := uuid.Parse(chi.URLParam(r, "uuid"))
	if err != nil {
		clog.Error(err, domain.ErrUUIDParse.Error())
		rest.DecodeError(w, r, domain.ErrUUIDParse, http.StatusInternalServerError)
		return
	}

	var payload updateUserRequest

	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		clog.Error(err, domain.ErrUpdate.Error())
		rest.DecodeError(w, r, domain.ErrUpdate, http.StatusUnprocessableEntity)
		return
	}

	validation := validation.New()

	if err := validation.BindStruct(r.Context(), payload); err != nil {
		clog.Error(err, domain.ErrUpdate.Error())
		validation.DecodeError(w, err)
		return
	}

	user := domain.User{
		Name:      payload.Name,
		Email:     payload.Email,
		UpdatedAt: time.Now(),
	}

	err = u.userUseCase.Update(r.Context(), uuid, &user)
	if err != nil {
		clog.Error(err, domain.ErrUpdate.Error())
		rest.DecodeError(w, r, domain.ErrUpdate, http.StatusUnprocessableEntity)
		return
	}

	rest.JSON(w, http.StatusOK, &rest.Message{Message: "Updated"})
}

// Update godoc
// @Summary      Delete an user
// @Description  delete an user by uuid
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string  true  "Insert your access token"  default(Bearer <Add access token here>)
// @Param        uuid           path      string  true  "user uuid"
// @Success      200            {object}  rest.Message
// @Failure      422            {object}  rest.Message
// @Failure      500            {object}  rest.Message
// @Router       /user/{uuid} [delete]
func (u *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uuid, err := uuid.Parse(chi.URLParam(r, "uuid"))
	if err != nil {
		clog.Error(err, domain.ErrDelete.Error())
		rest.DecodeError(w, r, domain.ErrDelete, http.StatusInternalServerError)
		return
	}

	err = u.userUseCase.Delete(r.Context(), uuid)
	if err != nil {
		clog.Error(err, domain.ErrDelete.Error())
		rest.DecodeError(w, r, domain.ErrDelete, http.StatusUnprocessableEntity)
		return
	}

	rest.JSON(w, http.StatusOK, &rest.Message{Message: "Deleted"})
}
