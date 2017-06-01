package ping

import (
	"time"

	"github.com/dakom/basic-site-api/lib/pages"

	"strconv"
)

func GotPongRequest(rData *pages.RequestData) {
	pong := strconv.FormatInt(time.Now().Unix(), 10)

	rData.SetJsonSuccessResponse(pages.JsonMapGeneric{
		"pong": pong,
	})
}
