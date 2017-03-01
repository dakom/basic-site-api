package accounts

import (
	"strconv"
	"strings"

	"github.com/dakom/basic-site-api/setup/config/static/statuscodes"
	"github.com/dakom/basic-site-api/lib/datastore"
	"github.com/dakom/basic-site-api/lib/pages"
)

type SubaccountInfo struct {
	Id        string `json:"uid"`
	Username  string `json:"uname"`
	FirstName string `json:"fname"`
	LastName  string `json:"lname"`
	AvatarId  string `json:"avid"`
}

func SubaccountsList(rData *pages.RequestData) {
	userIdKeys := rData.UserRecord.GetData().SubAccountIds

	infoList := make([]SubaccountInfo, len(userIdKeys))

	if len(userIdKeys) > 0 {

		genericRecords, err := datastore.GetMultiDataSimpleInt(rData.Ctx, datastore.USER_TYPE, userIdKeys)
		if err != nil {
			rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
			return
		}

		for idx, genericRecord := range genericRecords {
			userRecord, ok := genericRecord.(*datastore.UserRecord)
			if !ok {
				rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
				return
			}
			infoList[idx] = SubaccountInfo{
				Id:        userRecord.GetKeyIntAsString(),
				Username:  userRecord.GetData().UsernameHistory[0],
				FirstName: userRecord.GetData().FirstName,
				LastName:  userRecord.GetData().LastName,
				AvatarId:  strconv.FormatInt(userRecord.GetData().AvatarId, 10),
			}
		}
	}

	rData.SetJsonSuccessResponse(map[string]interface{}{
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
