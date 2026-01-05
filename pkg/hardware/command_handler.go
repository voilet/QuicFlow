package hardware

import (
	"encoding/json"

	"github.com/voilet/quic-flow/pkg/command"
)

// NewCommandResultHandler 创建硬件命令结果处理器
// 当 hardware.info 命令完成时，自动保存硬件信息到数据库
func NewCommandResultHandler(store *Store) command.CommandResultHandler {
	return command.CommandResultHandlerFunc(func(cmd *command.Command) {
		// 只处理成功的 hardware.info 命令
		if cmd.CommandType != command.CmdHardwareInfo {
			return
		}
		if cmd.Status != command.CommandStatusCompleted {
			return
		}
		if len(cmd.Result) == 0 {
			return
		}

		// 解析硬件信息
		var hardwareInfo command.HardwareInfoResult
		if err := json.Unmarshal(cmd.Result, &hardwareInfo); err != nil {
			// 解析失败，记录但不影响主流程
			return
		}

		// 保存到数据库（使用 upsert）
		_, err := store.SaveHardwareInfo(cmd.ClientID, &hardwareInfo)
		if err != nil {
			// 保存失败，记录但不影响主流程
			// 可以考虑使用日志系统记录
		}
	})
}
