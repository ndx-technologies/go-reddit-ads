package main

import (
	"flag"
	"log"
	"maps"
	"slices"
	"strings"

	cmdapply "github.com/ndx-technologies/go-reddit-ads/cmd/apply"
	cmdauth "github.com/ndx-technologies/go-reddit-ads/cmd/auth"
	cmdfetch "github.com/ndx-technologies/go-reddit-ads/cmd/fetch"
	cmdstatsadgroups "github.com/ndx-technologies/go-reddit-ads/cmd/stats_adgroups"
	cmdstatsads "github.com/ndx-technologies/go-reddit-ads/cmd/stats_ads"
	cmdstatscampaigns "github.com/ndx-technologies/go-reddit-ads/cmd/stats_campaigns"
	cmdtimeline "github.com/ndx-technologies/go-reddit-ads/cmd/timeline"
)

type CommandInfo struct {
	DocShort string
	Run      func(args []string)
}

const doc = `Reddit Ads Toolkit

Tools for structured access to Reddit Ads, export/import config, apply changes.
Use this toolkit to setup your AI-driven Reddit Ads GitOps.
`

var commands = map[string]CommandInfo{
	"auth":            {DocShort: cmdauth.DocShort, Run: cmdauth.Run},
	"fetch":           {DocShort: cmdfetch.DocShort, Run: cmdfetch.Run},
	"apply":           {DocShort: cmdapply.DocShort, Run: cmdapply.Run},
	"stats campaigns": {DocShort: cmdstatscampaigns.DocShort, Run: cmdstatscampaigns.Run},
	"stats adgroups":  {DocShort: cmdstatsadgroups.DocShort, Run: cmdstatsadgroups.Run},
	"stats ads":       {DocShort: cmdstatsads.DocShort, Run: cmdstatsads.Run},
	"timeline":        {DocShort: cmdtimeline.DocShort, Run: cmdtimeline.Run},
}

func main() {
	cmdNames := slices.Collect(maps.Keys(commands))

	flag.Usage = func() {
		w := flag.CommandLine.Output()
		w.Write([]byte(doc))
		w.Write([]byte("\nUsage:\n\n"))
		slices.Sort(cmdNames)
		for _, name := range cmdNames {
			w.Write([]byte(" " + name + " - " + commands[name].DocShort + "\n"))
		}
	}
	flag.Parse()

	cmd, rest := route(flag.Args(), cmdNames)
	if cmd == "" {
		flag.Usage()
		log.Fatal("unknown command, use -h for help")
	}

	commands[cmd].Run(rest)
}

func route(args []string, commands []string) (string, []string) {
	if len(args) == 0 {
		return "", nil
	}
	bestIdx := -1
	bestLen := 0
	bestRest := []string(nil)
	for i, cmd := range commands {
		parts := strings.Fields(cmd)
		if len(args) >= len(parts) && slices.Equal(args[:len(parts)], parts) {
			if len(parts) > bestLen {
				bestIdx = i
				bestLen = len(parts)
				bestRest = args[len(parts):]
			}
		}
	}
	if bestIdx < 0 {
		return "", nil
	}
	if len(bestRest) > 0 && !strings.HasPrefix(bestRest[0], "-") {
		return "", nil
	}
	return commands[bestIdx], bestRest
}
