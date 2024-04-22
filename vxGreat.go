package main

import (
	"database/sql"
	"fmt"
	"github.com/eatmoreapple/openwechat"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"strconv"
	"time"
)

var (
	self      *openwechat.Self
	testGroup openwechat.Groups
	bot       *openwechat.Bot
	flag      bool
)

type Message struct {
	ID        string
	Content   string
	Sender    string
	Timestamp time.Time // 添加这个字段来存储时间戳
}

type SQLiteMessageStore struct {
	db *sql.DB
}

func (store *SQLiteMessageStore) connect(sqlPath string) (err error) {
	store.db, err = sql.Open("sqlite3", sqlPath)
	return
}

func (store *SQLiteMessageStore) tableCreate() (err error) {
	createTableSQL := `CREATE TABLE IF NOT EXISTS messages (
        id TEXT PRIMARY KEY,
        content TEXT,
        sender 	TEXT,
        timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    )`
	_, err = store.db.Exec(createTableSQL)
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

func (store *SQLiteMessageStore) deleteMessage(id string) (err error) {
	tx, err := store.db.Begin()
	if err != nil {
		return
	}
	stmt, err := tx.Prepare("DELETE FROM messages WHERE id = ?")
	if err != nil {
		return
	}
	_, err = stmt.Exec(id)
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

func (store *SQLiteMessageStore) retrieveAllMessages() (messages []Message, err error) {
	rows, err := store.db.Query("SELECT * FROM messages")
	if err != nil {
		return
	}
	for rows.Next() {
		var msg Message
		if err = rows.Scan(&msg.ID, &msg.Content, &msg.Sender, &msg.Timestamp); err != nil {
			return
		}
		messages = append(messages, msg)
	}
	rows.Close()
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

func main() {
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

	flag = false
	// 桌面模式
	bot = openwechat.DefaultBot(openwechat.Desktop)
	// 注册消息处理函数
	bot.MessageHandler = func(msg *openwechat.Message) {
		if msg.IsSendByGroup() {
			if msg.IsText() {
				var sender *openwechat.User
				sender, _ = msg.Sender()
				//if sender.NickName == "testForRobot" {
				if sender.NickName == "焦虑的浅水湾业主群" {
					msgID := msg.MsgId
					content := msg.Content
					sender, _ = msg.SenderInGroup()
					if err = store.insertMessage(msgID, content, sender.NickName); err != nil {
						fmt.Println(err)
						return
					}
					fmt.Println("Message inserted successfully!")
					fmt.Println("ID", msgID)
					fmt.Println("Content", content)
					fmt.Println("Sender", sender.NickName)
				} else if sender.NickName == "testForRobot" && !flag {
					self, _ = bot.GetCurrentUser()
					if err != nil {
						fmt.Println(err)
						return
					}
					if self == nil {
						fmt.Println("self is nil")
						return
					}
					group, err := self.Groups()
					if err != nil {
						fmt.Println(err)
						return
					}
					testGroup = group.SearchByNickName(1, "testForRobot")
					flag = true
				}
			} else if msg.IsRecalled() {
				var revokedMsg *openwechat.RevokeMsg
				var message Message
				revokedMsg, _ = msg.RevokeMsg()
				msgID := strconv.FormatInt(revokedMsg.RevokeMsg.MsgId, 10)
				if message, err = store.retrieveMessage(msgID); err != nil {
					fmt.Println(err)
					return
				}
				fmt.Println(message.Sender + "撤回了：" + message.Content)
				store.deleteMessagesOlderThan(5)
				if flag {
					self.SendTextToGroup(testGroup.First(), message.Sender+"撤回了："+message.Content)
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
	err = bot.Block()
	if err != nil {
		return
	}
}
