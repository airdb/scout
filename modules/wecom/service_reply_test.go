package wecommod

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/silenceper/wechat/v2/work/kf/syncmsg"
)

func TestReply_ProcMsg(t *testing.T) {
	type args struct {
		ctx context.Context
		msg syncmsg.Message
	}

	tests := []struct {
		name string
		s    *ReplySvc
		args args
	}{
		// {"text msg", NewReply(), args{context.Background(), syncmsg.Message{
		// 	Origin:     3,
		// 	MsgType:    "text",
		// 	OriginData: generateTextData("[寻人]"),
		// }}},
		{"menu msg", NewReplySvc(
			replySvcDeps{}),
			args{context.Background(), syncmsg.Message{
				Origin:     3,
				MsgType:    "text",
				OriginData: generateTextData("志愿者"),
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := ReplySvc{}
			s.ProcMsg(tt.args.ctx, tt.args.msg)
		})
	}
}

func generateTextData(s string) []byte {
	baseMsg := syncmsg.BaseMessage{
		Origin: 3,
	}
	msg := syncmsg.Text{
		BaseMessage: baseMsg,
	}
	msg.MsgType = "text"
	msg.Text.Content = s

	b, _ := json.Marshal(msg)

	return b
}
