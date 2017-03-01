package accounts

import (
	"github.com/dakom/basic-site-api/lib/pages"

	_ "image/gif"
	_ "image/png"
)

////////////////////////////////////////////////////////////////////////////////////////
////THESE DO HEAVY LIFTING IN USERS.* BECAUSE IT NEEDS TO BE CALLED FROM OAUTH ALSO!////
////////////////////////////////////////////////////////////////////////////////////////

func GotRefreshTokenRequest(rData *pages.RequestData) {
	rData.SetJsonSuccessResponse(nil)
}
