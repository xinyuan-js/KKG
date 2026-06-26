package handler

import (
	"errors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"strings"
	"yuoj-go-backend/internal/common"
	"yuoj-go-backend/internal/model/entity"
)

type userRegisterReq struct{ UserAccount, UserPassword, CheckPassword string }
type userLoginReq struct{ UserAccount, UserPassword string }

func (h *Handler) UserRegister(c *gin.Context) {
	var req userRegisterReq
	mustBindJSON(c, &req)
	id, err := h.userSvc.Register(req.UserAccount, req.UserPassword, req.CheckPassword)
	mustNoErr(err)
	c.JSON(http.StatusOK, common.Success(id))
}

func (h *Handler) UserLogin(c *gin.Context) {
	var req userLoginReq
	mustBindJSON(c, &req)
	u, err := h.userSvc.Login(req.UserAccount, req.UserPassword)
	mustNoErr(err)
	sess := sessions.Default(c)
	sess.Set(common.UserLoginState, u.ID)
	_ = sess.Save()
	c.JSON(http.StatusOK, common.Success(loginUserVO(u)))
}

func (h *Handler) UserLogout(c *gin.Context) {
	sess := sessions.Default(c)
	sess.Delete(common.UserLoginState)
	_ = sess.Save()
	c.JSON(http.StatusOK, common.Success(true))
}

func (h *Handler) UserGetLogin(c *gin.Context) {
	c.JSON(http.StatusOK, common.Success(loginUserVO(h.mustLoginUser(c))))
}
func (h *Handler) UserWxLogin(c *gin.Context) {
	code := strings.TrimSpace(c.Query("code"))
	if code == "" {
		panic(common.NewBizError(common.ParamsError, "请求参数错误"))
	}
	unionID := "wx_union_" + code
	mpOpenID := "wx_open_" + code
	if h.cfg.WX.OpenAppID != "" && h.cfg.WX.OpenSecret != "" {
		realUnion, realOpen, err := h.fetchWxOpenUserInfo(code)
		if err == nil {
			if realUnion != "" {
				unionID = realUnion
			}
			if realOpen != "" {
				mpOpenID = realOpen
			}
		}
	}
	var u entity.User
	err := h.db.Where("unionId = ? AND isDelete = 0", unionID).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		u = entity.User{
			UnionID:    unionID,
			MpOpenID:   mpOpenID,
			UserName:   "wx_" + code,
			UserRole:   common.DefaultRole,
			UserAvatar: "",
		}
		mustNoErr(h.db.Create(&u).Error)
	} else {
		mustNoErr(err)
	}
	if u.UserRole == common.BanRole {
		panic(common.NewBizError(common.ForbiddenError, "该用户已被封，禁止登录"))
	}
	sess := sessions.Default(c)
	sess.Set(common.UserLoginState, u.ID)
	_ = sess.Save()
	c.JSON(http.StatusOK, common.Success(loginUserVO(&u)))
}

func (h *Handler) UserAdd(c *gin.Context) {
	login := h.mustLoginUser(c)
	var u entity.User
	mustBindJSON(c, &u)
	mustNoErr(h.userSvc.CreateByAdmin(login, &u))
	c.JSON(http.StatusOK, common.Success(u.ID))
}

func (h *Handler) UserDelete(c *gin.Context) {
	login := h.mustLoginUser(c)
	var req common.DeleteRequest
	mustBindJSON(c, &req)
	mustNoErr(h.userSvc.SoftDeleteBySuperAdmin(login, req.ID))
	c.JSON(http.StatusOK, common.Success(true))
}

func (h *Handler) UserUpdate(c *gin.Context) {
	login := h.mustLoginUser(c)
	var u entity.User
	mustBindJSON(c, &u)
	mustNoErr(h.userSvc.UpdateByAdmin(login, &u))
	c.JSON(http.StatusOK, common.Success(true))
}

func (h *Handler) UserGet(c *gin.Context) {
	h.mustAdmin(c)
	id := parseIDQuery(c)
	u, err := h.userSvc.GetByID(id)
	mustNoErr(err)
	c.JSON(http.StatusOK, common.Success(u))
}
func (h *Handler) UserGetVO(c *gin.Context) {
	id := parseIDQuery(c)
	u, err := h.userSvc.GetByID(id)
	mustNoErr(err)
	c.JSON(http.StatusOK, common.Success(userVO(u)))
}
func (h *Handler) UserList(c *gin.Context) {
	h.mustAdmin(c)
	var req common.PageRequest
	mustBindJSON(c, &req)
	req.Normalize()
	list, total, err := h.userSvc.List(req.Current, req.PageSize)
	mustNoErr(err)
	c.JSON(http.StatusOK, common.Success(common.PageResult{Records: list, Total: total, Current: req.Current, Size: req.PageSize}))
}
func (h *Handler) UserListVO(c *gin.Context) {
	var req common.PageRequest
	mustBindJSON(c, &req)
	req.Normalize()
	if req.PageSize > 20 {
		panic(common.NewBizError(common.ParamsError, "请求参数错误"))
	}
	list, total, err := h.userSvc.List(req.Current, req.PageSize)
	mustNoErr(err)
	vos := make([]map[string]interface{}, 0, len(list))
	for i := range list {
		vos = append(vos, userVO(&list[i]))
	}
	c.JSON(http.StatusOK, common.Success(common.PageResult{Records: vos, Total: total, Current: req.Current, Size: req.PageSize}))
}
func (h *Handler) UserUpdateMy(c *gin.Context) {
	login := h.mustLoginUser(c)
	var req entity.User
	mustBindJSON(c, &req)
	mustNoErr(h.userSvc.UpdateProfile(login.ID, &req))
	c.JSON(http.StatusOK, common.Success(true))
}
