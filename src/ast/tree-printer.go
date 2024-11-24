package ast

import (
	"bytes"
	"fmt"
	"io"
)

const (
	indent = "  "
)

func TreeString(node Node, indent string) string {
	buffer := &bytes.Buffer{}
	_ = PrintTree(buffer, node, indent)
	return buffer.String()
}

func PrintTree(output io.Writer, node Node, indent string) error {
	printer := &treePrinter{
		indent:     indent,
		labelStack: []string{},
		writer:     output,
	}
	node.Walk(printer)
	return printer.err
}

type treePrinter struct {
	indent     string
	labelStack []string
	writer     io.Writer
	err        error
}

func (printer *treePrinter) write(format string, args ...interface{}) {
	if printer.err != nil {
		return
	}

	if len(args) == 0 {
		_, printer.err = printer.writer.Write([]byte(format))
	} else {
		_, printer.err = fmt.Fprintf(printer.writer, format, args...)
	}
}

func (printer *treePrinter) writeLabel() {
	label := ""
	if len(printer.labelStack) > 0 {
		label = printer.labelStack[len(printer.labelStack)-1]
		printer.labelStack = printer.labelStack[:len(printer.labelStack)-1]
	}

	if len(label) > 0 {
		printer.write("\n")
		printer.write(printer.indent)
		printer.write(label)
	} else {
		printer.write(printer.indent)
	}
}

func (printer *treePrinter) endNode() {
	printer.indent = printer.indent[:len(printer.indent)-len(indent)]
	printer.write("\n")
	printer.write(printer.indent)
	printer.write("]")
}

func (printer *treePrinter) push(labels ...string) {
	printer.indent += indent

	for len(labels) > 0 {
		last := labels[len(labels)-1]
		labels = labels[:len(labels)-1]

		printer.labelStack = append(printer.labelStack, last)
	}
}

func (printer *treePrinter) list(
	header string,
	elementType string,
	size int,
	argLabels ...string,
) {
	printer.write(header)
	if size == 0 && len(argLabels) == 0 {
		printer.write("]")
	} else {
		for i := size - 1; i >= 0; i-- {
			printer.labelStack = append(
				printer.labelStack,
				fmt.Sprintf("%s%d=", elementType, i))
		}

		// push in reverse order
		printer.push(argLabels...)
	}
}

func (printer *treePrinter) endList(size int) {
	if size > 0 {
		printer.endNode()
	}
}

func (printer *treePrinter) Enter(n Node) {
	printer.writeLabel()

	switch node := n.(type) {
	case *RegisterDefinition:
		printer.write("[RegisterDefinition: Name=%s Loc=%s", node.Name, node.Loc())
		if node.Type != nil {
			printer.push("Type=")
		} else {
			printer.push()
		}
		printer.write("\n%sDefUses:", printer.indent)
		for ref, _ := range node.DefUses {
			parent := "(ins)"
			if ref.Parent == nil {
				parent = "(phi)"
			}
			printer.write("\n%s  %s: %s", printer.indent, parent, ref.Loc())
		}
	case *RegisterReference:
		printer.write("[RegisterReference: Name=%s Loc=%s", node.Name, node.Loc())
		printer.push()
		label := "(nil)"
		if node.UseDef != nil {
			if node.UseDef.Parent != nil {
				label = "(ins) "
			} else {
				label = "(phi) "
			}
			label += node.UseDef.Loc().String()
		}
		printer.write("\n%sUseDef: %s", printer.indent, label)
	case *GlobalLabelReference:
		printer.write("[GlobalLabelReference: Label=%s]", node.Label)
	case *IntImmediate:
		printer.write("[IntImmediate: Value=%d]", node.Value)
	case *FloatImmediate:
		printer.write("[FloatImmediate: Value=%e]", node.Value)

	case *AssignOperation:
		printer.write("[AssignOperation:")
		printer.push("Dest=", "Src=")
	case *UnaryOperation:
		printer.write("[UnaryOperation: Kind=%s", node.Kind)
		printer.push("Dest=", "Src=")
	case *BinaryOperation:
		printer.write("[BinnaryOperation: Kind=%s", node.Kind)
		printer.push("Dest=", "Src1=", "Src2=")
	case *FuncCall:
		printer.list(
			fmt.Sprintf("[FuncCall: Kind=%s", node.Kind),
			"Argument",
			len(node.Srcs),
			"Dest=",
			"Func=")

	case *Jump:
		printer.write("[Jump: Kind=%s Label=%s]", node.Kind, node.Label)
	case *ConditionalJump:
		printer.write("[ConditionalJump: Kind=%s Label=%s", node.Kind, node.Label)
		printer.push("Src1=", "Src2=")
	case *Terminal:
		printer.write("[Terminal: Kind=%s", node.Kind)
		printer.push("Src=")

	case NumberType:
		printer.write("[NumberType: Kind=%s]", node.Kind)
	case FunctionType:
		printer.list(
			"[FunctionType",
			"Parameter",
			len(node.ParameterTypes),
			"ReturnType=")

	case *FuncDefinition:
		printer.write("[FuncDefinition: Label=%s", node.Label)
		labels := []string{}
		for idx, _ := range node.Parameters {
			labels = append(labels, fmt.Sprintf("Parameter%d=", idx))
		}
		labels = append(labels, "ReturnType=")
		for idx, _ := range node.Blocks {
			labels = append(labels, fmt.Sprintf("Block%d=", idx))
		}
		printer.push(labels...)
	case *Block:
		labels := []string{}
		for i := 0; i < len(node.Phis); i++ {
			labels = append(labels, fmt.Sprintf("Phi%d=", i))
		}
		for i, _ := range node.Instructions {
			labels = append(labels, fmt.Sprintf("Instruction%d=", i))
		}

		printer.write("[Block: Label=%s Loc=%s", node.Label, node.Loc())
		printer.push(labels...)
	case *Phi:
		printer.list("[Phi:", "Src", len(node.Srcs), "Dest=")

	default:
		printer.write("unhandled node: %v", n)
	}
}

func (printer *treePrinter) Exit(n Node) {
	switch node := n.(type) {
	case *RegisterDefinition:
		printer.endNode()
	case *RegisterReference:
		printer.endNode()

	case *AssignOperation:
		printer.endNode()
	case *UnaryOperation:
		printer.endNode()
	case *BinaryOperation:
		printer.endNode()
	case *FuncCall:
		printer.endList(len(node.Srcs))

	case *ConditionalJump:
		printer.endNode()
	case *Terminal:
		printer.endNode()

	case FunctionType:
		printer.endList(len(node.ParameterTypes))

	case *FuncDefinition:
		printer.endNode()
	case *Block:
		printer.endNode()
	case *Phi:
		printer.endNode()
	}
}
