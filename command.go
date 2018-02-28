package main

import (
	"fmt"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/urfave/cli"
)

type Command struct {
	Name         string   `json:"name"`
	Aliases      []string `json:"aliases"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	Usage        string   `json:"usage"`
	Arguments    string   `json:"arguments"`
	Bin          string   `json:"bin"`
	AutoComplete bool     `json:"auto-complete"`

	Flags       []cli.Flag    `json:"-"`
	Docs        string        `json:"-"`
	BinSuffix   string        `json:"-"`
	OS          string        `json:"-"`
	Arch        string        `json:"-"`
	Subcommands []cli.Command `json:"-"`
}

func packageListDiff(oldcmds []commandPackage) {
	cmds := getCommands()

	var old []Command
	for _, oldcmd := range oldcmds {
		for _, cmd := range oldcmd.Commands {
			old = append(old, cmd)
		}
	}

	var new []Command
	for _, newcmd := range cmds {
		for _, cmd := range newcmd.Commands {
			new = append(new, cmd)
		}
	}

	var unchanged = make(map[string]bool)
	var added = make(map[string]bool)
	var removed = make(map[string]bool)

	for _, newCmd := range new {
		found := false
		for _, oldCmd := range old {
			if newCmd.Name == oldCmd.Name {
				found = true
				unchanged[newCmd.Name] = true
				break
			}
		}

		if !found {
			added[newCmd.Name] = true
		}
	}

	for _, oldCmd := range old {
		found := false
		for _, newCmd := range new {
			if newCmd.Name == oldCmd.Name {
				found = true
				break
			}
		}

		if !found {
			removed[oldCmd.Name] = true
		}
	}

	bold := color.New(color.FgWhite, color.Bold)

	color.Yellow("\nAvailable Commands:\n\n")
	for _, cmd := range getCommands() {
		for _, command := range cmd.Commands {
			if _, ok := unchanged[command.Name]; ok {
				bold.Printf("  %s", command.Name)
			} else if _, ok := added[command.Name]; ok {
				fmt.Print(color.GreenString("  %s", command.Name))
			} else if _, ok := removed[command.Name]; ok {
				fmt.Print(color.RedString("  %s", command.Name))
			}
			if len(command.Aliases) > 0 {
				var aliases string

				if len(command.Aliases) == 1 {
					aliases = "alias"
				} else {
					aliases = "aliases"
				}

				fmt.Printf(" (%s: ", aliases)
				for i, alias := range command.Aliases {
					if _, ok := unchanged[command.Name]; ok {
						bold.Print(alias)
					} else if _, ok := added[command.Name]; ok {
						fmt.Print(color.GreenString(alias))
					} else if _, ok := removed[command.Name]; ok {
						fmt.Print(color.RedString(alias))
					}

					if i < len(command.Aliases)-1 {
						fmt.Print(", ")
					}
				}
				fmt.Print(")")
			}

			fmt.Println()

			fmt.Printf("    %s\n", command.Description)
		}
	}
	fmt.Printf("\nSee \"%s\" for details.\n", color.BlueString("%s help [command]", self()))
}

func getBuiltinCommands() []commandPackage {
	commands := []commandPackage{
		{
			Commands: []Command{
				{
					Name:        "help",
					Description: "Displays help information",
					Arguments:   "[command] [sub-command]",
				},
			},
			action: cmdHelp,
		},
		{
			Commands: []Command{
				{
					Name:        "list",
					Description: "Displays available commands",
					Flags: []cli.Flag{
						cli.BoolFlag{
							Name:  "remote",
							Usage: "Display all available packages",
						},
					},
				},
			},
			action: cmdList,
		},
		{
			Commands: []Command{
				{
					Name:        "config",
					Arguments:   "<action> <setting> [value]",
					Description: "Manage configuration",
					Subcommands: []cli.Command{
						{
							Name:      "get",
							ArgsUsage: "<setting>",
							Action:    cmdConfigGet,
						},
						{
							Name:      "set",
							ArgsUsage: "<setting> <value>",
							Action:    cmdConfigSet,
						},
						{
							Name:      "list",
							ArgsUsage: "[section]",
							Action:    cmdConfigList,
						},
						{
							Name:      "unset",
							Aliases:   []string{"rm"},
							ArgsUsage: "<setting>",
							Action:    cmdConfigUnset,
						},
					},
				},
			},
		},
		{
			Commands: []Command{
				{
					Name:        "install",
					Arguments:   "<package name or repository URL>...",
					Description: "Fetch and install packages from a Git repository.",
					Flags: []cli.Flag{
						cli.BoolFlag{
							Name:  "force",
							Usage: "Force binary installation if available when source installation fails",
						},
					},
					Aliases: []string{"get"},
					Docs:    "Examples:\n\n   akamai install property purge\n   akamai install akamai/cli-property\n   akamai install git@github.com:akamai/cli-property.git\n   akamai install https://github.com/akamai/cli-property.git",
				},
			},
			action: cmdInstall,
		},
		{
			Commands: []Command{
				{
					Name:        "search",
					Arguments:   "<keyword>...",
					Description: "Search for packages in the official Akamai CLI package repository",
					Docs:        "Examples:\n\n   akamai search property",
				},
			},
			action: cmdSearch,
		},
		{
			Commands: []Command{
				{
					Name:        "uninstall",
					Arguments:   "<command>...",
					Description: "Uninstall package containing <command>",
				},
			},
			action: cmdUninstall,
		},
		{
			Commands: []Command{
				{
					Name:        "update",
					Arguments:   "[<command>...]",
					Description: "Update one or more commands. If no command is specified, all commands are updated",
					Flags: []cli.Flag{
						cli.BoolFlag{
							Name:  "force",
							Usage: "Force binary installation if available when source installation fails",
						},
					},
				},
			},
			action: cmdUpdate,
		},
	}

	upgradeCommand := getUpgradeCommand()
	if upgradeCommand != nil {
		commands = append(commands, *upgradeCommand)
	}

	return commands
}

func getCommands() []commandPackage {
	var commands []commandPackage
	var commandMap map[string]bool = make(map[string]bool)

	for _, cmd := range getBuiltinCommands() {
		commandMap[cmd.Commands[0].Name] = true
		commands = append(commands, cmd)
	}

	packagePaths := getPackagePaths()
	if packagePaths == "" {
		return commands
	}

	for _, dir := range filepath.SplitList(packagePaths) {
		cmdPackage, err := readPackage(dir)
		if err == nil {
			commands = append(commands, cmdPackage)
		}
	}

	return commands
}