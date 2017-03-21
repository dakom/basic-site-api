package accounts

import (
	"strconv"
	"strings"

	"github.com/dakom/basic-site-api/lib/datastore"
	"github.com/dakom/basic-site-api/lib/pages"
	"github.com/dakom/basic-site-api/setup/config/static/statuscodes"
	gaeds "google.golang.org/appengine/datastore"
)

type SubaccountInfo struct {
	Id        string `json:"uid"`
	Username  string `json:"uname"`
	FirstName string `json:"fname"`
	LastName  string `json:"lname"`
	AvatarId  string `json:"avid"`
}

func SubaccountsList(rData *pages.RequestData) {
	userIdInts := rData.UserRecord.GetData().SubAccountIds
	userIdKeys := datastore.GetMultiKeysFromInts(rData.Ctx, datastore.USER_TYPE, userIdInts, nil)

	infoList := make([]SubaccountInfo, len(userIdKeys))

	rData.LogInfo("Keys: %v", userIdInts)

	if len(userIdKeys) > 0 {

		userDatas := make([]datastore.UserData, len(userIdKeys))

		if multiError := gaeds.GetMulti(rData.Ctx, userIdKeys, userDatas); multiError != nil {
			rData.LogError("%v", multiError)
			rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
			return
		}

		for idx, userData := range userDatas {

			infoList[idx] = SubaccountInfo{
				Id:        strconv.FormatInt(userIdInts[idx], 10),
				Username:  GetPrimaryUsername(&userData),
				FirstName: userData.FirstName,
				LastName:  userData.LastName,
				AvatarId:  strconv.FormatInt(userData.AvatarId, 10),
			}
		}
	}

	rData.SetJsonSuccessResponse(pages.JsonMapGeneric{
		"list": infoList,
	})
}

func CreateSubaccountRequest(rData *pages.RequestData) {

	var terms bool

	if rData.HttpRequest.FormValue("terms") == "true" {
		terms = true
	}

	info := &RegisterInfo{
		Terms:      terms,
		Username:   strings.ToLower(strings.TrimSpace(rData.HttpRequest.FormValue("uname"))),
		FirstName:  strings.TrimSpace(rData.HttpRequest.FormValue("fname")),
		LastName:   strings.TrimSpace(rData.HttpRequest.FormValue("lname")),
		Password:   strings.TrimSpace(rData.HttpRequest.FormValue("pw")),
		LookupType: LOOKUP_TYPE_USERNAME,
		ParentId:   rData.UserRecord.GetKey().IntID(),
	}
	if err := DoRegister(rData, info); err != nil {
		rData.SetJsonErrorCodeResponse(err.Error())
		return
	}
	rData.SetJsonSuccessResponse(nil)
}
