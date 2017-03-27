package accounts

import (
	"strconv"
	"strings"

	"github.com/dakom/basic-site-api/lib/datastore"
	"github.com/dakom/basic-site-api/lib/pages"
	"github.com/dakom/basic-site-api/setup/config/static/statuscodes"

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

type PublicAccountInfo struct {
	Id        string `json:"uid"`
	Username  string `json:"uname"`
	FirstName string `json:"fname"`
	LastName  string `json:"lname"`
	AvatarId  string `json:"avid"`
}

func appUrlParamsFromRequest(rData *pages.RequestData) string {
	var appUrl string
	appId := strings.TrimSpace(rData.HttpRequest.FormValue("appId"))
	appPort := strings.TrimSpace(rData.HttpRequest.FormValue("appPort"))

	if appId != "" {
		appUrl += "/" + appId
	}
	if appPort != "" {
		appUrl += "/" + appPort
	}

	return appUrl
}
func GetUserInfosList(rData *pages.RequestData, ids []int64) ([]*PublicAccountInfo, error) {
	userInfos, err := GetUserInfosMap(rData, ids)
	if err != nil {
		return nil, err
	}

	userList := make([]*PublicAccountInfo, len(userInfos))
	idx := 0
	for _, userInfo := range userInfos {
		userList[idx] = userInfo
		idx++
	}

	return userList, nil
}

func GetUserInfosMap(rData *pages.RequestData, ids []int64) (map[int64]*PublicAccountInfo, error) {
	userRecords, err := GetUserRecordsMap(rData, ids)
	if err != nil {
		return nil, err
	}

	userMap := make(map[int64]*PublicAccountInfo)
	for key, record := range userRecords {
		userMap[key] = &PublicAccountInfo{
			Id:        record.GetKeyIntAsString(),
			Username:  GetPrimaryUsername(record.GetData()),
			FirstName: record.GetData().FirstName,
			LastName:  record.GetData().LastName,
			AvatarId:  strconv.FormatInt(record.GetData().AvatarId, 10),
		}
	}

	return userMap, nil
}

func GetUserRecordsMap(rData *pages.RequestData, ids []int64) (map[int64]*datastore.UserRecord, error) {

	userKeys := datastore.GetMultiKeysFromInts(rData.Ctx, datastore.USER_TYPE, ids, nil)
	userDatas := make([]*datastore.UserData, len(userKeys))

	if multiError := gaeds.GetMulti(rData.Ctx, userKeys, userDatas); multiError != nil {
		//theoretically we could just cull the bad ones... but missing users is really not ok
		rData.LogError("%v", multiError)
		rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
		return nil, multiError
	}

	userMap := make(map[int64]*datastore.UserRecord)

	for idx, userData := range userDatas {
		userKey := userKeys[idx]
		userId := ids[idx]

		if userMap[userId] == nil {
			userRecord := &datastore.UserRecord{}
			datastore.SetKey(rData.Ctx, userRecord, userKey)
			userRecord.SetData(userData)
			userMap[userId] = userRecord
		}
	}

	return userMap, nil
}

func GetFullNameShortened(userData *datastore.UserData) string {
	name := userData.FirstName

	if userData.LastName == "" {
		return name
	}
	if name != "" {
		name += " "
	}

	name += userData.LastName[:1] + "."

	return name
}

func GetPrimaryUsername(userData *datastore.UserData) string {
	namesLength := len(userData.UsernameHistory)
	if namesLength == 0 {
		return ""
	}

	return userData.UsernameHistory[namesLength-1]
}

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
