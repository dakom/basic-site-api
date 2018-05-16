package accounts

import (
	"github.com/dakom/basic-site-api/lib/pages"

	"encoding/json"
	"github.com/dakom/basic-site-api/lib/datastore"
	_ "image/gif"
	_ "image/png"

	"strconv"
)

type UserInfo struct {
    Id string `json:"uid"`
    Email string `json:"email"`
    FirstName string `json:"fname"`
    LastName string `json:"lname"`
    DisplayName string `json:"dname"`
    AvatarId string `json:"avid"`
    Jwt string `json:"jwt"`
    ErrorCode string `json:"code"`
}

func (u *UserInfo) GetString() (string, error) {
	jBytes, err := json.Marshal(u)
	if err != nil {
		return "", err
	}

	return string(jBytes), nil
}

func (u *UserInfo) SetJwt(jwt string) {
	u.Jwt = jwt
}
func (u *UserInfo) SetErrorCode(code string) {
	u.ErrorCode = code
}

func GetUserInfoFromRecord(userRecord *datastore.UserRecord) *UserInfo {
    return &UserInfo{
        Id:   userRecord.GetKeyIntAsString(),
        Email: userRecord.GetData().Email,
        FirstName: userRecord.GetData().FirstName,
        LastName: userRecord.GetData().LastName,
        DisplayName: userRecord.GetData().DisplayName,
        AvatarId: strconv.FormatInt(userRecord.GetData().AvatarId, 10),
    }
}

func GotSettingsInfoServiceRequest(rData *pages.RequestData) {
    userInfo := GetUserInfoFromRecord(rData.UserRecord)
	rData.SetJsonSuccessResponse(userInfo)
}
