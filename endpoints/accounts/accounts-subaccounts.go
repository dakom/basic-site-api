package accounts

import (
	"strings"

	"github.com/dakom/basic-site-api/lib/pages"
	"github.com/dakom/basic-site-api/setup/config/static/statuscodes"
)

type SubAccountInfo struct {
	*PublicAccountInfo
	Username string `json:"uname"`
}

func SubaccountsList(rData *pages.RequestData) {
	userIds := rData.UserRecord.GetData().SubAccountIds

	if len(userIds) > 0 {
		publicUserList, err := GetUserInfosList(rData, userIds)
		if err != nil {
			rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
			return
		}

		usernameMap, err := GetUsernamesForIds(rData, userIds)
		if err != nil {
			rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
			return
		}

		rData.LogInfo("%v", usernameMap)

		userList := make([]*SubAccountInfo, len(userIds))
		for idx, publicUserInfo := range publicUserList {
			subaccountInfo := &SubAccountInfo{
				PublicAccountInfo: publicUserInfo,
			}

			usernames := usernameMap[userIds[idx]]
			if len(usernames) != 1 {
				rData.LogError("Length of subaccount usernames should be 1 but instead it is %d! User Info: %v", len(usernames), publicUserInfo)
			}

			if len(usernames) > 0 {
				subaccountInfo.Username = usernames[0]
			}

			userList[idx] = subaccountInfo

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
