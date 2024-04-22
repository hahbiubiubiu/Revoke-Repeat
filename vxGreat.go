package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"github.com/eatmoreapple/openwechat"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"os"
	"strconv"
	"time"
)

var (
	self      *openwechat.Self
	testGroup openwechat.Groups
	bot       *openwechat.Bot
	aim_group string
)

type Message struct {
	ID        string
	Content   string
	Sender    string
	Timestamp time.Time // 添加这个字段来存储时间戳
}

type Image struct {
	ID        string
	Picture   []byte
	Sender    string
	Timestamp time.Time
}

type SQLiteMessageStore struct {
	db *sql.DB
}

func (store *SQLiteMessageStore) connect(sqlPath string) (err error) {
	store.db, err = sql.Open("sqlite3", sqlPath)
	return
}

func (store *SQLiteMessageStore) tableCreate() (err error) {
	_, err = store.db.Exec(`CREATE TABLE IF NOT EXISTS messages (
        id TEXT PRIMARY KEY,
        content TEXT,
        sender 	TEXT,
        timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    )`)
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = store.db.Exec(`CREATE TABLE IF NOT EXISTS images (
		id TEXT NOT NULL PRIMARY KEY,
		picture BLOB,
		sender 	TEXT,
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    )`)
	if err != nil {
		fmt.Println(err)
		return
	}
	return
}

func (store *SQLiteMessageStore) disconnect() {
	store.db.Close()
}

func (store *SQLiteMessageStore) insertMessage(id string, content string, sender string) (err error) {
	tx, err := store.db.Begin()
	if err != nil {
		return
	}
	stmt, err := tx.Prepare("INSERT INTO messages(id, content, sender) VALUES(?, ?, ?)")
	if err != nil {
		return
	}
	_, err = stmt.Exec(id, content, sender)
	if err != nil {
		return
	}
	tx.Commit()
	return
}

func (store *SQLiteMessageStore) retrieveMessage(id string) (message Message, err error) {
	row := store.db.QueryRow("SELECT * FROM messages WHERE id = ?", id)
	err = row.Scan(&message.ID, &message.Content, &message.Sender, &message.Timestamp)
	return
}

func (store *SQLiteMessageStore) deleteMessagesOlderThan(minutes int) (err error) {
	stmt, err := store.db.Prepare("DELETE FROM messages WHERE timestamp <= datetime('now', ? || ' minutes')")
	if err != nil {
		return
	}
	_, err = stmt.Exec(fmt.Sprintf("-%d", minutes))
	return
}

func (store *SQLiteMessageStore) insertImage(id string, picture []byte, sender string) (err error) {
	tx, err := store.db.Begin()
	if err != nil {
		return
	}
	stmt, err := tx.Prepare("INSERT INTO images(id, picture, sender) VALUES(?, ?, ?)")
	if err != nil {
		return
	}
	_, err = stmt.Exec(id, picture, sender)
	if err != nil {
		return
	}
	tx.Commit()
	return
}

func (store *SQLiteMessageStore) retrieveImage(id string) (image Image, err error) {
	row := store.db.QueryRow("SELECT * FROM images WHERE id = ?", id)
	err = row.Scan(&image.ID, &image.Picture, &image.Sender, &image.Timestamp)
	return
}

func (store *SQLiteMessageStore) deleteImagesOlderThan(minutes int) (err error) {
	stmt, err := store.db.Prepare("DELETE FROM images WHERE timestamp <= datetime('now', ? || ' minutes')")
	if err != nil {
		return
	}
	_, err = stmt.Exec(fmt.Sprintf("-%d", minutes))
	return
}

func main() {
	aim_group = "焦虑的浅水湾业主群"
	to_group := "真言"
	//aim_group = "123"
	// 文件路径
	filePath := "./messages.db"
	// 检查文件是否存在
	_, err := os.Stat(filePath)
	// 使用os.IsNotExist函数，如果返回True，则表示文件不存在
	if os.IsNotExist(err) {
		fmt.Println("File does not exist.")
	} else {
		// 文件存在，尝试删除
		err = os.Remove(filePath)
		// 若删除过程中出现错误，则输出错误，否则表示删除成功
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("File deleted successfully!")
		}
	}
	store := SQLiteMessageStore{}
	if err = store.connect(filePath); err != nil {
		fmt.Println(err)
		return
	}
	defer store.disconnect()
	if err = store.tableCreate(); err != nil {
		fmt.Println(err)
		return
	}

	// 桌面模式
	bot = openwechat.DefaultBot(openwechat.Desktop)

	// 注册消息处理函数
	bot.MessageHandler = func(msg *openwechat.Message) {
		if msg.IsSendByGroup() {
			var sender *openwechat.User
			sender, _ = msg.Sender()
			//fmt.Println("Sender", sender.NickName)
			//if sender.NickName == "testForRobot" {
			if sender.NickName == aim_group {
				if msg.IsText() {
					msgID := msg.MsgId
					content := msg.Content
					sender, _ = msg.SenderInGroup()
					if err = store.insertMessage(msgID, content, sender.NickName); err != nil {
						fmt.Println(err)
						return
					}
					fmt.Println("Message inserted successfully!")
					fmt.Println("Content", content)
					fmt.Println("Sender", sender.NickName)
				} else if msg.IsPicture() {
					resp, err := msg.GetPicture()
					if err != nil {
						fmt.Println(err)
						return
					}
					body, err := ioutil.ReadAll(resp.Body)
					if err != nil {
						fmt.Println(err)
						return
					}
					//err = ioutil.WriteFile("output.jpg", body, 0644)
					if err != nil {
						fmt.Println(err)
						return
					}
					sender, _ = msg.SenderInGroup()
					store.insertImage(msg.MsgId, body, sender.NickName)
					fmt.Println("Image inserted successfully!")
					fmt.Println("ID", msg.MsgId)
					//} else if msg.IsRenameGroup() {
					//re := regexp.MustCompile(".*“(.*?)”")
					//match := re.FindStringSubmatch(msg.Content)
					//if len(match) > 1 {
					//	fmt.Println("Group name changed to: " + match[1])
					//	aim_group = match[1]
					//	fmt.Println("Aim group changed to: " + aim_group)
					//} else {
					//	fmt.Println("No match")
					//}
				} else if msg.IsRecalled() {
					var revokedMsg *openwechat.RevokeMsg
					revokedMsg, _ = msg.RevokeMsg()
					msgID := strconv.FormatInt(revokedMsg.RevokeMsg.MsgId, 10)
					fmt.Println("撤回消息ID：" + msgID)
					var message Message
					if message, err = store.retrieveMessage(msgID); err == nil {
						fmt.Println(message.Sender + "撤回了：" + message.Content)
						store.deleteMessagesOlderThan(5)
						self.SendTextToGroup(testGroup.First(), message.Sender+"撤回了："+message.Content)
						return
					}
					var image Image
					if image, err = store.retrieveImage(msgID); err == nil {
						fmt.Println(image.Sender + "撤回了图片")
						store.deleteImagesOlderThan(5)
						pic := bytes.NewReader(image.Picture)
						self.SendTextToGroup(testGroup.First(), image.Sender+"撤回的图片：")
						self.SendImageToGroup(testGroup.First(), pic)
						return
					}
					fmt.Println("撤回消息不存在")
				}
			}
		}
	}

	// 执行热登录
	reloadStorage := openwechat.NewFileHotReloadStorage("storage.json")
	defer reloadStorage.Close()
	err = bot.HotLogin(reloadStorage, openwechat.NewRetryLoginOption())
	if err != nil {
		return
	}

	self, err = bot.GetCurrentUser()
	if err != nil {
		fmt.Println(err)
		return
	}
	group, err := self.Groups()
	if err != nil {
		fmt.Println(err)
		return
	}
	testGroup = group.SearchByNickName(1, to_group)

	err = bot.Block()
	if err != nil {
		return
	}
}
