package wecommod

import (
	"context"
	"net/http"

	"github.com/go-chi/render"
	"github.com/silenceper/wechat/v2/work/kf"
)

// KfList - 按场景获取客服列表.
func (h Handler) GetAccounts(w http.ResponseWriter, r *http.Request) {
	list, err := h.Kefu.AccountList()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, list.CommonError)
	}

	scene := r.URL.Query().Get("scene")
	if len(scene) == 0 {
		scene = "default"
	}

	retList := make([]map[string]string, 0)
	for _, item := range list.AccountList {
		info, err := h.Kefu.AddContactWay(kf.AddContactWayOptions{
			OpenKFID: item.OpenKFID,
			Scene:    scene,
		})
		if err != nil {
			continue
		}
		retList = append(retList, map[string]string{
			"name":   item.Name,
			"avatar": item.Avatar,
			"url":    info.URL,
		})
	}

	r = r.WithContext(context.WithValue(
		r.Context(), render.StatusCtxKey, http.StatusOK))

	render.JSON(w, r, retList)
}
