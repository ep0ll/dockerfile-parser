package instructions

import "github.com/ep0ll/dockerfile-parser/packages/go/parser"

func DefaultOptions() []parser.Option {
	return []parser.Option{
		parser.WithInstructionHandler("ADD", ParseAdd),
		parser.WithInstructionHandler("ARG", ParseArg),
		parser.WithInstructionHandler("CMD", ParseCmd),
		parser.WithInstructionHandler("COPY", ParseCopy),
		parser.WithInstructionHandler("ENTRYPOINT", ParseEntrypoint),
		parser.WithInstructionHandler("ENV", ParseEnv),
		parser.WithInstructionHandler("EXPOSE", ParseExpose),
		parser.WithInstructionHandler("FROM", ParseFrom),
		parser.WithInstructionHandler("HEALTHCHECK", ParseHealthcheck),
		parser.WithInstructionHandler("LABEL", ParseLabel),
		parser.WithInstructionHandler("MAINTAINER", ParseMaintainer),
		parser.WithInstructionHandler("ONBUILD", ParseOnbuild),
		parser.WithInstructionHandler("RUN", ParseRun),
		parser.WithInstructionHandler("SHELL", ParseShell),
		parser.WithInstructionHandler("STOPSIGNAL", ParseStopsignal),
		parser.WithInstructionHandler("USER", ParseUser),
		parser.WithInstructionHandler("VOLUME", ParseVolume),
		parser.WithInstructionHandler("WORKDIR", ParseWorkdir),
	}
}

func DefaultFlagOptions() []parser.Option {
	value := parser.FlagRequiredValue
	optional := parser.FlagOptionalInlineValue
	return []parser.Option{
		parser.WithFlagSpecs("", "ADD",
			parser.FlagSpec{Name: "chown", ValueMode: value}, parser.FlagSpec{Name: "chmod", ValueMode: value},
			parser.FlagSpec{Name: "link", ValueMode: optional}, parser.FlagSpec{Name: "exclude", ValueMode: value, Repeatable: true},
			parser.FlagSpec{Name: "checksum", ValueMode: value}, parser.FlagSpec{Name: "keep-git-dir", ValueMode: optional}, parser.FlagSpec{Name: "unpack", ValueMode: optional}),
		parser.WithFlagSpecs("", "COPY",
			parser.FlagSpec{Name: "from", ValueMode: value}, parser.FlagSpec{Name: "chown", ValueMode: value},
			parser.FlagSpec{Name: "chmod", ValueMode: value}, parser.FlagSpec{Name: "link", ValueMode: optional},
			parser.FlagSpec{Name: "parents", ValueMode: optional}, parser.FlagSpec{Name: "exclude", ValueMode: value, Repeatable: true}),
		parser.WithFlagSpecs("", "FROM", parser.FlagSpec{Name: "platform", ValueMode: value}),
		parser.WithFlagSpecs("", "RUN",
			parser.FlagSpec{Name: "mount", ValueMode: value, Repeatable: true}, parser.FlagSpec{Name: "network", ValueMode: value},
			parser.FlagSpec{Name: "security", ValueMode: value}, parser.FlagSpec{Name: "device", ValueMode: value, Repeatable: true}),
		parser.WithFlagSpecs("", "HEALTHCHECK",
			parser.FlagSpec{Name: "interval", ValueMode: value}, parser.FlagSpec{Name: "timeout", ValueMode: value},
			parser.FlagSpec{Name: "start-period", ValueMode: value}, parser.FlagSpec{Name: "start-interval", ValueMode: value},
			parser.FlagSpec{Name: "retries", ValueMode: value}),
		parser.WithFlagSpecs("", "ENV", parser.FlagSpec{Name: "unset", ValueMode: optional, Repeatable: false}),
		parser.WithFlagSpecs("", "LABEL", parser.FlagSpec{Name: "unset", ValueMode: optional, Repeatable: false}),
	}
}
