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
		if node.Type != nil {
			printer.write("[RegisterDefinition: Name=%s", node.Name)
			printer.push("Type=")
		} else {
			printer.write("[RegisterDefinition: Name=%s]", node.Name)
		}
	case *RegisterReference:
		printer.write("[RegisterReference: Name=%s]", node.Name)
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
	case *Terminate:
		printer.list(
			fmt.Sprintf("[Terminate: Kind=%s", node.Kind),
			"Argument",
			len(node.Srcs))

	case NumberType:
		printer.write("[NumberType: Kind=%s]", node.Kind)
	case FunctionType:
		printer.list(
			"[FunctionType",
			"Parameter",
			len(node.ParameterTypes),
			"ReturnType=")

	case *Declaration:
		printer.write("[Declaration: Kind=%s Label=%s", node.Kind, node.Label)
		printer.push("Type=")
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
		parents := []string{}
		for _, parent := range node.Parents {
			parents = append(parents, parent.Label)
		}

		children := []string{}
		for _, child := range node.Children {
			children = append(children, child.Label)
		}

		printer.list(
			fmt.Sprintf(
				"[Block: Label=%s Parents: %v Children: %v",
				node.Label,
				parents,
				children),
			"Instruction",
			len(node.Instructions))

	default:
		printer.write("unhandled node: %v", n)
	}
}

func (printer *treePrinter) Exit(n Node) {
	switch node := n.(type) {
	case *RegisterDefinition:
		if node.Type != nil {
			printer.endNode()
		}

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
	case *Terminate:
		printer.endList(len(node.Srcs))

	case FunctionType:
		printer.endList(len(node.ParameterTypes))

	case *Declaration:
		printer.endNode()
	case *FuncDefinition:
		printer.endNode()
	case *Block:
		printer.endList(len(node.Instructions))
	}
}
