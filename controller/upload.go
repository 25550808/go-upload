package controller

import (
	"crypto/md5"
	"io"
	"encoding/hex"
	"path"
	"os"
	"net/http"
	"strings"
	"image/jpeg"
	"image"
	"image/png"
	"image/gif"
	"errors"
	"strconv"
	"mime/multipart"
	"github.com/nfnt/resize"
	"github.com/axetroy/go-fs"
	"github.com/axetroy/go-upload/config"
	"github.com/gin-gonic/gin"
)

const FIELD = "file"

// 支持的图片后缀名
var supportImageExtNames = []string{".jpg", ".jpeg", ".png", ".ico", ".svg", ".bmp", ".gif"}

/**
Generate thumbnail
 */
func thumbnailify(imagePath string) (outputPath string, err error) {
	var (
		file     *os.File
		img      image.Image
		filename = path.Base(imagePath)
	)

	extname := strings.ToLower(path.Ext(imagePath))

	outputPath = path.Join(config.Upload.Path, config.Upload.Image.Thumbnail.Path, filename)

	// 读取文件
	if file, err = os.Open(imagePath); err != nil {
		return
	}

	defer file.Close()

	// decode jpeg into image.Image
	switch extname {
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(file)
		break
	case ".png":
		img, err = png.Decode(file)
		break
	case ".gif":
		img, err = gif.Decode(file)
		break
	default:
		err = errors.New("Unsupport file type" + extname)
		return
	}

	if img == nil {
		err = errors.New("Generate thumbnail fail...")
		return
	}

	m := resize.Thumbnail(uint(config.Upload.Image.Thumbnail.MaxWidth), uint(config.Upload.Image.Thumbnail.MaxHeight), img, resize.Lanczos3)

	out, err := os.Create(outputPath)
	if err != nil {
		return
	}
	defer out.Close()

	// write new image to file

	//decode jpeg/png/gif into image.Image
	switch extname {
	case ".jpg", ".jpeg":
		jpeg.Encode(out, m, nil)
		break
	case ".png":
		png.Encode(out, m)
		break
	case ".gif":
		gif.Encode(out, m, nil)
		break
	default:
		err = errors.New("Unsupport file type" + extname)
		return
	}

	return
}

/**
check a file is a image or not
 */
func isImage(extName string) bool {
	for i := 0; i < len(supportImageExtNames); i++ {
		if supportImageExtNames[i] == extName {
			return true
		}
	}
	return false
}

/**
Handler the parse error
 */
func parseFormFail(context *gin.Context) {
	context.JSON(http.StatusBadRequest, gin.H{
		"message": "Can not parse form",
	})
}

/**
Upload image handler
 */
func UploaderImage(context *gin.Context) {
	var (
		maxUploadSize = config.Upload.Image.MaxSize // 最大上传大小
		distPath      string                        // 最终的输出目录
		err           error
		file          *multipart.FileHeader
		src           multipart.File
		dist          *os.File
	)

	// Source
	if file, err = context.FormFile(FIELD); err != nil {
		parseFormFail(context)
		return
	}

	extname := strings.ToLower(path.Ext(file.Filename))

	if isImage(extname) == false {
		context.JSON(http.StatusBadRequest, gin.H{
			"message": "Unsupport upload file type " + extname,
		})
		return
	}

	if file.Size > int64(maxUploadSize) {
		context.JSON(http.StatusBadRequest, gin.H{
			"message": "Upload file too large, The max upload limit is " + strconv.Itoa(int(maxUploadSize)),
		})
		return
	}

	if src, err = file.Open(); err != nil {

	}
	defer src.Close()

	hash := md5.New()

	io.Copy(hash, src)

	md5string := hex.EncodeToString(hash.Sum([]byte("")))

	fileName := md5string + extname

	// Destination
	distPath = path.Join(config.Upload.Path, config.Upload.Image.Path, fileName)
	if dist, err = os.Create(distPath); err != nil {

	}
	defer dist.Close()

	// FIXME: open 2 times
	if src, err = file.Open(); err != nil {
		//
	}

	// Copy
	io.Copy(dist, src)

	// 压缩缩略图
	// 不管成功与否，都会进行下一步的返回
	if _, err := thumbnailify(distPath); err != nil {
		if config.Config.Mode != gin.ReleaseMode {
			panic(err)
		}
	}

	context.JSON(http.StatusOK, gin.H{
		"hash":     md5string,
		"filename": fileName,
		"origin":   file.Filename,
		"size":     file.Size,
	})
}

/**
Upload file handler
 */
func UploadFile(context *gin.Context) {
	var (
		isSupportFile bool
		maxUploadSize = config.Upload.Image.MaxSize  // 最大上传大小
		allowTypes    = config.Upload.File.AllowType // 可上传的文件类型
		distPath      string                         // 最终的输出目录
		err           error
		file          *multipart.FileHeader
		src           multipart.File
		dist          *os.File
	)
	// Source
	if file, err = context.FormFile(FIELD); err != nil {
		parseFormFail(context)
		return
	}

	extname := path.Ext(file.Filename)

	if len(allowTypes) != 0 {
		for i := 0; i < len(allowTypes); i++ {
			if allowTypes[i] == extname {
				isSupportFile = true
				break
			}
		}

		if isSupportFile == false {
			context.JSON(http.StatusBadRequest, gin.H{
				"message": "Unsupport upload file type " + extname,
			})
			return
		}
	}

	if file.Size > int64(maxUploadSize) {
		context.JSON(http.StatusBadRequest, gin.H{
			"message": "Upload file too large, The max upload limit is " + strconv.Itoa(int(maxUploadSize)),
		})
		return
	}

	if src, err = file.Open(); err != nil {
		// open the file fail...
	}
	defer src.Close()

	hash := md5.New()

	io.Copy(hash, src)

	md5string := hex.EncodeToString(hash.Sum([]byte("")))

	fileName := md5string + extname

	// Destination
	distPath = path.Join(config.Upload.Path, config.Upload.File.Path, fileName)
	if dist, err = os.Create(distPath); err != nil {
		// create dist file fail...
	}
	defer dist.Close()

	// FIXME: open 2 times
	if src, err = file.Open(); err != nil {
		//
	}

	// Copy
	io.Copy(dist, src)

	context.JSON(http.StatusOK, gin.H{
		"hash":     md5string,
		"filename": fileName,
		"origin":   file.Filename,
		"size":     file.Size,
	})
}

/**
Generate Upload example Template
 */
func UploaderTemplate(template string) (func(context *gin.Context)) {
	return func(context *gin.Context) {
		header := context.Writer.Header()
		header.Set("Content-Type", "text/html; charset=utf-8")
		context.String(200, `
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Upload</title>
</head>
<body>
<form action="/upload/image" method="post" enctype="multipart/form-data">
  <h2>Image Upload</h2>
  <input type="file" name="file">
  <input type="submit" value="Upload">
</form>

</hr>

<form action="/upload/file" method="post" enctype="multipart/form-data">
  <h2>File Upload</h2>
  <input type="file" name="file">
  <input type="submit" value="Upload">
</form>

</body>
</html>
	`)
	}

}

/**
Get Origin image
 */
func GetOriginImage(context *gin.Context) {
	filename := context.Param("filename")
	originImagePath := path.Join(config.Upload.Path, config.Upload.Image.Path, filename)
	if fs.PathExists(originImagePath) == false {
		// if the path not found
		http.NotFound(context.Writer, context.Request)
		return
	}
	http.ServeFile(context.Writer, context.Request, originImagePath)
}

/**
Get thumbnail image
 */
func GetThumbnailImage(context *gin.Context) {
	filename := context.Param("filename")
	originImagePath := path.Join(config.Upload.Path, config.Upload.Image.Path, filename)
	thumbnailImagePath := path.Join(config.Upload.Path, config.Upload.Image.Thumbnail.Path, filename)
	if fs.PathExists(thumbnailImagePath) == false {
		// if thumbnail image not exist, try to get origin image
		if fs.PathExists(originImagePath) == true {
			http.ServeFile(context.Writer, context.Request, originImagePath)
			return
		}
		// if the path not found
		http.NotFound(context.Writer, context.Request)
		return
	}
	http.ServeFile(context.Writer, context.Request, thumbnailImagePath)
}

/**
Get file raw
 */
func GetFileRaw(context *gin.Context) {
	filename := context.Param("filename")
	filePath := path.Join(config.Upload.Path, config.Upload.File.Path, filename)
	if isExistFile := fs.PathExists(filePath); isExistFile == false {
		// if the path not found
		http.NotFound(context.Writer, context.Request)
		return
	}
	http.ServeFile(context.Writer, context.Request, filePath)
}

/**
Download a file
 */
func DownloadFile(context *gin.Context) {
	filename := context.Param("filename")
	filePath := path.Join(config.Upload.Path, config.Upload.File.Path, filename)
	if isExistFile := fs.PathExists(filePath); isExistFile == false {
		// if the path not found
		http.NotFound(context.Writer, context.Request)
		return
	}
	http.ServeFile(context.Writer, context.Request, filePath)
}
