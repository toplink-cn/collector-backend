package system

import (
	"bytes"
	model_system "collector-backend/models/system"
	"collector-backend/pkg/logger"
	"collector-backend/util"
	"context"
	"encoding/json"
	"os/exec"
	"time"

	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

const CmdTimeout = 5 * time.Second

type SystemCollector struct {
	SystemInfo *model_system.SystemInfo
}

func NewSystemCollector(s *model_system.SystemInfo) *SystemCollector {
	return &SystemCollector{
		SystemInfo: s,
	}
}

func (sc *SystemCollector) Collect() {
	time.Sleep(10 * time.Second)
	sc.collectIOStat()
	sc.collectRam()
	sc.collectDisk()
	sc.collectNet()
	sc.SystemInfo.Time = time.Now()
	// logger.Println("collect done \n", sc.SystemInfo)
}

func (sc *SystemCollector) collectIOStat() {
	args := []string{"-x", "-o", "JSON", "1", "2"}
	out, err := sc.run("iostat", args)
	if err != nil {
		return
	}

	var iostat model_system.IoStat
	if err := json.Unmarshal([]byte(out), &iostat); err != nil {
		logger.Println("Failed to unmarshal JSON")
		return
	}

	if len(iostat.Sysstat.Hosts) > 0 {
		firstHost := iostat.Sysstat.Hosts[0]
		if len(firstHost.Statistics) > 0 {
			currentStatistic := firstHost.Statistics[0]
			percentage := 100 - currentStatistic.AvgCpu.Idel
			percentage = float32(util.RoundFloat(float64(percentage), 0))
			cpuParame := model_system.Parame{
				Key:   "cpu",
				Value: map[string]interface{}{"percentage": percentage},
			}
			sc.SystemInfo.Parames = append(sc.SystemInfo.Parames, cpuParame)

			disks := map[string]map[string]interface{}{}
			for _, d := range currentStatistic.Disks {
				disk := map[string]interface{}{}
				disk["kB_wrtn/s"] = d.WriteKBS
				disk["kB_read/s"] = d.ReadKBS
				disk["util"] = d.Util
				disks[d.DiskDevice] = disk
			}
			disksParame := model_system.Parame{
				Key:   "io",
				Value: disks,
			}
			sc.SystemInfo.Parames = append(sc.SystemInfo.Parames, disksParame)
		}
	}
}

func (sc *SystemCollector) collectRam() {
	vm, err := mem.VirtualMemory()
	if err != nil {
		logger.Printf("Failed to get virtual memory info, err: %v \n", err.Error())
		return
	}

	disksParame := model_system.Parame{
		Key: "ram",
		Value: map[string]interface{}{
			"total":      float32(vm.Total / 1024 / 1024),
			"used":       float32(vm.Used / 1024 / 1024),
			"percentage": float32(util.RoundFloat(vm.UsedPercent, 0)),
		},
	}
	sc.SystemInfo.Parames = append(sc.SystemInfo.Parames, disksParame)
}

func (sc *SystemCollector) collectDisk() {
	args := []string{"-c", `mount | grep /app | grep -v iso | grep -v /app/run | awk '{print $3}'`}
	out, err := sc.run("bash", args)
	if err != nil {
		logger.Println(err.Error())
		return
	}

	lines := bytes.Split(out, []byte{'\n'})
	diskPath := map[string]map[string]interface{}{}
	var total uint64
	var used uint64
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		path := string(line)
		usage, err := disk.Usage(path)
		if err != nil {
			logger.Println(err.Error())
			continue
		}

		usageStat := map[string]interface{}{
			"total":      usage.Total,
			"used":       usage.Used,
			"percentage": usage.UsedPercent,
		}

		diskPath[path] = usageStat

		total += usage.Total
		used += usage.Used
	}
	var percentage uint64
	if used == 0 || total == 0 {
		percentage = 0
	} else {
		percentage = used / total * 100
	}

	disksParame := model_system.Parame{
		Key: "disk",
		Value: map[string]interface{}{
			"total":      float32(total),
			"used":       float32(used),
			"percentage": float32(percentage),
		},
	}
	sc.SystemInfo.Parames = append(sc.SystemInfo.Parames, disksParame)

	diskPathParame := model_system.Parame{
		Key:   "disk_path",
		Value: diskPath,
	}
	sc.SystemInfo.Parames = append(sc.SystemInfo.Parames, diskPathParame)
}

func (sc *SystemCollector) collectNet() {
	ioCountersStat, err := net.IOCounters(true)
	if err != nil {
		logger.Println(err.Error())
		return
	}

	network := map[string]map[string]interface{}{}
	for _, ioCounterStat := range ioCountersStat {
		if ioCounterStat.Name == "veth" || ioCounterStat.Name == "br-" || ioCounterStat.Name == "docker" {
			continue
		}
		ioStat := map[string]interface{}{
			"in":  float32(ioCounterStat.BytesRecv),
			"out": float32(ioCounterStat.BytesSent),
		}

		network[ioCounterStat.Name] = ioStat
	}

	diskPathParame := model_system.Parame{
		Key:   "network",
		Value: network,
	}
	sc.SystemInfo.Parames = append(sc.SystemInfo.Parames, diskPathParame)
}

func (sc *SystemCollector) run(command string, args []string) ([]byte, error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, CmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, err
	}
	return out, nil
}
