package controller

import (
	"DataServer/m/models"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

func HandleConnection(conn net.Conn) {
	device := Device{ // 创建连接对象
		Did:  0,
		Role: 0,
		Con:  conn,
	}
	buffer := make([]byte, 10240) // 建立缓冲区
	fmt.Println("设备已连接：" + device.Con.RemoteAddr().String())
	for {
		num, err := conn.Read(buffer)
		if err != nil { // 断开连接
			break
		}
		fmt.Println("number:", num)
		HandleMessage(buffer, device)
	}
	HandleDisconnect(device)
}

func HandleMessage(msg []byte, device Device) {
	var err error
	var id identity
	var cmd command
	var data itemData
	fmt.Println("设备消息:", device.Con.RemoteAddr().String(), "> "+string(msg))
	switch device.Role {
	case ROLE_USER:
		err = json.Unmarshal(msg, &cmd)
		if err != nil {
			device.Con.Write([]byte("Error:" + err.Error()))
			fmt.Println(err.Error())
		}
		switch cmd.CMD {
		case "getData": // 查询100条记录
			items, err := models.FindItemsByDId(cmd.ToDid)
			if err != nil {
				device.Con.Write([]byte("Error:" + err.Error()))
				fmt.Println(err.Error())
				return
			}
			jsonData, err := json.Marshal(items)
			if err != nil {
				device.Con.Write([]byte("Error:" + err.Error()))
				fmt.Println(err.Error())
				return
			}
			device.Con.Write(jsonData)
		case "getNodes": // 获取节点
			json1, err := json.Marshal(NodeList)
			if err != nil {
				device.Con.Write([]byte("Error:" + err.Error()))
				fmt.Println(err.Error())
				return
			}
			device.Con.Write(json1)
		case "send": //下发指令到下位机
			node, ok := NodeList[cmd.ToDid]
			if !ok {
				device.Con.Write([]byte("Node Not Found!"))
				return
			}
			node.Con.Write([]byte(cmd.CMD))

		}
	case ROLE_NODE:
		err = json.Unmarshal(msg, data)
		if err != nil {
			device.Con.Write([]byte("Error:" + err.Error()))
			fmt.Println(err.Error())
		}
		item := models.Item{
			CreatedOn: int(time.Now().Unix()),
			Did:       device.Did,
			Light:     data.Light,
			Mq2:       data.Mq2,
			Mq135:     data.Mq135,
			Temp:      data.Temp,
			Wet:       data.Wet,
		}
		_, err = models.NewItem(item)
		if err != nil {
			fmt.Println("Error: " + err.Error())
			device.Con.Write([]byte("Error:" + err.Error()))
			return
		}
	default: // 获取身份
		err = json.Unmarshal(msg, &id)
		if err != nil {
			device.Con.Write([]byte("Error:" + err.Error()))
			fmt.Println(err.Error())
			return
		}
		device.Did = id.Did
		device.Role = ROLE_USER
		AppList[id.Did] = device
		if device.Role == ROLE_USER {
			AppList[device.Did] = device //添加至App在线列表
		} else if device.Role == ROLE_NODE {
			NodeList[device.Did] = device //添加至Node在线列表
		}
	}
}

func HandleDisconnect(device Device) {
	for k, v := range AppList {
		if v.Con == device.Con {
			fmt.Println("APP_", k, " at ", device.Con.RemoteAddr().String()+" 连接已断开.")
			return
		}
	}
	for k, v := range NodeList {
		if v.Con == device.Con {
			fmt.Println("Node_", k, " at ", device.Con.RemoteAddr().String()+" 连接已断开.")
			return
		}
	}
	fmt.Println("未知设备", " at ", device.Con.RemoteAddr().String()+" 连接已断开.")
	return
}