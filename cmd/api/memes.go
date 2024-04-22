package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/michaelgov-ctrl/memebase/internal/data"
	"github.com/michaelgov-ctrl/memebase/internal/validator"
)

func (app *application) createMemeHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Artist string `json:"artist"`
		Title  string `json:"title"`
		Meme   string `json:"meme"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	meme := &data.Meme{
		Artist: input.Artist,
		Title:  input.Title,
		Meme:   input.Meme,
	}

	v := validator.New()
	if data.ValidateMeme(v, meme); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	if err := app.models.Memes.Insert(meme); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/memes/%s", meme.ID))

	if err := app.writeJSON(w, http.StatusCreated, envelope{"meme": meme}, headers); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) showMemeHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	meme, err := app.models.Memes.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDocNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if err := app.writeJSON(w, http.StatusOK, envelope{"meme": meme}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) updateMemeHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	meme, err := app.models.Memes.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDocNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if r.Header.Get("X-Expected-Version") != "" {
		if strconv.Itoa(int(meme.Version)) != r.Header.Get("X-Expected-Version") {
			app.editConflictResponse(w, r)
			return
		}
	}

	var input struct {
		Artist *string `json:"artist"`
		Title  *string `json:"title"`
		Meme   *string `json:"meme"`
	}

	if err := app.readJSON(w, r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if input.Artist != nil {
		meme.Artist = *input.Artist
	}

	if input.Title != nil {
		meme.Title = *input.Title
	}

	if input.Meme != nil {
		meme.Meme = *input.Meme
	}

	v := validator.New()
	if data.ValidateMeme(v, meme); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	if err := app.models.Memes.Update(meme); err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if err := app.writeJSON(w, http.StatusOK, envelope{"meme": meme}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) deleteMemeHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	if err := app.models.Memes.Delete(id); err != nil {
		switch {
		case errors.Is(err, data.ErrDocNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if err := app.writeJSON(w, http.StatusOK, envelope{"message": "movie successfully deleted"}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) listMemesHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Artist string
		Title  string
		data.Filters
	}

	v := validator.New()
	qs := r.URL.Query()

	input.Artist = app.readString(qs, "artist", "")
	input.Title = app.readString(qs, "title", "")

	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 5, v)

	input.Filters.Sort = app.readString(qs, "sort", "created")
	input.Filters.SortSafelist = []string{"artist", "title", "created", "-artist", "-title", "-created"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	memes, metadata, err := app.models.Memes.GetAll(input.Artist, input.Title, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if err := app.writeJSON(w, http.StatusOK, envelope{"metadata": metadata, "memes": memes}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) showRandMemeHandler(w http.ResponseWriter, r *http.Request) {
	meme, err := app.models.Memes.GetRandom()
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDocNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	if err := app.writeJSON(w, http.StatusOK, envelope{"meme": meme}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
