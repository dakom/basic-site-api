package accounts

import (
	"encoding/base64"
	"image"
	"image/jpeg"
	"io"
	"strconv"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/nfnt/resize"
	"github.com/oliamb/cutter"

	"github.com/dakom/basic-site-api/setup/config/static/statuscodes"
	"github.com/dakom/basic-site-api/lib/datastore"
	"github.com/dakom/basic-site-api/lib/pages"
)

func GotAvatarFileChangeServiceRequest(rData *pages.RequestData) {
	//default is 128x128
	file, _, err := rData.HttpRequest.FormFile("file")
	if err != nil {
		rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
		return
	}
	defer file.Close()

	srcImage, _, err := image.Decode(io.LimitReader(file, rData.SiteConfig.MAX_READ_SIZE))
	if err != nil {
		rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
		return
	}

	err = UpdateAvatar(rData, srcImage, rData.UserRecord)
	if err != nil {
		rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
		return
	}

	rData.SetJsonSuccessCodeResponse(statuscodes.AVATAR_CHANGED)
}

func GotAvatarBase64ChangeServiceRequest(rData *pages.RequestData) {
	reader := base64.NewDecoder(base64.URLEncoding, io.LimitReader(strings.NewReader(rData.HttpRequest.FormValue("imgbytes")), rData.SiteConfig.MAX_READ_SIZE))
	srcImage, _, err := image.Decode(reader)
	if err != nil {
		rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
		return
	}

	err = UpdateAvatar(rData, srcImage, rData.UserRecord)
	if err != nil {
		rData.SetJsonErrorCodeResponse(statuscodes.TECHNICAL)
		return
	}

	rData.SetJsonSuccessResponse(nil)

}

func UpdateAvatar(rData *pages.RequestData, srcImage image.Image, userRecord *datastore.UserRecord) error {
	jpegOptions := &jpeg.Options{
		Quality: 90,
	}

	jpegSize := uint(128)
	jpegSmallSize := uint(32)

	newAvatarId := userRecord.GetData().AvatarId + 1
	//store original too so in case we need to batch convert them at some point for new avatar sizes or whatever
	newFilenameOriginal := avatarFilename(userRecord.GetKey().IntID(), newAvatarId, "-orig")
	newFilename := avatarFilename(userRecord.GetKey().IntID(), newAvatarId, "")
	newFilenameSmall := avatarFilename(userRecord.GetKey().IntID(), newAvatarId, "_32")

	//crop image to center square
	croppedImg, err := cutter.Crop(srcImage, cutter.Config{
		Width:   1,
		Height:  1,
		Mode:    cutter.Centered,
		Options: cutter.Ratio,
	})

	if err != nil {
		rData.LogError("error on image crop: %v", err)
		rData.SetHttpStatusResponse(400, statuscodes.MISSINGINFO)
		return err
	}

	//resize square for standard avatar size
	resizedImg := resize.Resize(jpegSize, jpegSize, croppedImg, resize.Lanczos3)
	resizedImgSmall := resize.Resize(jpegSmallSize, jpegSmallSize, croppedImg, resize.Lanczos3)

	//open connection to google cloud storage
	client, err := storage.NewClient(rData.Ctx)
	if err != nil {
		rData.LogError("failed to get gcs client: %v", err)
		rData.SetHttpStatusResponse(400, statuscodes.MISSINGINFO)
		return err
	}

	defer client.Close()

	bucket := client.Bucket(rData.SiteConfig.GCS_BUCKET_AVATAR)

	originalObject := bucket.Object(newFilenameOriginal)

	originalWriter := originalObject.NewWriter(rData.Ctx)
	originalWriter.ContentType = "image/jpeg"
	defer originalWriter.Close()

	err = jpeg.Encode(originalWriter, croppedImg, jpegOptions)
	if err != nil {
		rData.LogError("failed to save original cropped image: %v", err)
		rData.SetHttpStatusResponse(400, statuscodes.MISSINGINFO)
		return err
	}

	//main resized avatar
	resizedObject := bucket.Object(newFilename)
	resizedWriter := resizedObject.NewWriter(rData.Ctx)
	resizedWriter.ContentType = "image/jpeg"
	defer resizedWriter.Close()

	err = jpeg.Encode(resizedWriter, resizedImg, jpegOptions)
	if err != nil {
		rData.LogError("failed to save resized image: %v", err)
		rData.SetHttpStatusResponse(400, statuscodes.MISSINGINFO)
		return err
	}

	//small resized avatar
	resizedObjectSmall := bucket.Object(newFilenameSmall)
	resizedWriterSmall := resizedObjectSmall.NewWriter(rData.Ctx)
	resizedWriterSmall.ContentType = "image/jpeg"
	defer resizedWriterSmall.Close()

	err = jpeg.Encode(resizedWriterSmall, resizedImgSmall, jpegOptions)
	if err != nil {
		rData.LogError("failed to save resized image: %v", err)
		rData.SetHttpStatusResponse(400, statuscodes.MISSINGINFO)
		return err
	}

	//save new avatar id to datastore
	userRecord.GetData().AvatarId = newAvatarId

	err = datastore.Save(rData.Ctx, userRecord)
	if err != nil {
		rData.LogError("failed to save update user record: %v", err)
		rData.SetHttpStatusResponse(400, statuscodes.MISSINGINFO)
		return err
	}

	//delete the old one
	oldAvatarId := newAvatarId - 1

	if oldAvatarId > 0 {
		oldFilenameOriginal := avatarFilename(userRecord.GetKey().IntID(), oldAvatarId, "-orig")
		oldFilenameSmall := avatarFilename(userRecord.GetKey().IntID(), oldAvatarId, "_32")
		oldFilename := avatarFilename(userRecord.GetKey().IntID(), oldAvatarId, "")

		oldObjectOriginal := bucket.Object(oldFilenameOriginal)
		oldObjectSmall := bucket.Object(oldFilenameSmall)
		oldObject := bucket.Object(oldFilename)

		err = oldObjectOriginal.Delete(rData.Ctx)
		if err != nil {
			rData.LogError("failed to delete object, but still created new one: %v", err)

		}
		err = oldObjectSmall.Delete(rData.Ctx)
		if err != nil {
			rData.LogError("failed to delete object, but still created new one: %v", err)

		}
		err = oldObject.Delete(rData.Ctx)
		if err != nil {
			rData.LogError("failed to delete object, but still created new one: %v", err)

		}

	}

	return nil
}

func avatarFilename(userID int64, avatarID int64, suffix string) string {
	return strconv.FormatInt(userID, 10) + "/" + strconv.FormatInt(avatarID, 10) + suffix + ".jpg"
}
