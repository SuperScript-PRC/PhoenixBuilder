package Happy2018new

import (
	"encoding/json"
	"fmt"
	GameInterface "phoenixbuilder/game_control/game_interface"
	ResourcesControl "phoenixbuilder/game_control/resources_control"
	"phoenixbuilder/omega/defines"
	Happy2018new_depends "phoenixbuilder/omega/third_party/Happy2018new/depends"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pterm/pterm"
)

type MaintainPlayerPosition struct {
	*defines.BasicComponent
	apis      GameInterface.GameInterface
	table     Happy2018new_depends.PlayersPosInfo
	lock      sync.RWMutex
	signal    Happy2018new_depends.StateChange
	stoped    chan struct{}
	CheckTime int `json:"维护周期(单位:游戏刻)"`
}

func (o *MaintainPlayerPosition) ReceiveResponse() error {
	ticker := time.NewTicker(time.Second / 20 * time.Duration(o.CheckTime))
	defer ticker.Stop()
	// 初始化
	for {
		resp := o.apis.SendWSCommandWithResponse(
			"querytarget @a",
			ResourcesControl.CommandRequestOptions{
				TimeOut: time.Second * 5,
			},
		)
		if resp.Error != nil && resp.ErrorType == ResourcesControl.ErrCommandRequestTimeOut {
			<-ticker.C
			continue
		}
		result, err := o.apis.ParseTargetQueryingInfo(*resp.Respond)
		if err != nil {
			return fmt.Errorf("ReceiveResponse: %v", err)
		}
		// 请求并解析租赁符返回的玩家坐标及朝向信息
		for _, value := range result {
			player_uuid, err := uuid.Parse(value.UniqueId)
			if err != nil {
				return fmt.Errorf("ReceiveResponse: %v", err)
			}
			// 取得玩家的 UUID
			temp, _ := strconv.ParseFloat(
				strconv.FormatFloat(float64(value.Position[2]), 'f', 5, 32),
				32,
			)
			value.Position[2] = float32(temp) - 1.62001
			// 修正取得的 Y 轴坐标
			if player_kit := o.Frame.GetGameControl().GetPlayerKitByUUID(player_uuid); player_kit != nil {
				player_uq := player_kit.GetRelatedUQ()
				if player_uq == nil {
					continue
				}
				user_name := player_uq.Username
				// 取得玩家的名称
				o.lock.Lock()
				o.table[Happy2018new_depends.PlayerName(user_name)] = Happy2018new_depends.PosInfo{
					Dimension: value.Dimension,
					Position:  value.Position,
					YRot:      value.YRot,
				}
				o.lock.Unlock()
				o.signal.SendSignal()
				// 更新子结果
			}
			// 设置该玩家的坐标及朝向数据
		}
		// 更新玩家坐标和朝向信息
		select {
		case <-ticker.C:
		case <-o.stoped:
			return nil
		}
		// 等待下一次检测
	}
}
func (o *MaintainPlayerPosition) Init(
	settings *defines.ComponentConfig,
	storage defines.StorageAndLogProvider,
) {
	marshal, _ := json.Marshal(settings.Configs)
	err := json.Unmarshal(marshal, o)
	if err != nil {
		panic(err)
	}
	o.table = make(Happy2018new_depends.PlayersPosInfo)
	o.stoped = make(chan struct{}, 1)
}

func (o *MaintainPlayerPosition) Inject(frame defines.MainFrame) {
	o.Frame = frame
	o.apis = o.Frame.GetGameControl().GetInteraction()
	o.Frame.SetContext("global::storage::player_pos_table", o.table)
	o.Frame.SetContext("global::sync_mutex::player_pos_table", &o.lock)
	o.Frame.SetContext("global::regist_state_change::player_pos_table", o.signal.Register)
}

func (o *MaintainPlayerPosition) Activate() {
	go func() {
		for {
			err := o.ReceiveResponse()
			if err == nil {
				return
			}
			pterm.Error.Printf("MaintainPlayerPosition: %v\n", err)
		}
	}()
}

func (o *MaintainPlayerPosition) Stop() error {
	o.stoped <- struct{}{}
	return nil
}
