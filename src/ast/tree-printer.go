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
	case *VariableDefinition:
		printer.write("[VariableDefinition: Name=%s Loc=%s", node.Name, node.Loc())
		if node.Type != nil {
			printer.push("Type=")
		} else {
			printer.push()
		}
		printer.write("\n%sDefUses:", printer.indent)
		for ref, _ := range node.DefUses {
			parent := ""
			if ref.ParentInstruction != nil {
				parent = "(ins)"
				_, ok := ref.ParentInstruction.(*Phi)
				if ok {
					parent = "(phi)"
				}
			}
			printer.write("\n%s  %s: %s", printer.indent, parent, ref.Loc())
		}
	case *VariableReference:
		printer.write("[VariableReference: Name=%s Loc=%s", node.Name, node.Loc())
		printer.push()

		if node.UseDef != nil && node.UseDef.Type != nil {
			printer.write("\n%sType: %s", printer.indent, node.UseDef.Type)
		}
		parent := "(nil)"
		if node.UseDef != nil {
			if node.UseDef.ParentInstruction != nil {
				parent = "(ins) "
				_, ok := node.UseDef.ParentInstruction.(*Phi)
				if ok {
					parent = "(phi) "
				}
			}
			parent += node.UseDef.Loc().String()
		}
		printer.write("\n%sUseDef: %s", printer.indent, parent)
	case *GlobalLabelReference:
		printer.write("[GlobalLabelReference: Label=%s]", node.Label)
		if node.Signature != nil {
			printer.write("\n%sType: %s", printer.indent, node.Signature.Type())
		}
	case *IntImmediate:
		sign := ""
		if node.IsNegative {
			sign = "-"
		}
		printer.write("[IntImmediate: Value=%s%d]", sign, node.Value)
	case *FloatImmediate:
		printer.write("[FloatImmediate: Value=%e]", node.Value)

	case *CopyOperation:
		printer.write("[CopyOperation:")
		printer.push("Dest=", "Src=")
	case *UnaryOperation:
		printer.write("[UnaryOperation: Kind=%s", node.Kind)
		printer.push("Dest=", "Src=")
	case *BinaryOperation:
		printer.write("[BinaryOperation: Kind=%s", node.Kind)
		printer.push("Dest=", "Src1=", "Src2=")
	case *FuncCall:
		printer.list(
			fmt.Sprintf("[FuncCall: Kind=%s", node.Kind),
			"Argument",
			len(node.Args),
			"Dest=",
			"Func=")

	case *Jump:
		printer.write("[Jump: Kind=%s Label=%s]", node.Kind, node.Label)
	case *ConditionalJump:
		printer.write("[ConditionalJump: Kind=%s Label=%s", node.Kind, node.Label)
		printer.push("Src1=", "Src2=")
	case *Terminal:
		printer.list(
			fmt.Sprintf("[Terminal: Kind=%s", node.Kind),
			"CalleeSavedSource",
			len(node.CalleeSavedSources),
			"RetVal=")

	case *ErrorType:
		printer.write("[ErrorType]")
	case *PositiveIntLiteralType:
		printer.write("[PositiveIntLiteralType]")
	case *NegativeIntLiteralType:
		printer.write("[NegativeIntLiteralType]")
	case *FloatLiteralType:
		printer.write("[FloatLiteralType]")
	case *SignedIntType:
		printer.write("[SignedIntType: Kind=%s]", node.Kind)
	case *UnsignedIntType:
		printer.write("[UnsignedIntType: Kind=%s]", node.Kind)
	case *FloatType:
		printer.write("[FloatType: Kind=%s]", node.Kind)
	case *FunctionType:
		printer.list(
			fmt.Sprintf(
				"[FunctionType CallConventionName=%s",
				node.CallConventionName),
			"Parameter",
			len(node.ParameterTypes),
			"ReturnType=")

	case *FunctionDefinition:
		printer.write(
			"[FunctionDefinition: Label=%s CallConventionName=%s",
			node.Label,
			node.CallConventionName)
		labels := []string{}
		for idx, _ := range node.Parameters {
			labels = append(labels, fmt.Sprintf("Parameter%d=", idx))
		}
		for idx, _ := range node.PseudoParameters {
			labels = append(labels, fmt.Sprintf("PseudoParameter%d=", idx))
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
	case *VariableDefinition:
		printer.endNode()
	case *VariableReference:
		printer.endNode()

	case *CopyOperation:
		printer.endNode()
	case *UnaryOperation:
		printer.endNode()
	case *BinaryOperation:
		printer.endNode()
	case *FuncCall:
		printer.endList(len(node.Args))

	case *ConditionalJump:
		printer.endNode()
	case *Terminal:
		printer.endNode()

	case FunctionType:
		printer.endList(len(node.ParameterTypes))

	case *FunctionDefinition:
		printer.endNode()
	case *Block:
		printer.endNode()
	case *Phi:
		printer.endNode()
	}
}
