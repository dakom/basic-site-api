package accounts

import (
	"github.com/dakom/basic-site-api/lib/pages"

	_ "image/gif"
	_ "image/png"

	"strconv"
)

func GotSettingsInfoServiceRequest(rData *pages.RequestData) {
	rData.SetJsonSuccessResponse(pages.JsonMapGeneric{
		"uid":   rData.UserRecord.GetKeyIntAsString(),
		"email": rData.UserRecord.GetData().Email,
		"fname": rData.UserRecord.GetData().FirstName,
		"lname": rData.UserRecord.GetData().LastName,
		"avid":  strconv.FormatInt(rData.UserRecord.GetData().AvatarId, 10),
	})
}
