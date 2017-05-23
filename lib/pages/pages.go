package pages

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"strconv"

	"github.com/dakom/basic-site-api/lib/datastore"
	"github.com/dakom/basic-site-api/setup/config/custom"
	"golang.org/x/net/context"
	"google.golang.org/appengine/log"
)

type PageConfig struct {
	PageName             string
	Handler              func(*RequestData)
	HandlerType          int
	RequestSource        string
	Scopes               uint64
	RequiresDBScopeCheck bool
	AcceptAnyScope       bool
	SkipCsrfCheck        bool
}

type RequestData struct {
	Ctx                       context.Context
	SiteConfig                *custom.Config
	UserRecord                *datastore.UserRecord
	HttpWriter                http.ResponseWriter
	HttpRequest               *http.Request
	JsonResponse              JsonResponse
	HttpStatusResponseMessage string
	HttpStatusResponseBytes   []byte
	HttpStatusResponseCode    int
	HttpRedirectDestination   string
	HttpRedirectIsPermanent   bool
	PageConfig                *PageConfig
	ExtraUrlParams            []string
	JwtRecord                 *datastore.JwtRecord
	JwtString                 string
	DeleteJwtWhenFinished     bool
}

type JsonResponse interface {
	GetString() (string, error)
	SetJwt(string)
	SetErrorCode(string)
}

type JsonMapGeneric map[string]interface{}

func (j JsonMapGeneric) GetString() (string, error) {
	jBytes, err := json.Marshal(j)
	if err != nil {
		return "", err
	}

	return string(jBytes), nil
}

func (j JsonMapGeneric) SetJwt(jwt string) {
	j["jwt"] = jwt
}
func (j JsonMapGeneric) SetErrorCode(code string) {
	j["code"] = code
}

type TemplateData struct {
	CacheBuster int64
	PageData    interface{}
}

const (
	HANDLER_TYPE_JSON = iota
	HANDLER_TYPE_HTTP_STATUS
	HANDLER_TYPE_HTML_STRINGS
	HANDLER_TYPE_HTTP_REDIRECT
)

func (rData *RequestData) LogInfo(format string, args ...interface{}) {
	log.Infof(rData.Ctx, format, args...)
}

func (rData *RequestData) LogError(format string, args ...interface{}) {
	log.Errorf(rData.Ctx, format, args...)
}

func (rData *RequestData) SetHttpStatusResponse(code int, msg string, args ...interface{}) {
	rData.HttpStatusResponseMessage = fmt.Sprintf(msg, args...)
	rData.HttpStatusResponseCode = code
}

func (rData *RequestData) SetContentType(contentType string) {
	rData.HttpWriter.Header().Set("Content-Type", contentType)
}

func (rData *RequestData) SetJsonSuccessResponse(jsonResponse JsonResponse) {
	setJsonCodeResponse(rData, 200, jsonResponse, "")
}
func (rData *RequestData) SetJsonErrorResponse(jsonResponse JsonResponse) {
	setJsonCodeResponse(rData, 400, jsonResponse, "")
}

func (rData *RequestData) SetJsonSuccessCodeResponse(code string) {
	setJsonCodeResponse(rData, 200, nil, code)
}

func (rData *RequestData) SetJsonErrorCodeResponse(code string) {
	setJsonCodeResponse(rData, 400, nil, code)
}

func (rData *RequestData) SetJsonSuccessCodeWithDataResponse(code string, jsonResponse JsonResponse) {
	setJsonCodeResponse(rData, 200, jsonResponse, code)
}

func (rData *RequestData) SetJsonErrorCodeWithDataResponse(code string, jsonResponse JsonResponse) {
	setJsonCodeResponse(rData, 400, jsonResponse, code)
}

func setJsonCodeResponse(rData *RequestData, httpStatus int, jsonResponse JsonResponse, errorCode string) {
	if httpStatus != -1 {
		rData.HttpStatusResponseCode = httpStatus
	}
	if jsonResponse == nil {
		jsonResponse = make(JsonMapGeneric)
	}

	if errorCode != "" {
		jsonResponse.SetErrorCode(errorCode)
	}

	rData.JsonResponse = jsonResponse
}

func (rData *RequestData) OutputJsonString() error {
	var err error

	rData.SetContentType("application/json; charset=utf-8")

	rData.HttpStatusResponseMessage, err = rData.JsonResponse.GetString()
	if err != nil {
		rData.HttpStatusResponseCode = 400
	}

	rData.OutputHttpResponse()

	return err
}

func (rData *RequestData) OutputImageJpg(img *image.Image) error {
	buffer := new(bytes.Buffer)
	if err := jpeg.Encode(buffer, *img, nil); err != nil {
		return err
	}

	rData.SetContentType("image/jpeg")
	rData.HttpWriter.Header().Set("Content-Length", strconv.Itoa(len(buffer.Bytes())))
	if _, err := rData.HttpWriter.Write(buffer.Bytes()); err != nil {
		return err
	}
	return nil
}

func (rData *RequestData) OutputHttpResponse() {
	rData.HttpWriter.WriteHeader(rData.HttpStatusResponseCode)
	if rData.HttpStatusResponseMessage != "" {
		io.WriteString(rData.HttpWriter, rData.HttpStatusResponseMessage)
	}

	if len(rData.HttpStatusResponseBytes) > 0 {
		rData.HttpWriter.Write(rData.HttpStatusResponseBytes)
	}

}

func (rData *RequestData) OutputHtmlString(msg string, args ...interface{}) {
	fmt.Fprintf(rData.HttpWriter, msg, args...)
}
