package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Knetic/govaluate"
	"github.com/chyroc/lark"
	_ "github.com/joho/godotenv/autoload"
)

var rollDefault = "1d100"

type Message struct {
	Text string `json:"text"`
}

func unmarshalMessageContent(content string) string {
	var m Message
	err := json.Unmarshal([]byte(content), &m)
	if err != nil {
		return ""
	}
	return m.Text
}

func clean(ctx context.Context, cli *lark.Lark, event *lark.EventV2IMMessageReceiveV1) (string, error) {
	c := strings.TrimSpace(unmarshalMessageContent(event.Message.Content))
	log.Println("new msg: " + c)

	if c == "@_user_1" && event.Message.ParentID != "" {
		resp, _, err := cli.Message.GetMessage(ctx, &lark.GetMessageReq{
			MessageID: event.Message.ParentID,
		})
		if err != nil {
			log.Println("error getting parent message: " + err.Error())
			return "", errors.New("error getting parent message")
		}
		if len(resp.Items) > 0 {
			m := resp.Items[0]
			msg := unmarshalMessageContent(m.Body.Content)
			msg = strings.TrimPrefix(msg, "r ")
			msg = strings.TrimSpace(msg)
			msg = strings.Split(msg, ":")[0]
			return msg, nil
		}
	}

	i := strings.HasPrefix(c, "r ")
	if !i && c != "r" {
		log.Println("ignoring message as it is not a roll msg: " + c)
		return "", nil
	}
	clean := rollDefault
	if c != "r" {
		clean = strings.TrimPrefix(c, "r ")
	}
	clean = strings.TrimSpace(clean)

	return clean, nil
}

func main() {
	dedup := NewDeduplicator()

	cli := lark.New(
		lark.WithAppCredential(os.Getenv("FEISHU_APP_ID"), os.Getenv("FEISHU_APP_SECRET")),
		lark.WithEventCallbackVerify(os.Getenv("FEISHU_ENCRYPT_KEY"), os.Getenv("FEISHU_VERIFICATION_TOKEN")),
	)

	// handle message callback
	cli.EventCallback.HandlerEventV2IMMessageReceiveV1(func(ctx context.Context, cli *lark.Lark, schema string, header *lark.EventHeaderV2, event *lark.EventV2IMMessageReceiveV1) (string, error) {
		if !dedup.GetSet(event.Message.MessageID) {
			return "", nil
		}

		c := strings.TrimSpace(unmarshalMessageContent(event.Message.Content))
		if strings.HasPrefix(c, "=") {
			e := strings.TrimPrefix(c, "=")
			expr, err := govaluate.NewEvaluableExpression(e)
			if err != nil {
				_, _, err = cli.Message.Reply(event.Message.MessageID).SendText(ctx, fmt.Sprintf("已识别 evaluation 命令，但在 evaluate 时出现了问题："+err.Error()))
				return "", err
			}
			res, err := expr.Eval(nil)
			if err != nil {
				_, _, err = cli.Message.Reply(event.Message.MessageID).SendText(ctx, fmt.Sprintf("已识别 evaluation 命令，但在 evaluate 时出现了问题："+err.Error()))
				return "", err
			}
			cli.Message.Reply(event.Message.MessageID).SendText(ctx, fmt.Sprintf("%v：%v", e, res))
			return "", nil
		}

		if strings.HasPrefix(c, "mode") {
			e := strings.TrimPrefix(c, "mode")
			e = strings.TrimSpace(e)
			if e == "coc" {
				rollDefault = "1d100"
				cli.Message.Reply(event.Message.MessageID).SendText(ctx, "已切换到 COC 模式 (default roll expression: 1d100)")
				return "", nil
			} else if e == "dnd" {
				rollDefault = "1d20"
				cli.Message.Reply(event.Message.MessageID).SendText(ctx, "已切换到 D&D 模式 (default roll expression: 1d20)")
				return "", nil
			} else {
				cli.Message.Reply(event.Message.MessageID).SendText(ctx, "模式不存在，请输入 mode coc 或 mode dnd")
				return "", nil
			}
		}

		desc, err := clean(ctx, cli, event)
		if err != nil {
			return "", err
		}

		log.Println("rolling: " + desc)

		start := time.Now()
		r, err := roll(desc)
		end := time.Since(start)
		log.Println("rolling cost", end.String())
		if err != nil && err != ErrorIgnore {
			log.Println("failed to roll dice: " + err.Error())
			_, _, err = cli.Message.Reply(event.Message.MessageID).SendText(ctx, fmt.Sprintf("已识别 roll 命令，但在 roll 时出现了问题："+err.Error()))
			return "", err
		}
		if err == ErrorIgnore {
			return "", nil
		}

		log.Println("roll result:", r)

		_, _, err = cli.Message.Reply(event.Message.MessageID).SendText(ctx, r)
		return "", err
	})

	cli.EventCallback.HandlerEventCard(func(ctx context.Context, cli *lark.Lark, event *lark.EventCardCallback) (string, error) {
		log.Println("new card: " + event.Action.Tag)
		return "", nil
	})

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		log.Println("new callback received")
		cli.EventCallback.ListenCallback(r.Context(), r.Body, w)
	})

	resp, _, err := cli.Bot.GetBotInfo(context.Background(), &lark.GetBotInfoReq{})
	if err != nil {
		log.Fatal(err)
	} else {
		log.Printf("bot info: %v", resp)
	}

	log.Println("server started.")
	log.Fatal(http.ListenAndServe(":9726", nil))
}
