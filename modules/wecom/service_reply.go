package wecommod

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	openaimod "github.com/airdb/scout/modules/openai"
	"github.com/gofrs/uuid"
	"github.com/silenceper/wechat/v2/work/kf"
	"github.com/silenceper/wechat/v2/work/kf/sendmsg"
	"github.com/silenceper/wechat/v2/work/kf/syncmsg"
	"go.uber.org/fx"
	"golang.org/x/exp/slog"
)

type replySvcDeps struct {
	fx.In

	Logger    *slog.Logger
	ReplyTpls []*ReplyTpl
	ChatGpt   *openaimod.ChatGpt
}

type ReplySvc struct {
	deps *replySvcDeps
}

func NewReplySvc(deps replySvcDeps) *ReplySvc {
	return &ReplySvc{
		deps: &deps,
	}
}

// ProcMsg 处理单条消息, 并按消息来源颁发给不同的处理过程
func (s ReplySvc) ProcMsg(ctx context.Context, msg syncmsg.Message) { // 记录用户发送的消息
	s.saveMsg(ctx, &msg)
	switch msg.Origin {
	case 3: // 客户回复的消息
		s.userMsg(ctx, msg)
	case 4: // 系统推送的消息
		s.systemMsg(ctx, msg)
	case 5: // 接待人员在企业微信客户端发送的消息
		s.receptionistMsg(ctx, msg)
	default:
		log.Fatalf("unknown msg origin: %d", msg.Origin)
	}
}

// 处理客户回复的消息
func (s ReplySvc) userMsg(ctx context.Context, msg syncmsg.Message) {
	// 按整块消息进行匹配
	var rTpl *ReplyTpl
	for _, rt := range s.deps.ReplyTpls {
		if rt.Match(ctx, msg) {
			rTpl = rt
		}
	}
	if rTpl == nil {
		txtMsg, err := msg.GetTextMessage()
		if err != nil {
			return
		}
		msg, err := s.deps.ChatGpt.GetResponse(ctx, txtMsg.Text.Content)
		if err != nil {
			return
		}
		if len(msg) == 0 {
			return
		}
		// Default 默认消息
		rTpl = &ReplyTpl{
			ReplyType: ReplyTypeText,
			Message: fmt.Sprintf(
				"> %s\n\n-------------\n%s", txtMsg.Text.Content, msg),
		}
	}

	// 最终的冗余，这块代码应该不被执行
	if rTpl == nil {
		rTpl = &ReplyTpl{
			ReplyType: ReplyTypeText,
			Message:   DefaultMsg,
		}
	}

	var (
		msgResp      = rTpl.Gen(ctx, msg.ExternalUserID, msg.OpenKFID)
		hasMsgSendOk bool // 消息执行是否成功
	)

	ret, err := msgResp.Assume()
	if err != nil {
		log.Fatalf("can not assume msg: %s", err.Error())
		return
	}

	switch rTpl.ReplyType {
	case ReplyTypeText, ReplyTypeImage, ReplyTypeMenu:
		hasMsgSendOk = s.sendMsg(ctx, msg, ret)
	case ReplyTypeActionTrans: // 分配客服会话
		hasMsgSendOk = s.transMsg(ctx, msg, ret)
	}

	if hasMsgSendOk {
		s.saveMsg(ctx, msgResp)
	}
}

// 处理系统消息
func (s ReplySvc) systemMsg(ctx context.Context, msg syncmsg.Message) {
	var (
		ret          interface{}
		sentCackeKey string // 消息缓存key
		sentCackeTTL time.Duration
		redis        = MustFromCache(ctx)
	)

	switch msg.EventType {
	case "enter_session": // 用户进入会话事件
		tMsg, _ := msg.GetEnterSessionEvent()
		// 缓存上次该客户的欢迎消息发送，避免重复发送。
		sentCackeKey = strings.Join([]string{
			SentMsgPrefix, msg.EventType, tMsg.OpenKFID, tMsg.ExternalUserID,
		}, ":")
		sentCackeTTL = 6 * time.Hour
		// 检查最近6小时是否发送过
		lastSend, _ := redis.Get(ctx, sentCackeKey).Result()
		if len(lastSend) > 0 {
			return
		}

		uuid, err := uuid.NewV6()
		if err != nil {
			panic(err)
		}

		rMsg := &sendmsg.Text{
			Message: sendmsg.Message{
				ToUser:   tMsg.ExternalUserID,
				OpenKFID: tMsg.OpenKFID,
				MsgID:    strings.ReplaceAll(uuid.String(), "-", ""),
			},
			MsgType: "text",
		}
		rMsg.Text.Content = WelcomeMsg // 欢迎语
		ret = rMsg
	case "msg_send_fail": // 消息发送失败事件
		fallthrough
	case "servicer_status_change": // 客服人员接待状态变更事件
		fallthrough
	case "session_status_change": // 会话状态变更事件
		fallthrough
	default:
		s.deps.Logger.With("event_type", msg.EventType).Error("unknown system event type")
		return
	}

	if s.sendMsg(ctx, msg, ret) && len(sentCackeKey) > 0 {
		redis.Set(ctx, sentCackeKey, time.Now().String(), sentCackeTTL).Result()
	}
}

// 处理客服消息, 只需用入库
func (s ReplySvc) receptionistMsg(ctx context.Context, msg syncmsg.Message) {
	s.saveMsg(ctx, msg)
}

// 发送消息
func (s ReplySvc) sendMsg(ctx context.Context, msg syncmsg.Message, ret interface{}) bool {
	var (
		wekf = MustFromWekf(ctx)
	)
	params, _ := json.Marshal(ret)
	if info, err := wekf.SendMsg(ret); err == nil {
		log.Println("result:", msg.EventType, info.MsgID, ", msg:", string(params))

		return true
	} else {
		log.Println("result:", msg.EventType, ", err:", err.Error(), ", params: ", string(params))

		return false
	}
}

// 分配客服会话
func (s ReplySvc) transMsg(ctx context.Context, msg syncmsg.Message, ret interface{}) bool {
	var (
		wekf = MustFromWekf(ctx)
	)

	transMsg, ok := ret.(kf.ServiceStateTransOptions)
	if !ok {
		return false
	}
	transInfo, err := wekf.ServiceStateTrans(transMsg)
	if err != nil {
		log.Fatalf("trans msg err(%d): %s", transInfo.ErrCode, transInfo.ErrMsg)
		return false
	}

	return true
}

// 执行消息持久化
// TODO: 根据消息内容执行不同的持久化方式
func (s ReplySvc) saveMsg(ctx context.Context, data interface{}) {
	var (
		logger = MustFromLogger(ctx)
		talk   *Talk
		msg    *Message
	)

	switch m := data.(type) {
	case *ReplyMessage: // 返回的消息
		talk = &Talk{
			OpenKFID: m.OpenKFID,
			ToUserID: m.ToUser,
		}
		msg = &Message{
			MsgFrom:  "bot",
			Origin:   0,
			Msgid:    m.MsgID,
			Msgtype:  m.ReplyType,
			SendTime: time.Now(),
			Content:  m.ContentText,
		}
	case *syncmsg.Message: // 同步到的消息
		talk = &Talk{
			OpenKFID: m.OpenKFID,
			ToUserID: m.ExternalUserID,
		}
		msg = &Message{
			MsgFrom:  "sync",
			Origin:   m.Origin,
			Msgid:    m.MsgID,
			Msgtype:  m.MsgType,
			SendTime: time.Unix(int64(m.SendTime), 0),
			Content:  "",
			Raw:      string(m.GetOriginMessage()),
		}
		if m.MsgType == "text" {
			content, _ := m.GetTextMessage()
			msg.Content = content.Text.Content
		}
	default:
		log.Fatalf("save unknown data %v", data)
	}

	if talk != nil {
		bytes, _ := json.Marshal(talk)
		val := map[string]any{}
		json.Unmarshal(bytes, &val)
		logger.With("model", "talk").With(val).Info("new talk record")
	}

	if msg != nil {
		bytes, _ := json.Marshal(msg)
		val := map[string]any{}
		json.Unmarshal(bytes, &val)
		logger.With("model", "message").With(val).Info("new msg record")
	}
}
