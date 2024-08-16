package core

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"phoenixbuilder/fastbuilder/args"
	"phoenixbuilder/fastbuilder/environment"
	"phoenixbuilder/fastbuilder/external"
	"phoenixbuilder/fastbuilder/function"
	I18n "phoenixbuilder/fastbuilder/i18n"
	"phoenixbuilder/fastbuilder/move"
	fbauth "phoenixbuilder/fastbuilder/pv4"
	"phoenixbuilder/fastbuilder/py_rpc"
	cts "phoenixbuilder/fastbuilder/py_rpc/mod_event/client_to_server"
	cts_mc "phoenixbuilder/fastbuilder/py_rpc/mod_event/client_to_server/minecraft"
	cts_mc_p "phoenixbuilder/fastbuilder/py_rpc/mod_event/client_to_server/minecraft/preset"
	cts_mc_v "phoenixbuilder/fastbuilder/py_rpc/mod_event/client_to_server/minecraft/vip_event_system"
	mei "phoenixbuilder/fastbuilder/py_rpc/mod_event/interface"
	"phoenixbuilder/fastbuilder/readline"
	"phoenixbuilder/fastbuilder/signalhandler"
	fbtask "phoenixbuilder/fastbuilder/task"
	"phoenixbuilder/fastbuilder/types"
	"phoenixbuilder/fastbuilder/uqHolder"
	GameInterface "phoenixbuilder/game_control/game_interface"
	ResourcesControl "phoenixbuilder/game_control/resources_control"
	"phoenixbuilder/minecraft"
	"phoenixbuilder/minecraft/protocol"
	"phoenixbuilder/minecraft/protocol/packet"
	"phoenixbuilder/mirror/io/assembler"
	"phoenixbuilder/mirror/io/global"
	"phoenixbuilder/mirror/io/lru"
	"phoenixbuilder/omega/cli/embed"
	"runtime"
	"strings"
	"time"

	"github.com/pterm/pterm"
)

type CachedPacket struct {
	Packet packet.Packet
	Data   []byte
}

func EnterReadlineThread(env *environment.PBEnvironment, breaker chan struct{}) {
	if args.NoReadline {
		return
	}
	defer Fatal()
	gameInterface := env.GameInterface
	functionHolder := env.FunctionHolder.(*function.FunctionHolder)
	for {
		if breaker != nil {
			select {
			case <-breaker:
				return
			default:
			}
		}
		cmd := readline.Readline(env)
		if len(cmd) == 0 {
			continue
		}
		if env.OmegaAdaptorHolder != nil && !strings.Contains(cmd, "exit") {
			env.OmegaAdaptorHolder.(*embed.EmbeddedAdaptor).FeedBackendCommand(cmd)
			continue
		}
		switch cmd[0] {
		case '*':
			gameInterface.SendSettingsCommand(cmd[1:], false)
		case '.':
			resp := gameInterface.SendCommandWithResponse(
				cmd[1:],
				ResourcesControl.CommandRequestOptions{
					TimeOut: ResourcesControl.CommandRequestDefaultDeadLine,
				},
			)
			if resp.Error != nil {
				env.GameInterface.Output(
					pterm.Error.Sprintf(
						"Failed to get respond of \"%v\", and the following is the error log.",
						cmd[1:],
					),
				)
				env.GameInterface.Output(pterm.Error.Sprintf("%v", resp.Error.Error()))
			} else {
				env.GameInterface.Output(fmt.Sprintf("%+v", *resp.Respond))
			}
		case '!':
			resp := gameInterface.SendWSCommandWithResponse(
				cmd[1:],
				ResourcesControl.CommandRequestOptions{
					TimeOut: ResourcesControl.CommandRequestDefaultDeadLine,
				},
			)
			if resp.Error != nil {
				env.GameInterface.Output(
					pterm.Error.Sprintf(
						"Failed to get respond of \"%v\", and the following is the error log.",
						cmd[1:],
					),
				)
				env.GameInterface.Output(pterm.Error.Sprintf("%v", resp.Error.Error()))
			} else {
				env.GameInterface.Output(fmt.Sprintf("%+v", *resp.Respond))
			}
		case '~':
			resp := gameInterface.SendAICommandWithResponse(
				cmd[1:],
				ResourcesControl.CommandRequestOptions{
					TimeOut: ResourcesControl.CommandRequestNoDeadLine,
				},
			)
			if resp.Error != nil {
				env.GameInterface.Output(
					pterm.Error.Sprintf(
						"Failed to get respond of \"%v\", and the following is the error log.",
						cmd[1:],
					),
				)
				env.GameInterface.Output(pterm.Error.Sprintf("%v", resp.Error.Error()))
			} else {
				output := fmt.Sprintf(
					"PyRpc Result\n%+v\n\nPyRpc Output\n",
					resp.AICommand.Result,
				)
				if resp.AICommand.Output == nil {
					output = fmt.Sprintf("%s%+v\n", output, nil)
				} else {
					for _, value := range resp.AICommand.Output {
						output = fmt.Sprintf("%s%#v\n", output, value)
					}
				}
				output = fmt.Sprintf(
					"%s\nPyRpc PreCheckError\n%+v\n\nStandard Response\n%+v",
					output,
					resp.AICommand.PreCheckError,
					resp.Respond,
				)
				env.GameInterface.Output(pterm.Info.Sprint(output))
			}
		}
		functionHolder.Process(cmd)
	}
}

func EnterWorkerThread(env *environment.PBEnvironment, breaker chan struct{}) {
	conn := env.Connection.(*minecraft.Conn)
	functionHolder := env.FunctionHolder.(*function.FunctionHolder)

	chunkAssembler := assembler.NewAssembler(assembler.REQUEST_AGGRESSIVE, time.Second*5)
	// max 100 chunk requests per second
	chunkAssembler.CreateRequestScheduler(func(pk *packet.SubChunkRequest) {
		conn.WritePacket(pk)
	})
	// currentChunkConstructor := &world_provider.ChunkConstructor{}

	/*
			{
				go func() {
					time.Sleep(time.Second * 5)
					fmt.Println("STARTED")
					api := env.GameInterface.(*GameInterface.GameInterface)
					api.SendSettingsCommand("gamemode 1", true)
					api.SendSettingsCommand("replaceitem entity @s slot.hotbar 4 spawn_egg 12 51", true)
					api.SendWSCommandWithResponse("tp 72 117 -75")
					resp, _ := api.RenameItemByAnvil(
						[3]int32{72, 117, -75},
						`["direction": 0, "damage": "undamaged"]`,
						0,
						[]GameInterface.ItemRenamingRequest{
							{
								Slot: 4,
								Name: `§r§f§l◀【§6幻§e想§b乡 §f● §d贡献者§f】▶
		§r§a§lMC丶血小板 §f| §aBaby_2016 §f| §a时祗旧梦丶薰 §f| §a叶澄鑫 §f| §a晓月黯殇
		§r§a§lAS青璃 §f| §a饿兽黎杉 §f| §a普通的玩家312 §f| §a羽颖YY §f| §ahuangDMST
		§r§f§l◀【§6幻§e想§b乡 §f● §6留名墙§f】▶
		§r§b§l诺白郡主 §f| §b字幕君 §f| §b我布是远古 §f| §b我布是白给 §f| §b府白呀
		§r§b§lLZN乄德方 §f| §bCZG_丑八怪 §f| §b小雨4xy §f| §b牟菇2 §f| §bBMG2HFet
		§r§b§ltrdfjtss §f| §b家养高粱长跑 §f| §b紫依康 §f| §b亲爱的骷吸气副
		§r§b§l全服最肝 §f| §b诚挚的末影娘灌篮`,
							},
						},
					)
					fmt.Printf("%#v\n", resp)
					holderA := api.Resources.Container.Occupy()
					api.OpenInventory()
					for _, value := range resp {
						itemData, _ := api.Resources.Inventory.GetItemStackInfo(0, value.Destination.Slot)
						api.DropItemAll(
							protocol.StackRequestSlotInfo{
								ContainerID:    0xc,
								Slot:           value.Destination.Slot,
								StackNetworkID: itemData.StackNetworkID,
							},
							0,
						)
					}
					api.CloseContainer()
					api.Resources.Container.Release(holderA)
				}()
			}
	*/

	for {
		if breaker != nil {
			select {
			case <-breaker:
				return
			default:
			}
		}

		var pk packet.Packet
		var data []byte
		var err error
		if cache := env.CachedPacket.(<-chan CachedPacket); len(cache) > 0 {
			c := <-cache
			pk, data = c.Packet, c.Data
		} else {
			if pk, data, err = conn.ReadPacketAndBytes(); err != nil {
				panic(err)
			}
			if args.ShouldEnableOmegaSystem && !env.OmegaHasBootstrap {
				go func() {
					_, cb := embed.EnableOmegaSystem(env)
					cb()
				}()
				env.OmegaHasBootstrap = true
			}
		}

		env.ResourcesUpdater.(func(*packet.Packet))(&pk)

		if env.OmegaAdaptorHolder != nil {
			env.OmegaAdaptorHolder.(*embed.EmbeddedAdaptor).FeedPacketAndByte(pk, data)
			continue
		}

		env.UQHolder.(*uqHolder.UQHolder).Update(pk)
		if env.ExternalConnectionHandler != nil {
			env.ExternalConnectionHandler.(*external.ExternalConnectionHandler).PacketChannel <- data
		}
		// fmt.Println(omega_utils.PktIDInvMapping[int(pk.ID())])
		switch p := pk.(type) {
		case *packet.PyRpc:
			onPyRpc(p, env)
		case *packet.Text:
			if p.TextType == packet.TextTypeChat {
				if args.InGameResponse {
					if p.SourceName == env.RespondTo {
						functionHolder.Process(p.Message)
					}
				}
				break
			}
		case *packet.ActorEvent:
			if p.EventType == packet.ActorEventDeath && p.EntityRuntimeID == conn.GameData().EntityRuntimeID {
				conn.WritePacket(&packet.PlayerAction{
					EntityRuntimeID: conn.GameData().EntityRuntimeID,
					ActionType:      protocol.PlayerActionRespawn,
				})
			}
		case *packet.SubChunk:
			chunkData := chunkAssembler.OnNewSubChunk(p)
			if chunkData != nil {
				env.ChunkFeeder.(*global.ChunkFeeder).OnNewChunk(chunkData)
				env.LRUMemoryChunkCacher.(*lru.LRUMemoryChunkCacher).Write(chunkData)
			}
		case *packet.NetworkChunkPublisherUpdate:
			// pterm.Info.Println("packet.NetworkChunkPublisherUpdate", p)
			// missHash := []uint64{}
			// hitHash := []uint64{}
			// for i := uint64(0); i < 64; i++ {
			// 	missHash = append(missHash, uint64(10184224921554030005+i))
			// 	hitHash = append(hitHash, uint64(6346766690299427078-i))
			// }
			// conn.WritePacket(&packet.ClientCacheBlobStatus{
			// 	MissHashes: missHash,
			// 	HitHashes:  hitHash,
			// })
		case *packet.LevelChunk:
			// pterm.Info.Println("LevelChunk", p.BlobHashes, len(p.BlobHashes), p.CacheEnabled)
			// go func() {
			// 	for {

			// conn.WritePacket(&packet.ClientCacheBlobStatus{
			// 	MissHashes: []uint64{p.BlobHashes[0] + 1},
			// 	HitHashes:  []uint64{},
			// })
			// 		time.Sleep(100 * time.Millisecond)
			// 	}
			// }()
			if fbtask.CheckHasWorkingTask(env) {
				break
			}
			if exist := chunkAssembler.AddPendingTask(p); !exist {
				requests := chunkAssembler.GenRequestFromLevelChunk(p)
				chunkAssembler.ScheduleRequest(requests)
			}
		case *packet.Respawn:
			if p.EntityRuntimeID == conn.GameData().EntityRuntimeID {
				move.Position = p.Position
			}
		case *packet.MovePlayer:
			if p.EntityRuntimeID == conn.GameData().EntityRuntimeID {
				move.Position = p.Position
			} else if p.EntityRuntimeID == move.TargetRuntimeID {
				move.Target = p.Position
			}
		case *packet.CorrectPlayerMovePrediction:
			move.MoveP += 10
			if move.MoveP > 100 {
				move.MoveP = 0
			}
			move.Position = p.Position
			move.Jump()
		case *packet.AddPlayer:
			if move.TargetRuntimeID == 0 && p.EntityRuntimeID != conn.GameData().EntityRuntimeID {
				move.Target = p.Position
				move.TargetRuntimeID = p.EntityRuntimeID
				//fmt.Printf("Got target: %s\n",p.Username)
			}
		}
	}
}

func InitializeMinecraftConnection(ctx context.Context, authenticator minecraft.Authenticator) (conn *minecraft.Conn, err error) {
	if args.DebugMode {
		conn = &minecraft.Conn{
			DebugMode: true,
		}
	} else {
		dialer := minecraft.Dialer{
			Authenticator: authenticator,
		}
		conn, err = dialer.DialContext(ctx, "raknet")
	}
	if err != nil {
		return
	}
	conn.WritePacket(&packet.ClientCacheStatus{
		Enabled: false,
	})
	runtimeid := fmt.Sprintf("%d", conn.GameData().EntityUniqueID)
	conn.WritePacket(&packet.PyRpc{
		Value: py_rpc.Marshal(&py_rpc.SyncUsingMod{}),
	})
	conn.WritePacket(&packet.PyRpc{
		Value: py_rpc.Marshal(&py_rpc.SyncVipSkinUUID{nil}),
	})
	if !args.SkipMCPCheckChallenges {
		conn.WritePacket(&packet.PyRpc{
			Value: py_rpc.Marshal(&py_rpc.ClientLoadAddonsFinishedFromGac{}),
		})
	}
	{
		event := cts_mc_p.GetLoadedInstances{PlayerRuntimeID: runtimeid}
		module := cts_mc.Preset{Module: &mei.DefaultModule{Event: &event}}
		park := cts.Minecraft{Default: mei.Default{Module: &module}}
		conn.WritePacket(&packet.PyRpc{
			Value: py_rpc.Marshal(&py_rpc.ModEvent{
				Package: &park,
				Type:    py_rpc.ModEventClientToServer,
			}),
		})
	}
	conn.WritePacket(&packet.PyRpc{
		Value: py_rpc.Marshal(&py_rpc.ArenaGamePlayerFinishLoad{}),
	})
	{
		event := cts_mc_v.PlayerUiInit{RuntimeID: runtimeid}
		module := cts_mc.VIPEventSystem{Module: &mei.DefaultModule{Event: &event}}
		park := cts.Minecraft{Default: mei.Default{Module: &module}}
		conn.WritePacket(&packet.PyRpc{
			Value: py_rpc.Marshal(&py_rpc.ModEvent{
				Package: &park,
				Type:    py_rpc.ModEventClientToServer,
			}),
		})
	}
	return
}

func EstablishConnectionAndInitEnv(env *environment.PBEnvironment) {
	if env.FBAuthClient == nil {
		env.ClientOptions.AuthServer = args.AuthServer
		env.ClientOptions.RespondUserOverride = args.CustomGameName
		env.FBAuthClient = fbauth.CreateClient(env.ClientOptions)
	}
	env.MCPCheckChallengeSolveDown = make(chan struct{}, 1)
	pterm.Println(pterm.Yellow(fmt.Sprintf("%s: %s", I18n.T(I18n.ServerCodeTrans), env.LoginInfo.ServerCode)))

	if args.ExternalListenAddress != "" {
		external.ListenExt(env, args.ExternalListenAddress)
	}

	ctx, _ := context.WithTimeout(context.Background(), time.Second*30)
	authenticator := fbauth.NewAccessWrapper(
		env.FBAuthClient.(*fbauth.Client),
		env.LoginInfo.ServerCode,
		env.LoginInfo.ServerPasscode,
		env.LoginInfo.Token,
		env.LoginInfo.Username,
		env.LoginInfo.Password,
	)
	conn, err := InitializeMinecraftConnection(ctx, authenticator)

	if err != nil {
		pterm.Error.Println(err)
		if runtime.GOOS == "windows" {
			pterm.Error.Println(I18n.T(I18n.Crashed_OS_Windows))
			_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
		}
		panic(err)
	}
	if len(env.RespondTo) == 0 {
		if args.CustomGameName != "" {
			env.RespondTo = args.CustomGameName
		} else {
			env.RespondTo = env.FBAuthClient.(*fbauth.Client).RespondTo
		}
	}

	env.Connection = conn
	SolveMCPCheckChallenges(env)
	pterm.Println(pterm.Yellow(I18n.T(I18n.ConnectionEstablished)))

	env.UQHolder = uqHolder.NewUQHolder(conn.GameData().EntityRuntimeID)
	env.UQHolder.(*uqHolder.UQHolder).UpdateFromConn(conn)
	env.UQHolder.(*uqHolder.UQHolder).CurrentTick = 0
	env.Resources = &ResourcesControl.Resources{}
	env.ResourcesUpdater = env.Resources.(*ResourcesControl.Resources).Init()
	env.GameInterface = &GameInterface.GameInterface{
		WritePacket: env.Connection.(*minecraft.Conn).WritePacket,
		ClientInfo: GameInterface.ClientInfo{
			DisplayName:     env.Connection.(*minecraft.Conn).IdentityData().DisplayName,
			ClientIdentity:  env.Connection.(*minecraft.Conn).IdentityData().Identity,
			XUID:            env.Connection.(*minecraft.Conn).IdentityData().XUID,
			EntityRuntimeID: env.Connection.(*minecraft.Conn).GameData().EntityRuntimeID,
			EntityUniqueID:  env.Connection.(*minecraft.Conn).GameData().EntityUniqueID,
		},
		Resources: env.Resources.(*ResourcesControl.Resources),
	}
	if args.SkipMCPCheckChallenges {
		env.GameInterface.SendSettingsCommand("gamerule sendcommandfeedback false", true)
	}

	functionHolder := env.FunctionHolder.(*function.FunctionHolder)
	function.InitPresetFunctions(functionHolder)
	fbtask.InitTaskStatusDisplay(env)

	move.ConnectTime = time.Time{}
	move.Position = conn.GameData().PlayerPosition
	move.Pitch = conn.GameData().Pitch
	move.Yaw = conn.GameData().Yaw
	move.Connection = conn
	move.RuntimeID = conn.GameData().EntityRuntimeID

	signalhandler.Install(conn, env)

	taskholder := env.TaskHolder.(*fbtask.TaskHolder)
	types.ForwardedBrokSender = taskholder.BrokSender

	env.UQHolder.(*uqHolder.UQHolder).UpdateFromConn(conn)
}

func SolveMCPCheckChallenges(env *environment.PBEnvironment) {
	if args.SkipMCPCheckChallenges {
		env.CachedPacket = (<-chan packet.Packet)(make(chan packet.Packet))
		return
	}
	// check
	challengeTimeout := false
	challengeSolved := make(chan struct{}, 1)
	cachedPkt := make(chan CachedPacket, 32767)
	commandOutput := make(chan packet.CommandOutput, 1)
	timer := time.NewTimer(time.Second * 30)
	// prepare
	go func() {
		for {
			if challengeTimeout {
				return
			}
			// challenge timeout
			pk, data, err := env.Connection.(*minecraft.Conn).ReadPacketAndBytes()
			if !challengeTimeout && err != nil {
				panic(fmt.Sprintf("SolveMCPCheckChallenges: %v", err))
			}
			// read packet
			switch p := pk.(type) {
			case *packet.PyRpc:
				older_states := env.GetCheckNumEverPassed
				onPyRpc(p, env)
				if !older_states && env.GetCheckNumEverPassed {
					challengeSolved <- struct{}{}
				}
			case *packet.CommandOutput:
				commandOutput <- *p
				return
			default:
				cachedPkt <- CachedPacket{pk, data}
			}
			// for each incoming packet
		}
	}()
	// read packet and process
	select {
	case <-challengeSolved:
		WaitMCPCheckChallengesDown(env, commandOutput)
		env.MCPCheckChallengeSolveDown <- struct{}{}
		close(challengeSolved)
		close(cachedPkt)
		env.CachedPacket = (<-chan CachedPacket)(cachedPkt)
		return
	case <-timer.C:
		challengeTimeout = true
		panic("SolveMCPCheckChallenges: Failed to pass the MCPC check challenges, please try again later")
	}
	// wait for the challenge to end
}

func WaitMCPCheckChallengesDown(
	env *environment.PBEnvironment,
	command_output chan packet.CommandOutput,
) {
	ticker := time.NewTicker(time.Millisecond * 50)
	defer ticker.Stop()
	for {
		err := env.Connection.(*minecraft.Conn).WritePacket(&packet.CommandRequest{
			CommandLine: "WaitMCPCheckChallengesDown",
			CommandOrigin: protocol.CommandOrigin{
				Origin:    protocol.CommandOriginAutomationPlayer,
				UUID:      ResourcesControl.GenerateUUID(),
				RequestID: GameInterface.DefaultCommandRequestID,
			},
			Internal:  false,
			UnLimited: false,
		})
		if err != nil {
			panic(fmt.Sprintf("WaitMCPCheckChallengesDown: %v", err))
		}
		select {
		case <-command_output:
			close(command_output)
			return
		case <-ticker.C:
		}
	}
}

func onPyRpc(p *packet.PyRpc, env *environment.PBEnvironment) {
	conn := env.Connection.(*minecraft.Conn)
	if p.Error != nil {
		panic(fmt.Sprintf("onPyRpc: %v", p.Error))
	}
	if p.Value == nil {
		return
	}
	// prepare
	content, err := py_rpc.Unmarshal(p.Value)
	if err != nil {
		if env.GameInterface == nil {
			pterm.Warning.Printf("onPyRpc: %v\n", err)
		} else {
			env.GameInterface.Output(pterm.Warning.Sprintf("onPyRpc: %v", err))
		}
		return
	}
	// unmarshal
	switch c := content.(type) {
	case *py_rpc.HeartBeat:
		c.Type = py_rpc.ClientToServerHeartBeat
		conn.WritePacket(&packet.PyRpc{Value: py_rpc.Marshal(c)})
		// heart beat to test the device is still alive?
		// it seems that we just need to return it back to the server is OK
	case *py_rpc.StartType:
		if args.SkipMCPCheckChallenges {
			break
		}
		// if the user request us do not pass this challenge
		client := env.FBAuthClient.(*fbauth.Client)
		c.Content = client.TransferData(c.Content)
		c.Type = py_rpc.StartTypeResponse
		conn.WritePacket(&packet.PyRpc{Value: py_rpc.Marshal(c)})
		// get data and send packet
	case *py_rpc.GetMCPCheckNum:
		if args.SkipMCPCheckChallenges || env.GetCheckNumEverPassed {
			break
		}
		// if the challenges has been down,
		// or the user request us do not pass this challenge,
		// we do NOTHING
		client := env.FBAuthClient.(*fbauth.Client)
		arg, _ := json.Marshal([]any{
			c.FirstArg,
			c.SecondArg.Arg,
			env.Connection.(*minecraft.Conn).GameData().EntityUniqueID,
		})
		ret := client.TransferCheckNum(string(arg))
		// create request to the auth server and get response
		ret_p := []any{}
		json.Unmarshal([]byte(ret), &ret_p)
		if len(ret_p) > 7 {
			ret6, ok := ret_p[6].(float64)
			if ok {
				ret_p[6] = int64(ret6)
			}
		}
		// unmarshal response and adjust the data included
		conn.WritePacket(&packet.PyRpc{
			Value: py_rpc.Marshal(&py_rpc.SetMCPCheckNum{ret_p}),
		})
		env.GetCheckNumEverPassed = true
		// send packet and mark this challenges was finished
	}
	// do some actions for some specific PyRpc packets
}

func getUserInputMD5() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("MD5: ")
	code, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(code, "\r\n"), err
}
