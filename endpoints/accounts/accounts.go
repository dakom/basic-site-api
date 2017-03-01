package accounts

import (
	"github.com/dakom/basic-site-api/lib/datastore"

	"golang.org/x/net/context"
	gaeds "google.golang.org/appengine/datastore"
	gaesr "google.golang.org/appengine/search"
)

const PASSWORD_MIN_LENGTH int = 6
const PASSWORD_MAX_LENGTH int = 32

const (
	_ = 1 << iota
	LOOKUP_TYPE_USERNAME
	LOOKUP_TYPE_OAUTH
)

func GetUsernamesFromKey(c context.Context, userKey *gaeds.Key) ([]string, error) {
	var lookupDatas []datastore.UsernameLookupData
	var usernames []string
	query := gaeds.NewQuery(datastore.USER_NAME_LOOKUP_TYPE).Filter("UserId =", userKey.IntID())

	keys, err := query.GetAll(c, &lookupDatas)
	if err != nil {
		return nil, err
	}

	usernames = make([]string, len(keys))
	for idx, key := range keys {
		usernames[idx] = key.StringID()
	}

	return usernames, nil
}

//Differentiates between a "record not found", which will return a null record, and a proper technical error which returns error
func GetUserRecordViaUsername(c context.Context, username string) (*datastore.UserRecord, error) {
	var userRecord datastore.UserRecord
	var lookupRecord datastore.UsernameLookupRecord
	var exists bool
	var err error

	err = datastore.LoadFromKey(c, &lookupRecord, username)

	if err == gaeds.ErrNoSuchEntity {
		err = nil
		goto finished
	}

	if err != nil {
		goto finished
	}

	err = datastore.LoadFromKey(c, &userRecord, lookupRecord.GetData().UserId)

	if err == gaeds.ErrNoSuchEntity {
		err = nil
		goto finished
	}

	if err != nil {
		goto finished
	}

	exists = true

finished:

	if !exists {
		return nil, err
	}

	return &userRecord, err
}

//Also differentiates between a "record not found", which will return a null record, and a proper technical error which returns error
func GetUserRecordViaKey(c context.Context, keyVal interface{}) (*datastore.UserRecord, error) {
	var userRecord datastore.UserRecord
	var exists bool

	err := datastore.LoadFromKey(c, &userRecord, keyVal)
	if err == gaeds.ErrNoSuchEntity {
		err = nil
		goto finished
	}

	if err != nil {
		goto finished
	}

	exists = true

finished:

	if !exists {
		return nil, err
	}

	return &userRecord, err
}

func RemoveFromSearch(c context.Context, userID string) error {
	index, err := gaesr.Open(datastore.UserSearchType)

	if err != nil {
		return err
	}

	err = index.Delete(c, userID)

	if err != nil {
		return err
	}

	return nil
}
