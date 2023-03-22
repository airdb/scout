package wecommod

import (
	"context"
	"io"
	"net/http"

	"github.com/silenceper/wechat/v2/work/kf"
)

// GetCallback - recieve wxkf's notifies.
func (h Handler) GetCallback(w http.ResponseWriter, r *http.Request) {
	opts := kf.SignatureOptions{
		Signature: r.URL.Query().Get("msg_signature"),
		TimeStamp: r.URL.Query().Get("timestamp"),
		Nonce:     r.URL.Query().Get("nonce"),
		EchoStr:   r.URL.Query().Get("echostr"),
	}

	if len(opts.EchoStr) > 0 {
		data, err := h.Kefu.VerifyURL(opts)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write(nil)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(data))

		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.Logger.With("err", err).Info("can not read request body")
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("success"))

	ctx := context.TODO()
	cbMsg, err := h.Kefu.GetCallbackMessage(body)
	if err != nil {
		h.Logger.With("body", body, "err", err).Info("can not decrypt callback")
	}
	cmd := h.Redis.SIsMember(ctx, SyncMsgTokenProcessed, cbMsg.Token)
	if cmd.Val() {
		return
	}
	h.Redis.SAdd(ctx, SyncMsgTokenProcessed, cbMsg.Token)
	h.Logger.With("msg", cbMsg).Info("new callback message")

	syncMsgOpts := kf.SyncMsgOptions{Token: cbMsg.Token}
	// 获取上次消息游标
	cursor, err := h.Redis.Get(ctx, SyncMsgNextCursor).Result()
	if err == nil && len(cursor) > 0 {
		syncMsgOpts.Cursor = cursor
	}

	//h.Redis.Set(ctx, SyncMsgNextCursor, cbMsg.Token, 0)
	syncMsg, err := h.Kefu.SyncMsg(syncMsgOpts)
	if err == nil {
		// 保存本次消息游标
		cmd := h.Redis.Set(ctx, SyncMsgNextCursor, syncMsg.NextCursor, 0)
		if err := cmd.Err(); err != nil {
			h.Logger.With("err", err).Info("set next cursor error")
		}
	} else {
		h.Logger.With("err", err).Info("can not sync msg")
		// 清空游标，不然下次报游标错误
		h.Redis.Del(ctx, SyncMsgNextCursor)
	}

	ctx = WithCache(ctx, h.Redis)
	ctx = WithWekf(ctx, h.Kefu)
	ctx = WithLogger(ctx, h.Logger)

	for _, msg := range syncMsg.MsgList {
		cmd := h.Redis.SIsMember(ctx, SyncMsgProcessed, msg.MsgID)
		if cmd.Val() {
			continue
		}
		h.Redis.SAdd(ctx, SyncMsgProcessed, msg.MsgID)
		h.Logger.With("msg", msg.GetOriginMessage()).Info("sync from wechat")
		h.ReplySvc.ProcMsg(ctx, msg)
	}

}
