package cli

import "github.com/spf13/cobra"

const usageTemplate = `{{if .Runnable}}Использование:
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}Использование:
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Алиасы:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Примеры:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Доступные команды:
{{range .Commands}}{{if .IsAvailableCommand}}  {{rpad .Name .NamePadding }} {{.Short}}
{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Флаги:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Глобальные флаги:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Дополнительная справка:
  {{.CommandPath}} help [command]{{end}}
`

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "catalog",
		Short: "CLI для каталогизации и профилирования источников данных",
		Long: `catalog читает конфигурацию запуска, обходит источники данных,
снимает паспорт датасетов, сохраняет слепки в каталог
и показывает отчёты и различия.`,
		Example: `  catalog run --config ./demo/config/demo.yaml
  catalog report --latest
  catalog report --run-id 42
  catalog diff --latest
  catalog diff --from-run-id 41 --to-run-id 42`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.SetUsageTemplate(usageTemplate)
	cmd.SetHelpTemplate(usageTemplate)
	cmd.CompletionOptions.DisableDefaultCmd = true

	cmd.SetHelpCommand(&cobra.Command{
		Use:   "help [command]",
		Short: "Справка по командам",
		Long:  "Показать справку по одной команде или по всему CLI.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(helpCmd *cobra.Command, args []string) error {
			target := helpCmd.Root()

			if len(args) > 0 {
				found, _, err := helpCmd.Root().Find(args)
				if err != nil {
					return err
				}
				target = found
			}

			return target.Help()
		},
	})

	cmd.AddCommand(NewRunCmd())
	cmd.AddCommand(NewReportCmd())
	cmd.AddCommand(NewDiffCmd())

	return cmd
}
