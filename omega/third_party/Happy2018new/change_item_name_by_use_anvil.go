package Happy2018new

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"phoenixbuilder/fastbuilder/mcstructure"
	GameInterface "phoenixbuilder/game_control/game_interface"
	ResourcesControl "phoenixbuilder/game_control/resources_control"
	"phoenixbuilder/minecraft/nbt"
	"phoenixbuilder/minecraft/protocol"
	"phoenixbuilder/minecraft/protocol/packet"
	"phoenixbuilder/omega/defines"
	"strconv"
	"strings"
	"sync"

	"github.com/pterm/pterm"
)

type ChangeItemNameByUseAnvil struct {
	*defines.BasicComponent
	apis      GameInterface.GameInterface
	lockDown  sync.Mutex
	Triggers  []string `json:"菜单触发词"`
	Usage     string   `json:"菜单项描述"`
	FilePath  string   `json:"从何处提取物品的新名称(填写路径)"`
	Operators []string `json:"授权使用者"`
}

func (o *ChangeItemNameByUseAnvil) Init(settings *defines.ComponentConfig, storage defines.StorageAndLogProvider) {
	marshal, _ := json.Marshal(settings.Configs)
	if err := json.Unmarshal(marshal, o); err != nil {
		panic(err)
	}
}

func (o *ChangeItemNameByUseAnvil) Inject(frame defines.MainFrame) {
	o.Frame = frame
	o.apis = o.Frame.GetGameControl().GetInteraction()
	o.lockDown = sync.Mutex{}
	o.Frame.GetGameListener().SetGameMenuEntry(&defines.GameMenuEntry{
		MenuEntry: defines.MenuEntry{
			Triggers:     o.Triggers,
			ArgumentHint: "[快捷栏槽位: int] [x: int] [y: int] [z: int]",
			FinalTrigger: false,
			Usage:        o.Usage,
		},
		OptionalOnTriggerFn: o.ChangeItemNameRunner,
	})
}

func (o *ChangeItemNameByUseAnvil) ChangeItemNameRunner(chat *defines.GameChat) bool {
	go func() {
		{
			flag := false
			for _, name := range o.Operators {
				if name == chat.Name {
					flag = true
					break
				}
			}
			if !flag {
				o.Frame.GetGameControl().SayTo(chat.Name, "§c你没有权限使用这个功能")
				return
			}
		}
		{
			if !o.lockDown.TryLock() {
				o.Frame.GetGameControl().SayTo(chat.Name, "§c请求过于频繁§f，§c请稍后再试")
				return
			}
			defer o.lockDown.Unlock()
		}
		o.ChangeItemName(chat)
	}()
	return true
}

func (o *ChangeItemNameByUseAnvil) ChangeItemName(chat *defines.GameChat) {
	var mode uint8 = 0
	var targetSlot uint8 = 0
	var readPos []int32 = []int32{}
	var itemName string = ""
	var dropResp bool
	o.apis.ClientInfo.DisplayName = o.Frame.GetUQHolder().GetBotName()
	// 初始化
	if len(chat.Msg) > 0 {
		got, err := strconv.ParseUint(chat.Msg[0], 10, 32)
		if err != nil {
			o.Frame.GetGameControl().SayTo(chat.Name, "§c无法解析槽位数据§f，§c请确认你提供了正确的整数\n详细日志已发送到控制台")
			pterm.Error.Printf("修改物品名称: %v\n", err)
			return
		}
		targetSlot = uint8(got)
		if targetSlot > 8 {
			o.Frame.GetGameControl().SayTo(chat.Name, fmt.Sprintf("§c你提供的槽位参数 §b%v §c已大于 §b8", targetSlot))
			return
		}
	} else {
		o.Frame.GetGameControl().SayTo(chat.Name, "§e你没有提供槽位参数§f，§e现在默认重定向为 §b0")
	}
	// 确定槽位位置
	if len(chat.Msg) > 1 {
		mode = 1
		if len(chat.Msg) < 4 {
			o.Frame.GetGameControl().SayTo(chat.Name, "§c提供的参数不足§f，§c当前缺少一个或多个坐标")
			return
		}
		for i := 0; i < 3; i++ {
			got, err := strconv.ParseInt(chat.Msg[i+1], 10, 32)
			if err != nil {
				o.Frame.GetGameControl().SayTo(chat.Name, fmt.Sprintf("§c无法解析坐标数据§f，§c错误发生在位置 §b%v §f，§c请确认你提供了正确的坐标数据\n详细日志已发送到控制台", i))
				pterm.Error.Printf("修改物品名称: %v\n", err)
				return
			}
			readPos = append(readPos, int32(got))
		}
	}
	// 如果用户希望在游戏内完成名称编辑操作
	if mode == 0 {
		datas, err := o.Frame.GetFileData(o.FilePath)
		if err != nil {
			o.Frame.GetGameControl().SayTo(chat.Name, fmt.Sprintf("§c无法打开 §bomega_storage/data/%v §c处的文件\n详细日志已发送到控制台", o.FilePath))
			pterm.Error.Printf("修改物品名称: %v\n", err)
			return
		}
		if len(datas) <= 0 {
			o.Frame.GetGameControl().SayTo(chat.Name, fmt.Sprintf("§bomega_storage/data/%v §c处的文件没有填写物品名称§f，§c可能这个文件是个空文件§f，§c也可能是文件本身不存在", o.FilePath))
			return
		}
		itemName = strings.ReplaceAll(string(datas), "\r", "")
	} else if mode == 1 {
		holder := o.apis.Resources.Structure.Occupy()
		resp, err := o.apis.SendStructureRequestWithResponse(
			&packet.StructureTemplateDataRequest{
				StructureName: "Omega:ChangeItemNameByUseAnvil",
				Position:      protocol.BlockPos{readPos[0], readPos[1], readPos[2]},
				Settings: protocol.StructureSettings{
					PaletteName:               "default",
					IgnoreEntities:            true,
					IgnoreBlocks:              false,
					Size:                      protocol.BlockPos{1, 1, 1},
					Offset:                    protocol.BlockPos{0, 0, 0},
					LastEditingPlayerUniqueID: o.apis.ClientInfo.EntityUniqueID,
					Rotation:                  0,
					Mirror:                    0,
					Integrity:                 100,
					Seed:                      0,
					AllowNonTickingChunks:     false,
				},
				RequestType: packet.StructureTemplateRequestExportFromSave,
			},
		)
		o.apis.Resources.Structure.Release(holder)
		if err != nil {
			o.Frame.GetGameControl().SayTo(chat.Name, "§c未能请求命令方块数据\n详细日志已发送到控制台")
			pterm.Error.Printf("修改物品名称: %v\n", err)
			return
		}
		// 请求结构数据
		_, reversedMap, _ := mcstructure.SplitArea(
			mcstructure.BlockPos{readPos[0], readPos[1], readPos[2]},
			mcstructure.BlockPos{readPos[0], readPos[1], readPos[2]},
			64,
			64,
			true,
		)
		got, err := mcstructure.GetMCStructureData(
			mcstructure.Area{
				BeginX: readPos[0],
				BeginY: readPos[1],
				BeginZ: readPos[2],
				SizeX:  1,
				SizeY:  1,
				SizeZ:  1,
			},
			resp.StructureTemplate,
		)
		if err != nil {
			o.Frame.GetGameControl().SayTo(chat.Name, "§c未能请求命令方块数据\n详细日志已发送到控制台")
			pterm.Error.Printf("修改物品名称: %v\n", err)
			return
		}
		allAreas := []mcstructure.Mcstructure{got}
		processedData, err := mcstructure.DumpBlocks(
			allAreas,
			reversedMap,
			mcstructure.Area{
				BeginX: int32(readPos[0]),
				BeginY: int32(readPos[1]),
				BeginZ: int32(readPos[2]),
				SizeX:  1,
				SizeY:  1,
				SizeZ:  1,
			},
		)
		if err != nil {
			o.Frame.GetGameControl().SayTo(chat.Name, "§c未能请求命令方块数据\n详细日志已发送到控制台")
			pterm.Error.Printf("修改物品名称: resp = %#v\n", resp)
			return
		}
		if len(processedData) <= 0 {
			o.Frame.GetGameControl().SayTo(chat.Name, "§c未能请求命令方块数据\n详细日志已发送到控制台")
			pterm.Error.Printf("修改物品名称: resp = %#v\n", resp)
			return
		}
		// 从结构中提取目标方块的原始 NBT 数据
		newBuffer := bytes.NewBuffer(processedData[0].NBTData)
		var commandBlockNBT map[string]interface{}
		err = nbt.NewDecoderWithEncoding(newBuffer, nbt.LittleEndian).Decode(&commandBlockNBT)
		if err != nil {
			o.Frame.GetGameControl().SayTo(chat.Name, "§c未能请求命令方块数据\n详细日志已发送到控制台")
			pterm.Error.Printf("修改物品名称: processedData[0].NBTData = %#v\n", processedData[0].NBTData)
			return
		}
		// 解码目标方块的 NBT 数据
		_, ok := commandBlockNBT["Command"]
		if !ok {
			o.Frame.GetGameControl().SayTo(chat.Name, "§c目标方块不是命令方块")
			return
		}
		// 确定得到的方块是命令方块
		itemName, _ = commandBlockNBT["Command"].(string)
		err = json.Unmarshal([]byte(fmt.Sprintf(`"%s"`, itemName)), &itemName)
		if err != nil {
			o.Frame.GetGameControl().SayTo(chat.Name, fmt.Sprintf("§c命令方块包含的字符串无效\n错误信息为 §f%v", err))
			return
		}
		// 取得物品新名称，并进行反转义
	}
	// 获取物品的新名称
	itemDatas, err := o.apis.Resources.Inventory.GetItemStackInfo(0, targetSlot)
	if err != nil {
		o.Frame.GetGameControl().SayTo(chat.Name, fmt.Sprintf("§c在读取快捷栏 §b%v §c时发送了错误\n详细日志已发送到控制台", targetSlot))
		pterm.Error.Printf("修改物品名称: %v\n", err)
		return
	}
	if itemDatas.Stack.ItemType.NetworkID == 0 {
		o.Frame.GetGameControl().SayTo(chat.Name, fmt.Sprintf("§c请确保机器人在快捷栏 §b%v §c有一个物品\n详细日志已发送到控制台", targetSlot))
		pterm.Warning.Printf("修改物品名称: itemDatas = %#v\n", itemDatas)
		return
	}
	// 确定被改名物品存在
	cmdResp := o.apis.SendWSCommandWithResponse(
		fmt.Sprintf("querytarget %#v", chat.Name),
		ResourcesControl.CommandRequestOptions{
			TimeOut: ResourcesControl.CommandRequestNoDeadLine,
		},
	)
	if cmdResp.Error != nil {
		panic(pterm.Error.Sprintf("修改物品名称: %v", cmdResp.Error))
	}
	parseAns, err := o.apis.ParseTargetQueryingInfo(*cmdResp.Respond)
	if err != nil {
		panic(pterm.Error.Sprintf("修改物品名称: %v", err))
	}
	if len(parseAns) <= 0 {
		o.Frame.GetGameControl().SayTo(chat.Name, "§c机器人可能没有 §bOP §c权限")
		return
	}
	pos := [3]int32{
		int32(math.Floor(float64(parseAns[0].Position[0]))),
		int32(math.Floor(float64(parseAns[0].Position[1] - 1.62001001834869))),
		int32(math.Floor(float64(parseAns[0].Position[2]))),
	}
	// 取得请求者当前的坐标
	err = o.apis.SendSettingsCommand(
		fmt.Sprintf(`execute as @a[name=%#v] at @s run tp @a[name=%#v] %d %d %d`, chat.Name, o.apis.ClientInfo.DisplayName, pos[0], pos[1], pos[2]),
		false,
	)
	if err != nil {
		panic(pterm.Error.Sprintf("修改物品名称: %v", err))
	}
	cmdResp = o.apis.SendWSCommandWithResponse(
		"list",
		ResourcesControl.CommandRequestOptions{
			TimeOut: ResourcesControl.CommandRequestDefaultDeadLine,
		},
	)
	if cmdResp.Error != nil {
		panic(pterm.Error.Sprintf("修改物品名称: %v", cmdResp.Error))
	}
	// 将机器人传送到玩家处
	resp, err := o.apis.RenameItemByAnvil(
		pos,
		`["direction"=0,"damage"="undamaged"]`,
		targetSlot,
		[]GameInterface.ItemRenamingRequest{
			{
				Slot: targetSlot,
				Name: itemName,
			},
		},
	)
	if err != nil {
		o.Frame.GetGameControl().SayTo(chat.Name, "§c物品名称修改失败\n详细日志已发送到控制台")
		pterm.Error.Printf("修改物品名称: %v\n", err)
		return
	}
	if !resp[0].Successful {
		if resp[0].Destination == nil {
			o.Frame.GetGameControl().SayTo(chat.Name, "§c物品名称修改失败§f，§c请检查机器人的背包是否还有空位§f！\n§c原物品已丢出")
			return
		} else {
			o.Frame.GetGameControl().SayTo(chat.Name, "§c物品名称修改失败§f，§c请检查新的名称是否与原始名称相同或该物品是否可被移动")
			realSlot := resp[0].Destination.Slot
			if realSlot != targetSlot {
				o.Frame.GetGameControl().SayTo(chat.Name, fmt.Sprintf("§e检测到原物品栏已被占用§f，\n§e现在已将该物品还原到物品栏 §b%v", realSlot))
			}
			return
		}
	}
	// 修改物品名称
	newItemDatas, err := o.apis.Resources.Inventory.GetItemStackInfo(0, resp[0].Destination.Slot)
	if err != nil {
		o.Frame.GetGameControl().SayTo(chat.Name, fmt.Sprintf("§c在读取快捷栏 §b%v §c时发送了错误\n详细日志已发送到控制台", resp[0].Destination.Slot))
		pterm.Error.Printf("修改物品名称: %v\n", err)
		return
	}
	// 读取新物品的数据
	err = o.apis.SendSettingsCommand(
		fmt.Sprintf(`execute as @a[name=%#v] at @s run tp @a[name=%#v] %d %d %d`, chat.Name, o.apis.ClientInfo.DisplayName, pos[0], pos[1], pos[2]),
		false,
	)
	if err != nil {
		panic(pterm.Error.Sprintf("修改物品名称: %v", err))
	}
	cmdResp = o.apis.SendWSCommandWithResponse(
		"list",
		ResourcesControl.CommandRequestOptions{
			TimeOut: ResourcesControl.CommandRequestDefaultDeadLine,
		},
	)
	if cmdResp.Error != nil {
		panic(pterm.Error.Sprintf("修改物品名称: %v", cmdResp.Error))
	}
	// 再次将机器人传送到玩家处
	if resp[0].Destination.Slot < 9 {
		dropResp, err = o.apis.DropItemAll(
			protocol.StackRequestSlotInfo{
				ContainerID:    GameInterface.ContainerIDHotBar,
				Slot:           resp[0].Destination.Slot,
				StackNetworkID: newItemDatas.StackNetworkID,
			},
			0,
		)
		if err != nil {
			o.Frame.GetGameControl().SayTo(chat.Name, "§c尝试丢出新物品时失败\n详细日志已发送到控制台")
			pterm.Error.Printf("修改物品名称: %v\n", err)
			return
		}
	} else {
		holder := o.apis.Resources.Container.Occupy()
		o.apis.OpenInventory()
		dropResp, err = o.apis.DropItemAll(
			protocol.StackRequestSlotInfo{
				ContainerID:    0xc,
				Slot:           resp[0].Destination.Slot,
				StackNetworkID: newItemDatas.StackNetworkID,
			},
			0,
		)
		if err != nil {
			o.Frame.GetGameControl().SayTo(chat.Name, "§c尝试丢出新物品时失败\n详细日志已发送到控制台")
			pterm.Error.Printf("修改物品名称: %v\n", err)
			return
		}
		o.apis.CloseContainer()
		o.apis.Resources.Container.Release(holder)
	}
	if !dropResp {
		o.Frame.GetGameControl().SayTo(chat.Name, "§c尝试丢出新物品时失败\n详细日志已发送到控制台")
		pterm.Error.Printf("修改物品名称: dropResp = %#v\n", dropResp)
		return
	}
	// 丢出新物品
	o.Frame.GetGameControl().SayTo(chat.Name, "§a已成功修改物品名称")
	// 返回值
}
