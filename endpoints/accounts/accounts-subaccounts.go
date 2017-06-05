package accounts

import (
	"strings"

	"github.com/dakom/basic-site-api/lib/pages"
)

type SubAccountInfo struct {
	PublicAccountInfo
	Username string `json:"uname"`
}

func SubaccountsList(rData *pages.RequestData) {
	userIds := rData.UserRecord.GetData().SubAccountIds

	if len(userIds) > 0 {
		userRecords, err := GetUserRecordsMap(rData, userIds)
		if err != nil {
			rData.SetJsonErrorCodeResponse(err.Error())
			return
		}

		userList := make([]SubAccountInfo, 0, len(userRecords))
		for _, record := range userRecords {
			uInfo := SubAccountInfo{
				PublicAccountInfo: *GetUserInfo(record),
				Username:          record.GetData().UsernameLookups[0],
			}
			userList = append(userList, uInfo)
		}

		rData.SetJsonSuccessResponse(pages.JsonMapGeneric{
			"list": userList,
		})
	} else {
		rData.SetJsonSuccessResponse(pages.JsonMapGeneric{
			"list": []SubAccountInfo{},
		})
	}

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
