package handler

import (
	"crypto/sha1"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/tencentyun/cos-go-sdk-v5"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"kkg-backend/internal/oj/common"
)

func (h *Handler) FileUpload(c *gin.Context) {
	login := h.mustLoginUser(c)
	biz := strings.TrimSpace(c.PostForm("biz"))
	if biz == "" {
		panic(common.NewBizError(common.ParamsError, "请求参数错误"))
	}
	file, err := c.FormFile("file")
	mustNoErr(err)
	validUploadFile(file.Size, file.Filename, biz)
	now := time.Now().Format("20060102150405")
	dst := fmt.Sprintf("/tmp/%s/%d/%s-%s", biz, login.ID, now, filepath.Base(file.Filename))
	if h.cfg.COS.Enabled && h.cfg.COS.BucketURL != "" && h.cfg.COS.SecretID != "" && h.cfg.COS.SecretKey != "" {
		key := fmt.Sprintf("/%s/%d/%s-%s", biz, login.ID, now, filepath.Base(file.Filename))
		url, err := h.uploadToCOS(c, file, key)
		mustNoErr(err)
		c.JSON(http.StatusOK, common.Success(url))
		return
	}
	mustNoErr(os.MkdirAll(filepath.Dir(dst), 0o755))
	mustNoErr(c.SaveUploadedFile(file, dst))
	c.JSON(http.StatusOK, common.Success("file://"+dst))
}
func (h *Handler) WxGet(c *gin.Context) {
	echostr := c.Query("echostr")
	if h.cfg.WX.MpToken == "" {
		c.String(http.StatusOK, echostr)
		return
	}
	signature := c.Query("signature")
	timestamp := c.Query("timestamp")
	nonce := c.Query("nonce")
	if wxCheckSignature(h.cfg.WX.MpToken, signature, timestamp, nonce) {
		c.String(http.StatusOK, echostr)
		return
	}
	c.String(http.StatusOK, "")
}
func (h *Handler) WxPost(c *gin.Context) {
	var msg wxInMessage
	body, _ := io.ReadAll(c.Request.Body)
	if xml.Unmarshal(body, &msg) != nil {
		c.String(http.StatusOK, "")
		return
	}
	reply := "感谢关注"
	if msg.MsgType == "text" && strings.TrimSpace(msg.Content) != "" {
		reply = "收到：" + msg.Content
	}
	if msg.MsgType == "event" && strings.EqualFold(msg.Event, "CLICK") {
		reply = "你点击了菜单：" + msg.EventKey
	}
	out := wxOutMessage{
		XMLName:      xml.Name{Local: "xml"},
		ToUserName:   cdata(msg.FromUserName),
		FromUserName: cdata(msg.ToUserName),
		CreateTime:   time.Now().Unix(),
		MsgType:      cdata("text"),
		Content:      cdata(reply),
	}
	xmlBytes, _ := xml.Marshal(out)
	c.Header("Content-Type", "application/xml;charset=utf-8")
	c.String(http.StatusOK, string(xmlBytes))
}
func (h *Handler) WxSetMenu(c *gin.Context) { c.JSON(http.StatusOK, common.Success("ok")) }

type cdata string

func (c cdata) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.EncodeElement(struct {
		Text string `xml:",cdata"`
	}{Text: string(c)}, start)
}

type wxInMessage struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType"`
	Content      string   `xml:"Content"`
	Event        string   `xml:"Event"`
	EventKey     string   `xml:"EventKey"`
}

type wxOutMessage struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   cdata    `xml:"ToUserName"`
	FromUserName cdata    `xml:"FromUserName"`
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      cdata    `xml:"MsgType"`
	Content      cdata    `xml:"Content"`
}

func wxCheckSignature(token, signature, timestamp, nonce string) bool {
	arr := []string{token, timestamp, nonce}
	sort.Strings(arr)
	sum := sha1.Sum([]byte(strings.Join(arr, "")))
	return fmt.Sprintf("%x", sum) == signature
}

func (h *Handler) fetchWxOpenUserInfo(code string) (unionID, openID string, err error) {
	tokenURL := fmt.Sprintf("https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code", h.cfg.WX.OpenAppID, h.cfg.WX.OpenSecret, code)
	resp, err := http.Get(tokenURL)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var tk map[string]interface{}
	if json.Unmarshal(body, &tk) != nil {
		return "", "", errors.New("wx token parse error")
	}
	accessToken, _ := tk["access_token"].(string)
	openID, _ = tk["openid"].(string)
	unionID, _ = tk["unionid"].(string)
	if accessToken == "" || openID == "" {
		return unionID, openID, errors.New("wx token missing")
	}
	infoURL := fmt.Sprintf("https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s", accessToken, openID)
	resp2, err := http.Get(infoURL)
	if err != nil {
		return unionID, openID, err
	}
	defer resp2.Body.Close()
	body2, _ := io.ReadAll(resp2.Body)
	var info map[string]interface{}
	if json.Unmarshal(body2, &info) == nil {
		if u, ok := info["unionid"].(string); ok && u != "" {
			unionID = u
		}
		if o, ok := info["openid"].(string); ok && o != "" {
			openID = o
		}
	}
	return unionID, openID, nil
}

func (h *Handler) uploadToCOS(c *gin.Context, fileHeader *multipart.FileHeader, key string) (string, error) {
	u, err := url.Parse(h.cfg.COS.BucketURL)
	if err != nil {
		return "", err
	}
	b := &cos.BaseURL{BucketURL: u}
	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  h.cfg.COS.SecretID,
			SecretKey: h.cfg.COS.SecretKey,
		},
	})
	f, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()
	_, err = client.Object.Put(c.Request.Context(), key, f, nil)
	if err != nil {
		return "", err
	}
	host := strings.TrimRight(h.cfg.COS.Host, "/")
	if host != "" {
		return host + key, nil
	}
	return h.cfg.COS.BucketURL + key, nil
}
