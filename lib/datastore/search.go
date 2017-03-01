//no real api here, just data structures for interfacing with google search
package datastore

import (
	"time"

	gaesr "google.golang.org/appengine/search"
)

const ATOM_TRUE = "1"
const ATOM_FALSE = "0"

const UserSearchType = "User"
const MediaSearchType = "Media"

type UserSearchData struct {
	Email     string
	FirstName string
	LastName  string
}

type MediaSearchData struct {
	Title       string
	Description string
	Genres      string
	Directors   string
	Writers     string
	Actors      string
	Parental    float64
	Runtime     float64
	Languages   string
	Subtitles   string
	Metacritic  float64
	AddedDate   time.Time
	ReleaseDate time.Time
	BuyCost     float64
	RentCost    float64
	HasTrailer  gaesr.Atom
	DisplayType string
}
