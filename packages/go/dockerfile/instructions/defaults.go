package instructions

import "github.com/ep0ll/dockerfile-parser/packages/go/parser"

// DefaultOptions registers every built-in Dockerfile instruction handler.
// Each handler can be replaced independently with WithInstructionHandler.
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

// DefaultFlagOptions composes the default flags declared next to each
// instruction handler. Custom dialects can replace the complete schema for
// an individual instruction and scope with parser.WithFlagSpecs.
func DefaultFlagOptions() []parser.Option {
	return []parser.Option{
		addFlagOptions(),
		copyFlagOptions(),
		fromFlagOptions(),
		runFlagOptions(),
		healthcheckFlagOptions(),
		envFlagOptions(),
		labelFlagOptions(),
		volumeFlagOptions(),
	}
}
