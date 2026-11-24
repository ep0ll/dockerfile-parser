package dockerfile_test

import (
	"strings"
	"testing"

	"github.com/ep0ll/dockerfile-parser/packages/go/dockerfile"
	"github.com/ep0ll/dockerfile-parser/packages/go/dockerfile/heredoc"
)

func TestRunHeredocIsCollectedBeforeReturningInstruction(t *testing.T) {
	source := "RUN cat <<EOF\nhello world\nEOF\nFROM alpine\n"
	p, err := dockerfile.NewParser(strings.NewReader(source))
	if err != nil {
		t.Fatal(err)
	}
	run, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	if run.Name != "RUN" || len(run.Bodies) != 1 {
		t.Fatalf("unexpected RUN instruction: %#v", run)
	}
	body := run.Bodies[0]
	if body.Kind != "heredoc" || body.Name != "EOF" || body.Content != "hello world\n" {
		t.Fatalf("unexpected heredoc body: %#v", body)
	}
	if body.Position.StartLine != 2 || body.Position.EndLine != 3 {
		t.Fatalf("body range = %#v", body.Position)
	}
	if run.Position.StartLine != 1 || run.Position.EndLine != 3 {
		t.Fatalf("instruction range = %#v", run.Position)
	}
	if spec, ok := body.Data.(heredoc.Spec); !ok || !spec.Expand {
		t.Fatalf("heredoc metadata = %#v", body.Data)
	}

	next, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	if next.Name != "FROM" || next.Position.StartLine != 4 {
		t.Fatalf("next instruction = %#v", next)
	}
}

func TestQuotedAndTabStrippingHeredocs(t *testing.T) {
	source := "RUN cat <<'NOEXPAND' <<-STRIP\n$VALUE\nNOEXPAND\n\tline one\n\tline two\n\tSTRIP\n"
	p, err := dockerfile.NewParser(strings.NewReader(source))
	if err != nil {
		t.Fatal(err)
	}
	run, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Bodies) != 2 {
		t.Fatalf("got %d heredocs, want 2", len(run.Bodies))
	}
	first := run.Bodies[0]
	firstSpec := first.Data.(heredoc.Spec)
	if first.Content != "$VALUE\n" || firstSpec.Expand || firstSpec.StripTabs {
		t.Fatalf("first heredoc = %#v metadata=%#v", first, firstSpec)
	}
	second := run.Bodies[1]
	secondSpec := second.Data.(heredoc.Spec)
	if second.Content != "line one\nline two\n" || !secondSpec.StripTabs || !secondSpec.Expand {
		t.Fatalf("second heredoc = %#v metadata=%#v", second, secondSpec)
	}
}

func TestOnbuildCollectsBodyThroughTheNestedInstructionHandler(t *testing.T) {
	source := "ONBUILD RUN cat <<EOF\nnested body\nEOF\nFROM alpine\n"
	p, err := dockerfile.NewParser(strings.NewReader(source))
	if err != nil {
		t.Fatal(err)
	}
	onbuild, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	if onbuild.Name != "ONBUILD" || len(onbuild.Children) != 1 {
		t.Fatalf("unexpected ONBUILD node: %#v", onbuild)
	}
	child := onbuild.Children[0]
	if child.Name != "RUN" || len(child.Bodies) != 1 || child.Bodies[0].Content != "nested body\n" {
		t.Fatalf("nested RUN did not receive heredoc: %#v", child)
	}
	if onbuild.Position.EndLine != 3 || child.Position.EndLine != 3 {
		t.Fatalf("nested source ranges: parent=%#v child=%#v", onbuild.Position, child.Position)
	}
	next, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	if next.Name != "FROM" || next.Position.StartLine != 4 {
		t.Fatalf("next instruction = %#v", next)
	}
}

func TestJSONRunDoesNotConsumeFollowingLinesAsHeredoc(t *testing.T) {
	p, err := dockerfile.NewParser(strings.NewReader("RUN [\"echo\", \"<<EOF\"]\nFROM alpine\n"))
	if err != nil {
		t.Fatal(err)
	}
	run, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Bodies) != 0 {
		t.Fatalf("JSON RUN unexpectedly has bodies: %#v", run.Bodies)
	}
	next, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	if next.Name != "FROM" {
		t.Fatalf("next instruction = %#v", next)
	}
}

func TestUnterminatedHeredocReportsConsumedSourceRange(t *testing.T) {
	p, err := dockerfile.NewParser(strings.NewReader("RUN cat <<EOF\nbody without terminator\n"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Next()
	if err == nil || !strings.Contains(err.Error(), "unterminated heredoc") || !strings.Contains(err.Error(), "lines 1-2") {
		t.Fatalf("unexpected unterminated heredoc error: %v", err)
	}
}


func TestAddAndCopyCollectHeredocBodies(t *testing.T) {
	source := "COPY <<EOF /app/config\nkey=value\nEOF\n" +
		"ADD <<'SCRIPT' /app/start.sh\necho \\"$HOME\\"\nSCRIPT\n" +
		"FROM alpine\n"
	p, err := dockerfile.NewParser(strings.NewReader(source))
	if err != nil {
		t.Fatal(err)
	}

	copyInstruction, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	if copyInstruction.Name != "COPY" || len(copyInstruction.Bodies) != 1 ||
		copyInstruction.Bodies[0].Content != "key=value\\n" {
		t.Fatalf("COPY heredoc = %#v", copyInstruction)
	}

	addInstruction, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	if addInstruction.Name != "ADD" || len(addInstruction.Bodies) != 1 ||
		addInstruction.Bodies[0].Content != "echo \\"$HOME\\"\\n" {
		t.Fatalf("ADD heredoc = %#v", addInstruction)
	}
	spec := addInstruction.Bodies[0].Data.(heredoc.Spec)
	if spec.Expand {
		t.Fatal("quoted ADD heredoc delimiter should disable expansion")
	}

	next, err := p.Next()
	if err != nil {
		t.Fatal(err)
	}
	if next.Name != "FROM" {
		t.Fatalf("next instruction = %#v", next)
	}
}
