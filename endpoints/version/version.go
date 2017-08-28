package version

import (
	"github.com/dakom/basic-site-api/lib/pages"
)

func GotVersionRequest(rData *pages.RequestData) {

	rData.SetJsonSuccessResponse(pages.JsonMapGeneric{
		"version": rData.SiteConfig.VERSION,
	})
}
