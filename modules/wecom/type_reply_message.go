package wecommod

import (
	"encoding/json"

	"github.com/silenceper/wechat/v2/work/kf"
	"github.com/silenceper/wechat/v2/work/kf/sendmsg"
)

type ContentMenu struct {
	HeadContent string
	List        []interface{}
	TailContent string
}

type ReplyMessage struct {
	ToUser   string
	OpenKFID string
	MsgID    string

	ReplyType    string
	ContentText  string
	ContentImage string
	ContentMenu  ContentMenu

	ActionTransState    int
	ActionTransServicer string

	msg interface{} // 组装后的消息体
}

func NewReplyMessage(toUser, openKFID, msgID string) *ReplyMessage {
	return &ReplyMessage{
		ToUser: toUser, OpenKFID: openKFID, MsgID: msgID,
	}
}

// Assume 用于组成微信客服接口请求体
func (m ReplyMessage) Assume() (interface{}, error) {
	return m.msg, nil
}

func (m *ReplyMessage) SetText(s string) {
	m.ReplyType = WecomMsgTypeText
	m.ContentText = s

	msg := sendmsg.Text{
		Message: m.getMessageHead(),
		MsgType: m.ReplyType,
	}
	msg.Text.Content = m.ContentText
	m.msg = msg
}

func (m *ReplyMessage) SetImage(s string) {
	m.ReplyType = WecomMsgTypeImg
	m.ContentImage = s

	msg := sendmsg.Image{
		Message: m.getMessageHead(),
		MsgType: m.ReplyType,
	}
	msg.Image.MediaID = m.ContentImage
	m.msg = msg
}

func (m *ReplyMessage) SetMenu(cm ContentMenu) {
	m.ReplyType = WecomMsgTypeMenu
	m.ContentMenu = cm

	msg := sendmsg.Menu{
		Message: m.getMessageHead(),
		MsgType: m.ReplyType,
	}
	msg.MsgMenu.HeadContent = m.ContentMenu.HeadContent
	msg.MsgMenu.List = m.ContentMenu.List
	msg.MsgMenu.TailContent = m.ContentMenu.TailContent
	m.msg = msg
}

func (m *ReplyMessage) SetActionTrans(state int, servicer string) {
	m.ReplyType = WecomMsgTypeActionTrans
	m.ActionTransState = state
	m.ActionTransServicer = servicer

	msg := kf.ServiceStateTransOptions{
		OpenKFID:       m.OpenKFID,
		ExternalUserID: m.ToUser,
		ServiceState:   m.ActionTransState,
		ServicerUserID: m.ActionTransServicer,
	}
	m.msg = msg
}

// Content 需要记录消息内容
func (m ReplyMessage) Content() string {
	var content string
	switch m.ReplyType {
	case WecomMsgTypeText:
		content = m.ContentText
	case WecomMsgTypeImg:
		content = m.ContentImage
	case WecomMsgTypeVideo:
		fallthrough
	case WecomMsgTypeVoice:
		fallthrough
	case WecomMsgTypeFile:
		fallthrough
	case WecomMsgTypeLocation:
		content = ""
	case WecomMsgTypeMenu:
		if bs, err := json.Marshal(m.ContentMenu); err == nil {
			content = string(bs)
		} else {
			content = ""
		}
	case WecomMsgTypeActionTrans:
		if bs, err := json.Marshal(m.ContentMenu); err == nil {
			content = string(bs)
		} else {
			content = ""
		}
	default:
		content = ""
	}

	return content
}

func (m ReplyMessage) getMessageHead() sendmsg.Message {
	return sendmsg.Message{
		ToUser:   m.ToUser,
		OpenKFID: m.OpenKFID,
		MsgID:    m.MsgID,
	}
}
