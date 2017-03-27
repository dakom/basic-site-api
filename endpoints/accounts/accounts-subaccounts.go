package accounts

import (
	"strings"

	"github.com/dakom/basic-site-api/lib/pages"
	"github.com/dakom/basic-site-api/setup/config/static/statuscodes"
)

func SubaccountsList(rData *pages.RequestData) {
	userIds := rData.UserRecord.GetData().SubAccountIds

	if len(userIds) > 0 {
		userList, err := GetUserInfosList(rData, userIds)
		if err != nil {
			rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
			return
		}

		rData.SetJsonSuccessResponse(pages.JsonMapGeneric{
			"list": userList,
		})
	} else {
		rData.SetJsonSuccessResponse(pages.JsonMapGeneric{
			"list": []PublicAccountInfo{},
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
