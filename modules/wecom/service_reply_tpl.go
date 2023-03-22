package wecommod

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/silenceper/wechat/v2/work/kf/syncmsg"
)

const (
	WelcomeMsg = "您好，欢迎光临。"
	DefaultMsg = "回复“帮助”查看更多内容"
)

// 消息匹配方式
type MatchMethod int

const (
	MatchMethodFull     MatchMethod = iota // 匹配全文
	MatchMethodFullCi                      // 匹配全文, 不区分大小写
	MatchMethodKeyword                     // 匹配关键字
	MatchMethodPrefix                      // 匹配前缀
	MatchMethodRegexp                      // 正则匹配
	MatchMethodImg                         // 匹配图片
	MatchMethodVideo                       // 匹配视频
	MatchMethodVoice                       // 匹配音频
	MatchMethodFile                        // 匹配文件
	MatchMethodLocation                    // 匹配定位
)

// ReplyType 返回消息类型
type ReplyType int

const (
	ReplyTypeText        ReplyType = iota       // 返回文本消息
	ReplyTypeImage                              // 返回图片消息
	ReplyTypeMenu                               // 返回菜单消息
	ReplyTypeActionTrans ReplyType = iota + 100 // 分配客服会话
)

// 填充式消息
type ReplyCallback func(mr *ReplyMessage) error

func NewTpls() []*ReplyTpl {
	// TplReplys 根据用户消息内容，返回对话内容
	return []*ReplyTpl{
		{"text", MatchMethodFull, "default", ReplyTypeText, WelcomeMsg},
		{"text", MatchMethodFull, "欢迎语", ReplyTypeText, WelcomeMsg},
		{"text", MatchMethodFullCi, "ping", ReplyTypeText, "Pong!"},
		{"text", MatchMethodFullCi, "pong", ReplyTypeText, "Ping!"},
		{"text", MatchMethodFull, "帮助", ReplyTypeMenu, ReplyCallback(func(mr *ReplyMessage) error {
			cm := ContentMenu{
				HeadContent: WelcomeMsg,
				List: []interface{}{
					map[string]interface{}{
						"type":  "click",
						"click": map[string]string{"id": "welcome", "content": "欢迎语"},
					},
				},
			}
			mr.SetMenu(cm)
			return nil
		})},
		{"text", MatchMethodFull, "人工客服", ReplyTypeActionTrans, ""},
	}
}

type ReplyTpl struct {
	MatchType   string      // 消息配置类型, 可选值 text, image, video, voice, file, location
	MatchMethod MatchMethod // 消息配置方式, 可选值: full, keyword, prefix
	MatchValue  string      // 消息配置内容

	ReplyType ReplyType   // 返回消息类型, 可选值: action, text
	Message   interface{} // 消息内容 or interface
}

// Gen 组装消息
func (rt ReplyTpl) Gen(ctx context.Context, toUser, openKFID string) *ReplyMessage {
	var (
		uuid, err = uuid.NewV6()
		redis     = MustFromCache(ctx)
		wekf      = MustFromWekf(ctx)
	)
	if err != nil {
		panic(uuid)
	}
	ret := NewReplyMessage(toUser, openKFID, strings.ReplaceAll(uuid.String(), "-", ""))

	switch rt.ReplyType {
	case ReplyTypeText: // 文本消息
		ret.SetText(rt.Message.(string))
	case ReplyTypeImage: // 图片消息
		msg, ok := rt.Message.(string)
		if !ok {
			return nil
		}
		if strings.HasPrefix(msg, InviteImagePrefix) {
			if mediaID, err := redis.Get(context.TODO(), msg).Result(); err == nil {
				log.Println("debug:", mediaID)
				ret.SetImage(mediaID)
			} else {
				ret.SetText(fmt.Sprintf(
					"can no find image: %s", msg[len(InviteImagePrefix):],
				))
			}
		} else {
			ret.SetImage(msg)
		}
	case ReplyTypeMenu: // 菜单消息
		callback, ok := rt.Message.(ReplyCallback)
		if ok {
			callback(ret)
		}
	case ReplyTypeActionTrans:
		// 查找客服账号列表
		accountList, err := wekf.AccountList()
		if err != nil || len(accountList.AccountList) == 0 {
			return nil
		}
		account := accountList.AccountList[0]

		// 接待人员列表
		receptionisList, err := wekf.ReceptionistList(account.OpenKFID)
		if err != nil || len(receptionisList.ReceptionistList) == 0 {
			return nil
		}
		receptionis := receptionisList.ReceptionistList[0]

		ret.SetActionTrans(3, receptionis.UserID)
	}

	return ret
}

// Match 根据不同的消息类型选择不同的匹配方式
func (rt ReplyTpl) Match(ctx context.Context, msg syncmsg.Message) bool {
	switch msg.MsgType {
	case WecomMsgTypeText:
		if info, err := msg.GetTextMessage(); err == nil {
			return rt.matchText(info.Text.Content)
		}
		return false
	case WecomMsgTypeImg: // 图片
		return true
	case WecomMsgTypeVideo: // 视频
		return true
	case WecomMsgTypeVoice: // 语音
		return true
	case WecomMsgTypeFile: // 文件
		return true
	case WecomMsgTypeLocation: // 位置
		return true
	default: // 默认回复
		log.Fatalf("unknown user msg type: %s", msg.MsgType)
		return false
	}
}

func (rt ReplyTpl) matchText(s string) bool {
	switch rt.MatchMethod {
	case MatchMethodFull:
		return rt.MatchValue == s
	case MatchMethodFullCi:
		return strings.ToLower(rt.MatchValue) == strings.ToLower(s)
	case MatchMethodKeyword:
		return strings.Contains(s, rt.MatchValue)
	case MatchMethodPrefix:
		return strings.HasPrefix(s, rt.MatchValue)
	default:
		return false
	}
}
