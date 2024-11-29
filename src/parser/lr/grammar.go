// Auto-generated from source: grammar.lr

package lr

import (
	fmt "fmt"
	ast "github.com/pattyshack/chickadee/ast"
	parseutil "github.com/pattyshack/gt/parseutil"
	io "io"
)

type SymbolId int

const (
	IntegerLiteralToken = SymbolId(256)
	FloatLiteralToken   = SymbolId(257)
	StringLiteralToken  = SymbolId(258)
	IdentifierToken     = SymbolId(259)
	LparenToken         = SymbolId(260)
	RparenToken         = SymbolId(261)
	LbraceToken         = SymbolId(262)
	RbraceToken         = SymbolId(263)
	CommaToken          = SymbolId(264)
	ColonToken          = SymbolId(265)
	AtToken             = SymbolId(266)
	PercentToken        = SymbolId(267)
	EqualToken          = SymbolId(268)
	DefineToken         = SymbolId(269)
	FuncToken           = SymbolId(270)
)

type DefinitionReducer interface {
	// 24:2: definition -> func: ...
	FuncToDefinition(Define_ *TokenValue, Func_ *TokenValue, GlobalLabel_ *ast.GlobalLabelReference, Lparen_ *TokenValue, Parameters_ []*ast.VariableDefinition, Rparen_ *TokenValue, Type_ ast.Type, Lbrace_ *TokenValue) (ast.Line, error)
}

type RbraceReducer interface {
	// 26:16: rbrace -> ...
	ToRbrace(Rbrace_ *TokenValue) (ast.Line, error)
}

type GlobalLabelReducer interface {
	// 32:38: global_label -> ...
	ToGlobalLabel(At_ *TokenValue, Identifier_ *TokenValue) (*ast.GlobalLabelReference, error)
}

type LocalLabelReducer interface {
	// 34:27: local_label -> ...
	ToLocalLabel(Colon_ *TokenValue, Identifier_ *TokenValue) (ParsedLocalLabel, error)
}

type VariableReferenceReducer interface {
	// 36:41: variable_reference -> ...
	ToVariableReference(Percent_ *TokenValue, Identifier_ *TokenValue) (*ast.VariableReference, error)
}

type IdentifierReducer interface {

	// 40:2: identifier -> string: ...
	StringToIdentifier(StringLiteral_ *TokenValue) (*TokenValue, error)
}

type IntImmediateReducer interface {
	// 46:26: int_immediate -> ...
	ToIntImmediate(IntegerLiteral_ *TokenValue) (ast.Value, error)
}

type FloatImmediateReducer interface {
	// 48:28: float_immediate -> ...
	ToFloatImmediate(FloatLiteral_ *TokenValue) (ast.Value, error)
}

type TypedVariableDefinitionReducer interface {
	// 50:49: typed_variable_definition -> ...
	ToTypedVariableDefinition(VariableReference_ *ast.VariableReference, Type_ ast.Type) (*ast.VariableDefinition, error)
}

type VariableDefinitionReducer interface {

	// 54:2: variable_definition -> inferred: ...
	InferredToVariableDefinition(VariableReference_ *ast.VariableReference) (*ast.VariableDefinition, error)
}

type ParametersReducer interface {

	// 67:2: parameters -> improper: ...
	ImproperToParameters(ProperParameters_ []*ast.VariableDefinition, Comma_ *TokenValue) ([]*ast.VariableDefinition, error)

	// 68:2: parameters -> nil: ...
	NilToParameters() ([]*ast.VariableDefinition, error)
}

type ProperParametersReducer interface {
	// 71:2: proper_parameters -> add: ...
	AddToProperParameters(ProperParameters_ []*ast.VariableDefinition, Comma_ *TokenValue, TypedVariableDefinition_ *ast.VariableDefinition) ([]*ast.VariableDefinition, error)

	// 72:2: proper_parameters -> new: ...
	NewToProperParameters(TypedVariableDefinition_ *ast.VariableDefinition) ([]*ast.VariableDefinition, error)
}

type ArgumentsReducer interface {

	// 76:2: arguments -> improper: ...
	ImproperToArguments(ProperArguments_ []ast.Value, Comma_ *TokenValue) ([]ast.Value, error)

	// 77:2: arguments -> nil: ...
	NilToArguments() ([]ast.Value, error)
}

type ProperArgumentsReducer interface {
	// 80:2: proper_arguments -> add: ...
	AddToProperArguments(ProperArguments_ []ast.Value, Comma_ *TokenValue, Value_ ast.Value) ([]ast.Value, error)

	// 81:2: proper_arguments -> new: ...
	NewToProperArguments(Value_ ast.Value) ([]ast.Value, error)
}

type TypesReducer interface {

	// 85:2: types -> improper: ...
	ImproperToTypes(ProperTypes_ []ast.Type, Comma_ *TokenValue) ([]ast.Type, error)

	// 86:2: types -> nil: ...
	NilToTypes() ([]ast.Type, error)
}

type ProperTypesReducer interface {
	// 89:2: proper_types -> add: ...
	AddToProperTypes(ProperTypes_ []ast.Type, Comma_ *TokenValue, Type_ ast.Type) ([]ast.Type, error)

	// 90:2: proper_types -> new: ...
	NewToProperTypes(Type_ ast.Type) ([]ast.Type, error)
}

type OperationInstructionReducer interface {
	// 97:2: operation_instruction -> assign: ...
	AssignToOperationInstruction(VariableDefinition_ *ast.VariableDefinition, Equal_ *TokenValue, Value_ ast.Value) (ast.Instruction, error)

	// 98:2: operation_instruction -> unary: ...
	UnaryToOperationInstruction(VariableDefinition_ *ast.VariableDefinition, Equal_ *TokenValue, Identifier_ *TokenValue, Value_ ast.Value) (ast.Instruction, error)

	// 99:2: operation_instruction -> binary: ...
	BinaryToOperationInstruction(VariableDefinition_ *ast.VariableDefinition, Equal_ *TokenValue, Identifier_ *TokenValue, Value_ ast.Value, Comma_ *TokenValue, Value_2 ast.Value) (ast.Instruction, error)

	// 100:2: operation_instruction -> call: ...
	CallToOperationInstruction(VariableDefinition_ *ast.VariableDefinition, Equal_ *TokenValue, Identifier_ *TokenValue, Value_ ast.Value, Lparen_ *TokenValue, Arguments_ []ast.Value, Rparen_ *TokenValue) (ast.Instruction, error)
}

type ControlFlowInstructionReducer interface {
	// 103:2: control_flow_instruction -> unconditional: ...
	UnconditionalToControlFlowInstruction(Identifier_ *TokenValue, LocalLabel_ ParsedLocalLabel) (ast.Instruction, error)

	// 104:2: control_flow_instruction -> conditional: ...
	ConditionalToControlFlowInstruction(Identifier_ *TokenValue, LocalLabel_ ParsedLocalLabel, Comma_ *TokenValue, Value_ ast.Value, Comma_2 *TokenValue, Value_2 ast.Value) (ast.Instruction, error)

	// 105:2: control_flow_instruction -> terminal: ...
	TerminalToControlFlowInstruction(Identifier_ *TokenValue, Value_ ast.Value) (ast.Instruction, error)
}

type NumberTypeReducer interface {
	// 116:21: number_type -> ...
	ToNumberType(Identifier_ *TokenValue) (ast.Type, error)
}

type FuncTypeReducer interface {
	// 118:19: func_type -> ...
	ToFuncType(Func_ *TokenValue, Lparen_ *TokenValue, Types_ []ast.Type, Rparen_ *TokenValue, Type_ ast.Type) (ast.Type, error)
}

type Reducer interface {
	DefinitionReducer
	RbraceReducer
	GlobalLabelReducer
	LocalLabelReducer
	VariableReferenceReducer
	IdentifierReducer
	IntImmediateReducer
	FloatImmediateReducer
	TypedVariableDefinitionReducer
	VariableDefinitionReducer
	ParametersReducer
	ProperParametersReducer
	ArgumentsReducer
	ProperArgumentsReducer
	TypesReducer
	ProperTypesReducer
	OperationInstructionReducer
	ControlFlowInstructionReducer
	NumberTypeReducer
	FuncTypeReducer
}

type ParseErrorHandler interface {
	Error(nextToken parseutil.Token[SymbolId], parseStack _Stack) error
}

type DefaultParseErrorHandler struct{}

func (DefaultParseErrorHandler) Error(nextToken parseutil.Token[SymbolId], stack _Stack) error {
	return parseutil.NewLocationError(
		nextToken.Loc(),
		"syntax error: unexpected symbol %s. expecting %v",
		nextToken.Id(),
		ExpectedTerminals(stack[len(stack)-1].StateId))
}

func ExpectedTerminals(id _StateId) []SymbolId {
	switch id {
	case _State1:
		return []SymbolId{IdentifierToken, RbraceToken, ColonToken, PercentToken, DefineToken}
	case _State2:
		return []SymbolId{_EndMarker}
	case _State3:
		return []SymbolId{StringLiteralToken, IdentifierToken}
	case _State4:
		return []SymbolId{FuncToken}
	case _State5:
		return []SymbolId{IntegerLiteralToken, FloatLiteralToken, ColonToken, AtToken, PercentToken}
	case _State6:
		return []SymbolId{StringLiteralToken, IdentifierToken}
	case _State7:
		return []SymbolId{EqualToken}
	case _State9:
		return []SymbolId{AtToken}
	case _State10:
		return []SymbolId{StringLiteralToken, IdentifierToken}
	case _State12:
		return []SymbolId{IntegerLiteralToken, FloatLiteralToken, IdentifierToken, AtToken, PercentToken}
	case _State13:
		return []SymbolId{LparenToken}
	case _State14:
		return []SymbolId{LparenToken}
	case _State15:
		return []SymbolId{IntegerLiteralToken, FloatLiteralToken, AtToken, PercentToken}
	case _State16:
		return []SymbolId{IntegerLiteralToken, FloatLiteralToken, AtToken, PercentToken}
	case _State19:
		return []SymbolId{CommaToken}
	case _State22:
		return []SymbolId{RparenToken}
	case _State23:
		return []SymbolId{RparenToken}
	case _State25:
		return []SymbolId{IdentifierToken, FuncToken}
	case _State26:
		return []SymbolId{IntegerLiteralToken, FloatLiteralToken, AtToken, PercentToken}
	case _State27:
		return []SymbolId{IntegerLiteralToken, FloatLiteralToken, AtToken, PercentToken}
	case _State30:
		return []SymbolId{IdentifierToken, FuncToken}
	case _State31:
		return []SymbolId{IdentifierToken, FuncToken}
	case _State33:
		return []SymbolId{RparenToken}
	case _State35:
		return []SymbolId{LbraceToken}
	}

	return nil
}

func Parse(lexer parseutil.Lexer[parseutil.Token[SymbolId]], reducer Reducer) (ast.Line, error) {

	return ParseWithCustomErrorHandler(
		lexer,
		reducer,
		DefaultParseErrorHandler{})
}

func ParseWithCustomErrorHandler(
	lexer parseutil.Lexer[parseutil.Token[SymbolId]],
	reducer Reducer,
	errHandler ParseErrorHandler,
) (
	ast.Line,
	error,
) {
	item, err := _Parse(lexer, reducer, errHandler, _State1)
	if err != nil {
		var errRetVal ast.Line
		return errRetVal, err
	}
	return item.Line, nil
}

// ================================================================
// Parser internal implementation
// User should normally avoid directly accessing the following code
// ================================================================

func _Parse(
	lexer parseutil.Lexer[parseutil.Token[SymbolId]],
	reducer Reducer,
	errHandler ParseErrorHandler,
	startState _StateId,
) (
	*_StackItem,
	error,
) {
	stateStack := _Stack{
		// Note: we don't have to populate the start symbol since its value
		// is never accessed.
		&_StackItem{startState, nil},
	}

	symbolStack := &_PseudoSymbolStack{lexer: lexer}

	for {
		nextSymbol, err := symbolStack.Top()
		if err != nil {
			return nil, err
		}

		action, ok := _ActionTable.Get(
			stateStack[len(stateStack)-1].StateId,
			nextSymbol.Id())
		if !ok {
			return nil, errHandler.Error(nextSymbol, stateStack)
		}

		if action.ActionType == _ShiftAction {
			stateStack = append(stateStack, action.ShiftItem(nextSymbol))

			_, err = symbolStack.Pop()
			if err != nil {
				return nil, err
			}
		} else if action.ActionType == _ReduceAction {
			var reduceSymbol *Symbol
			stateStack, reduceSymbol, err = action.ReduceSymbol(
				reducer,
				stateStack)
			if err != nil {
				return nil, err
			}

			symbolStack.Push(reduceSymbol)
		} else if action.ActionType == _ShiftAndReduceAction {
			stateStack = append(stateStack, action.ShiftItem(nextSymbol))

			_, err = symbolStack.Pop()
			if err != nil {
				return nil, err
			}

			var reduceSymbol *Symbol
			stateStack, reduceSymbol, err = action.ReduceSymbol(
				reducer,
				stateStack)
			if err != nil {
				return nil, err
			}

			symbolStack.Push(reduceSymbol)
		} else if action.ActionType == _AcceptAction {
			if len(stateStack) != 2 {
				panic("This should never happen")
			}
			return stateStack[1], nil
		} else {
			panic("Unknown action type: " + action.ActionType.String())
		}
	}
}

func (i SymbolId) String() string {
	switch i {
	case _EndMarker:
		return "$"
	case _WildcardMarker:
		return "*"
	case IntegerLiteralToken:
		return "INTEGER_LITERAL"
	case FloatLiteralToken:
		return "FLOAT_LITERAL"
	case StringLiteralToken:
		return "STRING_LITERAL"
	case IdentifierToken:
		return "IDENTIFIER"
	case LparenToken:
		return "LPAREN"
	case RparenToken:
		return "RPAREN"
	case LbraceToken:
		return "LBRACE"
	case RbraceToken:
		return "RBRACE"
	case CommaToken:
		return "COMMA"
	case ColonToken:
		return "COLON"
	case AtToken:
		return "AT"
	case PercentToken:
		return "PERCENT"
	case EqualToken:
		return "EQUAL"
	case DefineToken:
		return "DEFINE"
	case FuncToken:
		return "FUNC"
	case LineType:
		return "line"
	case DefinitionType:
		return "definition"
	case RbraceType:
		return "rbrace"
	case GlobalLabelType:
		return "global_label"
	case LocalLabelType:
		return "local_label"
	case VariableReferenceType:
		return "variable_reference"
	case IdentifierType:
		return "identifier"
	case ImmediateType:
		return "immediate"
	case IntImmediateType:
		return "int_immediate"
	case FloatImmediateType:
		return "float_immediate"
	case TypedVariableDefinitionType:
		return "typed_variable_definition"
	case VariableDefinitionType:
		return "variable_definition"
	case ValueType:
		return "value"
	case ParametersType:
		return "parameters"
	case ProperParametersType:
		return "proper_parameters"
	case ArgumentsType:
		return "arguments"
	case ProperArgumentsType:
		return "proper_arguments"
	case TypesType:
		return "types"
	case ProperTypesType:
		return "proper_types"
	case OperationInstructionType:
		return "operation_instruction"
	case ControlFlowInstructionType:
		return "control_flow_instruction"
	case TypeType:
		return "type"
	case NumberTypeType:
		return "number_type"
	case FuncTypeType:
		return "func_type"
	default:
		return fmt.Sprintf("?unknown symbol %d?", int(i))
	}
}

const (
	_EndMarker      = SymbolId(0)
	_WildcardMarker = SymbolId(-1)

	LineType                    = SymbolId(271)
	DefinitionType              = SymbolId(272)
	RbraceType                  = SymbolId(273)
	GlobalLabelType             = SymbolId(274)
	LocalLabelType              = SymbolId(275)
	VariableReferenceType       = SymbolId(276)
	IdentifierType              = SymbolId(277)
	ImmediateType               = SymbolId(278)
	IntImmediateType            = SymbolId(279)
	FloatImmediateType          = SymbolId(280)
	TypedVariableDefinitionType = SymbolId(281)
	VariableDefinitionType      = SymbolId(282)
	ValueType                   = SymbolId(283)
	ParametersType              = SymbolId(284)
	ProperParametersType        = SymbolId(285)
	ArgumentsType               = SymbolId(286)
	ProperArgumentsType         = SymbolId(287)
	TypesType                   = SymbolId(288)
	ProperTypesType             = SymbolId(289)
	OperationInstructionType    = SymbolId(290)
	ControlFlowInstructionType  = SymbolId(291)
	TypeType                    = SymbolId(292)
	NumberTypeType              = SymbolId(293)
	FuncTypeType                = SymbolId(294)
)

type _ActionType int

const (
	// NOTE: error action is implicit
	_ShiftAction          = _ActionType(0)
	_ReduceAction         = _ActionType(1)
	_ShiftAndReduceAction = _ActionType(2)
	_AcceptAction         = _ActionType(3)
)

func (i _ActionType) String() string {
	switch i {
	case _ShiftAction:
		return "shift"
	case _ReduceAction:
		return "reduce"
	case _ShiftAndReduceAction:
		return "shift-and-reduce"
	case _AcceptAction:
		return "accept"
	default:
		return fmt.Sprintf("?Unknown action %d?", int(i))
	}
}

type _ReduceType int

const (
	_ReduceDefinitionToLine                            = _ReduceType(1)
	_ReduceRbraceToLine                                = _ReduceType(2)
	_ReduceLocalLabelToLine                            = _ReduceType(3)
	_ReduceOperationInstructionToLine                  = _ReduceType(4)
	_ReduceControlFlowInstructionToLine                = _ReduceType(5)
	_ReduceFuncToDefinition                            = _ReduceType(6)
	_ReduceToRbrace                                    = _ReduceType(7)
	_ReduceToGlobalLabel                               = _ReduceType(8)
	_ReduceToLocalLabel                                = _ReduceType(9)
	_ReduceToVariableReference                         = _ReduceType(10)
	_ReduceIdentifierToIdentifier                      = _ReduceType(11)
	_ReduceStringToIdentifier                          = _ReduceType(12)
	_ReduceIntImmediateToImmediate                     = _ReduceType(13)
	_ReduceFloatImmediateToImmediate                   = _ReduceType(14)
	_ReduceToIntImmediate                              = _ReduceType(15)
	_ReduceToFloatImmediate                            = _ReduceType(16)
	_ReduceToTypedVariableDefinition                   = _ReduceType(17)
	_ReduceTypedVariableDefinitionToVariableDefinition = _ReduceType(18)
	_ReduceInferredToVariableDefinition                = _ReduceType(19)
	_ReduceVariableReferenceToValue                    = _ReduceType(20)
	_ReduceGlobalLabelToValue                          = _ReduceType(21)
	_ReduceImmediateToValue                            = _ReduceType(22)
	_ReduceProperParametersToParameters                = _ReduceType(23)
	_ReduceImproperToParameters                        = _ReduceType(24)
	_ReduceNilToParameters                             = _ReduceType(25)
	_ReduceAddToProperParameters                       = _ReduceType(26)
	_ReduceNewToProperParameters                       = _ReduceType(27)
	_ReduceProperArgumentsToArguments                  = _ReduceType(28)
	_ReduceImproperToArguments                         = _ReduceType(29)
	_ReduceNilToArguments                              = _ReduceType(30)
	_ReduceAddToProperArguments                        = _ReduceType(31)
	_ReduceNewToProperArguments                        = _ReduceType(32)
	_ReduceProperTypesToTypes                          = _ReduceType(33)
	_ReduceImproperToTypes                             = _ReduceType(34)
	_ReduceNilToTypes                                  = _ReduceType(35)
	_ReduceAddToProperTypes                            = _ReduceType(36)
	_ReduceNewToProperTypes                            = _ReduceType(37)
	_ReduceAssignToOperationInstruction                = _ReduceType(38)
	_ReduceUnaryToOperationInstruction                 = _ReduceType(39)
	_ReduceBinaryToOperationInstruction                = _ReduceType(40)
	_ReduceCallToOperationInstruction                  = _ReduceType(41)
	_ReduceUnconditionalToControlFlowInstruction       = _ReduceType(42)
	_ReduceConditionalToControlFlowInstruction         = _ReduceType(43)
	_ReduceTerminalToControlFlowInstruction            = _ReduceType(44)
	_ReduceNumberTypeToType                            = _ReduceType(45)
	_ReduceFuncTypeToType                              = _ReduceType(46)
	_ReduceToNumberType                                = _ReduceType(47)
	_ReduceToFuncType                                  = _ReduceType(48)
)

func (i _ReduceType) String() string {
	switch i {
	case _ReduceDefinitionToLine:
		return "DefinitionToLine"
	case _ReduceRbraceToLine:
		return "RbraceToLine"
	case _ReduceLocalLabelToLine:
		return "LocalLabelToLine"
	case _ReduceOperationInstructionToLine:
		return "OperationInstructionToLine"
	case _ReduceControlFlowInstructionToLine:
		return "ControlFlowInstructionToLine"
	case _ReduceFuncToDefinition:
		return "FuncToDefinition"
	case _ReduceToRbrace:
		return "ToRbrace"
	case _ReduceToGlobalLabel:
		return "ToGlobalLabel"
	case _ReduceToLocalLabel:
		return "ToLocalLabel"
	case _ReduceToVariableReference:
		return "ToVariableReference"
	case _ReduceIdentifierToIdentifier:
		return "IdentifierToIdentifier"
	case _ReduceStringToIdentifier:
		return "StringToIdentifier"
	case _ReduceIntImmediateToImmediate:
		return "IntImmediateToImmediate"
	case _ReduceFloatImmediateToImmediate:
		return "FloatImmediateToImmediate"
	case _ReduceToIntImmediate:
		return "ToIntImmediate"
	case _ReduceToFloatImmediate:
		return "ToFloatImmediate"
	case _ReduceToTypedVariableDefinition:
		return "ToTypedVariableDefinition"
	case _ReduceTypedVariableDefinitionToVariableDefinition:
		return "TypedVariableDefinitionToVariableDefinition"
	case _ReduceInferredToVariableDefinition:
		return "InferredToVariableDefinition"
	case _ReduceVariableReferenceToValue:
		return "VariableReferenceToValue"
	case _ReduceGlobalLabelToValue:
		return "GlobalLabelToValue"
	case _ReduceImmediateToValue:
		return "ImmediateToValue"
	case _ReduceProperParametersToParameters:
		return "ProperParametersToParameters"
	case _ReduceImproperToParameters:
		return "ImproperToParameters"
	case _ReduceNilToParameters:
		return "NilToParameters"
	case _ReduceAddToProperParameters:
		return "AddToProperParameters"
	case _ReduceNewToProperParameters:
		return "NewToProperParameters"
	case _ReduceProperArgumentsToArguments:
		return "ProperArgumentsToArguments"
	case _ReduceImproperToArguments:
		return "ImproperToArguments"
	case _ReduceNilToArguments:
		return "NilToArguments"
	case _ReduceAddToProperArguments:
		return "AddToProperArguments"
	case _ReduceNewToProperArguments:
		return "NewToProperArguments"
	case _ReduceProperTypesToTypes:
		return "ProperTypesToTypes"
	case _ReduceImproperToTypes:
		return "ImproperToTypes"
	case _ReduceNilToTypes:
		return "NilToTypes"
	case _ReduceAddToProperTypes:
		return "AddToProperTypes"
	case _ReduceNewToProperTypes:
		return "NewToProperTypes"
	case _ReduceAssignToOperationInstruction:
		return "AssignToOperationInstruction"
	case _ReduceUnaryToOperationInstruction:
		return "UnaryToOperationInstruction"
	case _ReduceBinaryToOperationInstruction:
		return "BinaryToOperationInstruction"
	case _ReduceCallToOperationInstruction:
		return "CallToOperationInstruction"
	case _ReduceUnconditionalToControlFlowInstruction:
		return "UnconditionalToControlFlowInstruction"
	case _ReduceConditionalToControlFlowInstruction:
		return "ConditionalToControlFlowInstruction"
	case _ReduceTerminalToControlFlowInstruction:
		return "TerminalToControlFlowInstruction"
	case _ReduceNumberTypeToType:
		return "NumberTypeToType"
	case _ReduceFuncTypeToType:
		return "FuncTypeToType"
	case _ReduceToNumberType:
		return "ToNumberType"
	case _ReduceToFuncType:
		return "ToFuncType"
	default:
		return fmt.Sprintf("?unknown reduce type %d?", int(i))
	}
}

type _StateId int

func (id _StateId) String() string {
	return fmt.Sprintf("State %d", int(id))
}

const (
	_State1  = _StateId(1)
	_State2  = _StateId(2)
	_State3  = _StateId(3)
	_State4  = _StateId(4)
	_State5  = _StateId(5)
	_State6  = _StateId(6)
	_State7  = _StateId(7)
	_State8  = _StateId(8)
	_State9  = _StateId(9)
	_State10 = _StateId(10)
	_State11 = _StateId(11)
	_State12 = _StateId(12)
	_State13 = _StateId(13)
	_State14 = _StateId(14)
	_State15 = _StateId(15)
	_State16 = _StateId(16)
	_State17 = _StateId(17)
	_State18 = _StateId(18)
	_State19 = _StateId(19)
	_State20 = _StateId(20)
	_State21 = _StateId(21)
	_State22 = _StateId(22)
	_State23 = _StateId(23)
	_State24 = _StateId(24)
	_State25 = _StateId(25)
	_State26 = _StateId(26)
	_State27 = _StateId(27)
	_State28 = _StateId(28)
	_State29 = _StateId(29)
	_State30 = _StateId(30)
	_State31 = _StateId(31)
	_State32 = _StateId(32)
	_State33 = _StateId(33)
	_State34 = _StateId(34)
	_State35 = _StateId(35)
	_State36 = _StateId(36)
)

type Symbol struct {
	SymbolId_ SymbolId

	Generic_ parseutil.TokenValue[SymbolId]

	Arguments            []ast.Value
	Count                *TokenCount
	GlobalLabelReference *ast.GlobalLabelReference
	Instruction          ast.Instruction
	Line                 ast.Line
	LocalLabel           ParsedLocalLabel
	OpValue              ast.Value
	Parameters           []*ast.VariableDefinition
	Type                 ast.Type
	Types                []ast.Type
	Value                *TokenValue
	VariableDefinition   *ast.VariableDefinition
	VariableReference    *ast.VariableReference
}

func NewSymbol(token parseutil.Token[SymbolId]) (*Symbol, error) {
	symbol, ok := token.(*Symbol)
	if ok {
		return symbol, nil
	}

	symbol = &Symbol{SymbolId_: token.Id()}
	switch token.Id() {
	case _EndMarker:
		val, ok := token.(parseutil.TokenValue[SymbolId])
		if !ok {
			return nil, parseutil.NewLocationError(
				token.Loc(),
				"invalid value type for token %s. "+
					"expecting parseutil.TokenValue[SymbolId]",
				token.Id())
		}
		symbol.Generic_ = val
	case IntegerLiteralToken, FloatLiteralToken, StringLiteralToken, IdentifierToken, LparenToken, RparenToken, LbraceToken, RbraceToken, CommaToken, ColonToken, AtToken, PercentToken, EqualToken, DefineToken, FuncToken:
		val, ok := token.(*TokenValue)
		if !ok {
			return nil, parseutil.NewLocationError(
				token.Loc(),
				"invalid value type for token %s. "+
					"expecting *TokenValue",
				token.Id())
		}
		symbol.Value = val
	default:
		return nil, parseutil.NewLocationError(
			token.Loc(),
			"unexpected token type: %s",
			token.Id())
	}
	return symbol, nil
}

func (s *Symbol) Id() SymbolId {
	return s.SymbolId_
}

func (s *Symbol) StartEnd() parseutil.StartEndPos {
	type locator interface{ StartEnd() parseutil.StartEndPos }
	switch s.SymbolId_ {
	case ArgumentsType, ProperArgumentsType:
		loc, ok := interface{}(s.Arguments).(locator)
		if ok {
			return loc.StartEnd()
		}
	case GlobalLabelType:
		loc, ok := interface{}(s.GlobalLabelReference).(locator)
		if ok {
			return loc.StartEnd()
		}
	case OperationInstructionType, ControlFlowInstructionType:
		loc, ok := interface{}(s.Instruction).(locator)
		if ok {
			return loc.StartEnd()
		}
	case LineType, DefinitionType, RbraceType:
		loc, ok := interface{}(s.Line).(locator)
		if ok {
			return loc.StartEnd()
		}
	case LocalLabelType:
		loc, ok := interface{}(s.LocalLabel).(locator)
		if ok {
			return loc.StartEnd()
		}
	case ImmediateType, IntImmediateType, FloatImmediateType, ValueType:
		loc, ok := interface{}(s.OpValue).(locator)
		if ok {
			return loc.StartEnd()
		}
	case ParametersType, ProperParametersType:
		loc, ok := interface{}(s.Parameters).(locator)
		if ok {
			return loc.StartEnd()
		}
	case TypeType, NumberTypeType, FuncTypeType:
		loc, ok := interface{}(s.Type).(locator)
		if ok {
			return loc.StartEnd()
		}
	case TypesType, ProperTypesType:
		loc, ok := interface{}(s.Types).(locator)
		if ok {
			return loc.StartEnd()
		}
	case IntegerLiteralToken, FloatLiteralToken, StringLiteralToken, IdentifierToken, LparenToken, RparenToken, LbraceToken, RbraceToken, CommaToken, ColonToken, AtToken, PercentToken, EqualToken, DefineToken, FuncToken, IdentifierType:
		loc, ok := interface{}(s.Value).(locator)
		if ok {
			return loc.StartEnd()
		}
	case TypedVariableDefinitionType, VariableDefinitionType:
		loc, ok := interface{}(s.VariableDefinition).(locator)
		if ok {
			return loc.StartEnd()
		}
	case VariableReferenceType:
		loc, ok := interface{}(s.VariableReference).(locator)
		if ok {
			return loc.StartEnd()
		}
	}
	return s.Generic_.StartEnd()
}

func (s *Symbol) Loc() parseutil.Location {
	type locator interface{ Loc() parseutil.Location }
	switch s.SymbolId_ {
	case ArgumentsType, ProperArgumentsType:
		loc, ok := interface{}(s.Arguments).(locator)
		if ok {
			return loc.Loc()
		}
	case GlobalLabelType:
		loc, ok := interface{}(s.GlobalLabelReference).(locator)
		if ok {
			return loc.Loc()
		}
	case OperationInstructionType, ControlFlowInstructionType:
		loc, ok := interface{}(s.Instruction).(locator)
		if ok {
			return loc.Loc()
		}
	case LineType, DefinitionType, RbraceType:
		loc, ok := interface{}(s.Line).(locator)
		if ok {
			return loc.Loc()
		}
	case LocalLabelType:
		loc, ok := interface{}(s.LocalLabel).(locator)
		if ok {
			return loc.Loc()
		}
	case ImmediateType, IntImmediateType, FloatImmediateType, ValueType:
		loc, ok := interface{}(s.OpValue).(locator)
		if ok {
			return loc.Loc()
		}
	case ParametersType, ProperParametersType:
		loc, ok := interface{}(s.Parameters).(locator)
		if ok {
			return loc.Loc()
		}
	case TypeType, NumberTypeType, FuncTypeType:
		loc, ok := interface{}(s.Type).(locator)
		if ok {
			return loc.Loc()
		}
	case TypesType, ProperTypesType:
		loc, ok := interface{}(s.Types).(locator)
		if ok {
			return loc.Loc()
		}
	case IntegerLiteralToken, FloatLiteralToken, StringLiteralToken, IdentifierToken, LparenToken, RparenToken, LbraceToken, RbraceToken, CommaToken, ColonToken, AtToken, PercentToken, EqualToken, DefineToken, FuncToken, IdentifierType:
		loc, ok := interface{}(s.Value).(locator)
		if ok {
			return loc.Loc()
		}
	case TypedVariableDefinitionType, VariableDefinitionType:
		loc, ok := interface{}(s.VariableDefinition).(locator)
		if ok {
			return loc.Loc()
		}
	case VariableReferenceType:
		loc, ok := interface{}(s.VariableReference).(locator)
		if ok {
			return loc.Loc()
		}
	}
	return s.Generic_.Loc()
}

func (s *Symbol) End() parseutil.Location {
	type locator interface{ End() parseutil.Location }
	switch s.SymbolId_ {
	case ArgumentsType, ProperArgumentsType:
		loc, ok := interface{}(s.Arguments).(locator)
		if ok {
			return loc.End()
		}
	case GlobalLabelType:
		loc, ok := interface{}(s.GlobalLabelReference).(locator)
		if ok {
			return loc.End()
		}
	case OperationInstructionType, ControlFlowInstructionType:
		loc, ok := interface{}(s.Instruction).(locator)
		if ok {
			return loc.End()
		}
	case LineType, DefinitionType, RbraceType:
		loc, ok := interface{}(s.Line).(locator)
		if ok {
			return loc.End()
		}
	case LocalLabelType:
		loc, ok := interface{}(s.LocalLabel).(locator)
		if ok {
			return loc.End()
		}
	case ImmediateType, IntImmediateType, FloatImmediateType, ValueType:
		loc, ok := interface{}(s.OpValue).(locator)
		if ok {
			return loc.End()
		}
	case ParametersType, ProperParametersType:
		loc, ok := interface{}(s.Parameters).(locator)
		if ok {
			return loc.End()
		}
	case TypeType, NumberTypeType, FuncTypeType:
		loc, ok := interface{}(s.Type).(locator)
		if ok {
			return loc.End()
		}
	case TypesType, ProperTypesType:
		loc, ok := interface{}(s.Types).(locator)
		if ok {
			return loc.End()
		}
	case IntegerLiteralToken, FloatLiteralToken, StringLiteralToken, IdentifierToken, LparenToken, RparenToken, LbraceToken, RbraceToken, CommaToken, ColonToken, AtToken, PercentToken, EqualToken, DefineToken, FuncToken, IdentifierType:
		loc, ok := interface{}(s.Value).(locator)
		if ok {
			return loc.End()
		}
	case TypedVariableDefinitionType, VariableDefinitionType:
		loc, ok := interface{}(s.VariableDefinition).(locator)
		if ok {
			return loc.End()
		}
	case VariableReferenceType:
		loc, ok := interface{}(s.VariableReference).(locator)
		if ok {
			return loc.End()
		}
	}
	return s.Generic_.End()
}

type _PseudoSymbolStack struct {
	lexer parseutil.Lexer[parseutil.Token[SymbolId]]
	top   []*Symbol
}

func (stack *_PseudoSymbolStack) Top() (*Symbol, error) {
	if len(stack.top) == 0 {
		token, err := stack.lexer.Next()
		if err != nil {
			if err != io.EOF {
				return nil, parseutil.NewLocationError(
					stack.lexer.CurrentLocation(),
					"unexpected lex error: %w",
					err)
			}
			token = parseutil.TokenValue[SymbolId]{
				SymbolId: _EndMarker,
				StartEndPos: parseutil.StartEndPos{
					StartPos: stack.lexer.CurrentLocation(),
					EndPos:   stack.lexer.CurrentLocation(),
				},
			}
		}
		item, err := NewSymbol(token)
		if err != nil {
			return nil, err
		}
		stack.top = append(stack.top, item)
	}
	return stack.top[len(stack.top)-1], nil
}

func (stack *_PseudoSymbolStack) Push(symbol *Symbol) {
	stack.top = append(stack.top, symbol)
}

func (stack *_PseudoSymbolStack) Pop() (*Symbol, error) {
	if len(stack.top) == 0 {
		return nil, fmt.Errorf("internal error: cannot pop an empty top")
	}
	ret := stack.top[len(stack.top)-1]
	stack.top = stack.top[:len(stack.top)-1]
	return ret, nil
}

type _StackItem struct {
	StateId _StateId

	*Symbol
}

type _Stack []*_StackItem

type _Action struct {
	ActionType _ActionType

	ShiftStateId _StateId
	ReduceType   _ReduceType
}

func (act *_Action) ShiftItem(symbol *Symbol) *_StackItem {
	return &_StackItem{StateId: act.ShiftStateId, Symbol: symbol}
}

func (act *_Action) ReduceSymbol(
	reducer Reducer,
	stack _Stack,
) (
	_Stack,
	*Symbol,
	error,
) {
	var err error
	symbol := &Symbol{}
	switch act.ReduceType {
	case _ReduceDefinitionToLine:
		args := stack[len(stack)-1:]
		stack = stack[:len(stack)-1]
		symbol.SymbolId_ = LineType
		//line grammar.lr:14:4
		symbol.Line = args[0].Line
		err = nil
	case _ReduceRbraceToLine:
		args := stack[len(stack)-1:]
		stack = stack[:len(stack)-1]
		symbol.SymbolId_ = LineType
		//line grammar.lr:15:4
		symbol.Line = args[0].Line
		err = nil
	case _ReduceLocalLabelToLine:
		args := stack[len(stack)-1:]
		stack = stack[:len(stack)-1]
		symbol.SymbolId_ = LineType
		//line grammar.lr:16:4
		symbol.Line = args[0].LocalLabel
		err = nil
	case _ReduceOperationInstructionToLine:
		args := stack[len(stack)-1:]
		stack = stack[:len(stack)-1]
		symbol.SymbolId_ = LineType
		//line grammar.lr:17:4
		symbol.Line = args[0].Instruction
		err = nil
	case _ReduceControlFlowInstructionToLine:
		args := stack[len(stack)-1:]
		stack = stack[:len(stack)-1]
		symbol.SymbolId_ = LineType
		//line grammar.lr:18:4
		symbol.Line = args[0].Instruction
		err = nil
	case _ReduceFuncToDefinition:
		args := stack[len(stack)-8:]
		stack = stack[:len(stack)-8]
		symbol.SymbolId_ = DefinitionType
		symbol.Line, err = reducer.FuncToDefinition(args[0].Value, args[1].Value, args[2].GlobalLabelReference, args[3].Value, args[4].Parameters, args[5].Value, args[6].Type, args[7].Value)
	case _ReduceToRbrace:
		args := stack[len(stack)-1:]
		stack = stack[:len(stack)-1]
		symbol.SymbolId_ = RbraceType
		symbol.Line, err = reducer.ToRbrace(args[0].Value)
	case _ReduceToGlobalLabel:
		args := stack[len(stack)-2:]
		stack = stack[:len(stack)-2]
		symbol.SymbolId_ = GlobalLabelType
		symbol.GlobalLabelReference, err = reducer.ToGlobalLabel(args[0].Value, args[1].Value)
	case _ReduceToLocalLabel:
		args := stack[len(stack)-2:]
		stack = stack[:len(stack)-2]
		symbol.SymbolId_ = LocalLabelType
		symbol.LocalLabel, err = reducer.ToLocalLabel(args[0].Value, args[1].Value)
	case _ReduceToVariableReference:
		args := stack[len(stack)-2:]
		stack = stack[:len(stack)-2]
		symbol.SymbolId_ = VariableReferenceType
		symbol.VariableReference, err = reducer.ToVariableReference(args[0].Value, args[1].Value)
	case _ReduceIdentifierToIdentifier:
		args := stack[len(stack)-1:]
		stack = stack[:len(stack)-1]
		symbol.SymbolId_ = IdentifierType
		//line grammar.lr:39:4
		symbol.Value = args[0].Value
		err = nil
	case _ReduceStringToIdentifier:
		args := stack[len(stack)-1:]
		stack = stack[:len(stack)-1]
		symbol.SymbolId_ = IdentifierType
		symbol.Value, err = reducer.StringToIdentifier(args[0].Value)
	case _ReduceIntImmediateToImmediate:
		args := stack[len(stack)-1:]
		stack = stack[:len(stack)-1]
		symbol.SymbolId_ = ImmediateType
		//line grammar.lr:43:4
		symbol.OpValue = args[0].OpValue
		err = nil
	case _ReduceFloatImmediateToImmediate:
		args := stack[len(stack)-1:]
		stack = stack[:len(stack)-1]
		symbol.SymbolId_ = ImmediateType
		//line grammar.lr:44:4
		symbol.OpValue = args[0].OpValue
		err = nil
	case _ReduceToIntImmediate:
		args := stack[len(stack)-1:]
		stack = stack[:len(stack)-1]
		symbol.SymbolId_ = IntImmediateType
		symbol.OpValue, err = reducer.ToIntImmediate(args[0].Value)
	case _ReduceToFloatImmediate:
		args := stack[len(stack)-1:]
		stack = stack[:len(stack)-1]
		symbol.SymbolId_ = FloatImmediateType
		symbol.OpValue, err = reducer.ToFloatImmediate(args[0].Value)
	case _ReduceToTypedVariableDefinition:
		args := stack[len(stack)-2:]
		stack = stack[:len(stack)-2]
		symbol.SymbolId_ = TypedVariableDefinitionType
		symbol.VariableDefinition, err = reducer.ToTypedVariableDefinition(args[0].VariableReference, args[1].Type)
	case _ReduceTypedVariableDefinitionToVariableDefinition:
		args := stack[len(stack)-1:]
		stack = stack[:len(stack)-1]
		symbol.SymbolId_ = VariableDefinitionType
		//line grammar.lr:53:4
		symbol.VariableDefinition = args[0].VariableDefinition
		err = nil
	case _ReduceInferredToVariableDefinition:
		args := stack[len(stack)-1:]
		stack = stack[:len(stack)-1]
		symbol.SymbolId_ = VariableDefinitionType
		symbol.VariableDefinition, err = reducer.InferredToVariableDefinition(args[0].VariableReference)
	case _ReduceVariableReferenceToValue:
		args := stack[len(stack)-1:]
		stack = stack[:len(stack)-1]
		symbol.SymbolId_ = ValueType
		//line grammar.lr:57:4
		symbol.OpValue = args[0].VariableReference
		err = nil
	case _ReduceGlobalLabelToValue:
		args := stack[len(stack)-1:]
		stack = stack[:len(stack)-1]
		symbol.SymbolId_ = ValueType
		//line grammar.lr:58:4
		symbol.OpValue = args[0].GlobalLabelReference
		err = nil
	case _ReduceImmediateToValue:
		args := stack[len(stack)-1:]
		stack = stack[:len(stack)-1]
		symbol.SymbolId_ = ValueType
		//line grammar.lr:59:4
		symbol.OpValue = args[0].OpValue
		err = nil
	case _ReduceProperParametersToParameters:
		args := stack[len(stack)-1:]
		stack = stack[:len(stack)-1]
		symbol.SymbolId_ = ParametersType
		//line grammar.lr:66:4
		symbol.Parameters = args[0].Parameters
		err = nil
	case _ReduceImproperToParameters:
		args := stack[len(stack)-2:]
		stack = stack[:len(stack)-2]
		symbol.SymbolId_ = ParametersType
		symbol.Parameters, err = reducer.ImproperToParameters(args[0].Parameters, args[1].Value)
	case _ReduceNilToParameters:
		symbol.SymbolId_ = ParametersType
		symbol.Parameters, err = reducer.NilToParameters()
	case _ReduceAddToProperParameters:
		args := stack[len(stack)-3:]
		stack = stack[:len(stack)-3]
		symbol.SymbolId_ = ProperParametersType
		symbol.Parameters, err = reducer.AddToProperParameters(args[0].Parameters, args[1].Value, args[2].VariableDefinition)
	case _ReduceNewToProperParameters:
		args := stack[len(stack)-1:]
		stack = stack[:len(stack)-1]
		symbol.SymbolId_ = ProperParametersType
		symbol.Parameters, err = reducer.NewToProperParameters(args[0].VariableDefinition)
	case _ReduceProperArgumentsToArguments:
		args := stack[len(stack)-1:]
		stack = stack[:len(stack)-1]
		symbol.SymbolId_ = ArgumentsType
		//line grammar.lr:75:4
		symbol.Arguments = args[0].Arguments
		err = nil
	case _ReduceImproperToArguments:
		args := stack[len(stack)-2:]
		stack = stack[:len(stack)-2]
		symbol.SymbolId_ = ArgumentsType
		symbol.Arguments, err = reducer.ImproperToArguments(args[0].Arguments, args[1].Value)
	case _ReduceNilToArguments:
		symbol.SymbolId_ = ArgumentsType
		symbol.Arguments, err = reducer.NilToArguments()
	case _ReduceAddToProperArguments:
		args := stack[len(stack)-3:]
		stack = stack[:len(stack)-3]
		symbol.SymbolId_ = ProperArgumentsType
		symbol.Arguments, err = reducer.AddToProperArguments(args[0].Arguments, args[1].Value, args[2].OpValue)
	case _ReduceNewToProperArguments:
		args := stack[len(stack)-1:]
		stack = stack[:len(stack)-1]
		symbol.SymbolId_ = ProperArgumentsType
		symbol.Arguments, err = reducer.NewToProperArguments(args[0].OpValue)
	case _ReduceProperTypesToTypes:
		args := stack[len(stack)-1:]
		stack = stack[:len(stack)-1]
		symbol.SymbolId_ = TypesType
		//line grammar.lr:84:4
		symbol.Types = args[0].Types
		err = nil
	case _ReduceImproperToTypes:
		args := stack[len(stack)-2:]
		stack = stack[:len(stack)-2]
		symbol.SymbolId_ = TypesType
		symbol.Types, err = reducer.ImproperToTypes(args[0].Types, args[1].Value)
	case _ReduceNilToTypes:
		symbol.SymbolId_ = TypesType
		symbol.Types, err = reducer.NilToTypes()
	case _ReduceAddToProperTypes:
		args := stack[len(stack)-3:]
		stack = stack[:len(stack)-3]
		symbol.SymbolId_ = ProperTypesType
		symbol.Types, err = reducer.AddToProperTypes(args[0].Types, args[1].Value, args[2].Type)
	case _ReduceNewToProperTypes:
		args := stack[len(stack)-1:]
		stack = stack[:len(stack)-1]
		symbol.SymbolId_ = ProperTypesType
		symbol.Types, err = reducer.NewToProperTypes(args[0].Type)
	case _ReduceAssignToOperationInstruction:
		args := stack[len(stack)-3:]
		stack = stack[:len(stack)-3]
		symbol.SymbolId_ = OperationInstructionType
		symbol.Instruction, err = reducer.AssignToOperationInstruction(args[0].VariableDefinition, args[1].Value, args[2].OpValue)
	case _ReduceUnaryToOperationInstruction:
		args := stack[len(stack)-4:]
		stack = stack[:len(stack)-4]
		symbol.SymbolId_ = OperationInstructionType
		symbol.Instruction, err = reducer.UnaryToOperationInstruction(args[0].VariableDefinition, args[1].Value, args[2].Value, args[3].OpValue)
	case _ReduceBinaryToOperationInstruction:
		args := stack[len(stack)-6:]
		stack = stack[:len(stack)-6]
		symbol.SymbolId_ = OperationInstructionType
		symbol.Instruction, err = reducer.BinaryToOperationInstruction(args[0].VariableDefinition, args[1].Value, args[2].Value, args[3].OpValue, args[4].Value, args[5].OpValue)
	case _ReduceCallToOperationInstruction:
		args := stack[len(stack)-7:]
		stack = stack[:len(stack)-7]
		symbol.SymbolId_ = OperationInstructionType
		symbol.Instruction, err = reducer.CallToOperationInstruction(args[0].VariableDefinition, args[1].Value, args[2].Value, args[3].OpValue, args[4].Value, args[5].Arguments, args[6].Value)
	case _ReduceUnconditionalToControlFlowInstruction:
		args := stack[len(stack)-2:]
		stack = stack[:len(stack)-2]
		symbol.SymbolId_ = ControlFlowInstructionType
		symbol.Instruction, err = reducer.UnconditionalToControlFlowInstruction(args[0].Value, args[1].LocalLabel)
	case _ReduceConditionalToControlFlowInstruction:
		args := stack[len(stack)-6:]
		stack = stack[:len(stack)-6]
		symbol.SymbolId_ = ControlFlowInstructionType
		symbol.Instruction, err = reducer.ConditionalToControlFlowInstruction(args[0].Value, args[1].LocalLabel, args[2].Value, args[3].OpValue, args[4].Value, args[5].OpValue)
	case _ReduceTerminalToControlFlowInstruction:
		args := stack[len(stack)-2:]
		stack = stack[:len(stack)-2]
		symbol.SymbolId_ = ControlFlowInstructionType
		symbol.Instruction, err = reducer.TerminalToControlFlowInstruction(args[0].Value, args[1].OpValue)
	case _ReduceNumberTypeToType:
		args := stack[len(stack)-1:]
		stack = stack[:len(stack)-1]
		symbol.SymbolId_ = TypeType
		//line grammar.lr:113:4
		symbol.Type = args[0].Type
		err = nil
	case _ReduceFuncTypeToType:
		args := stack[len(stack)-1:]
		stack = stack[:len(stack)-1]
		symbol.SymbolId_ = TypeType
		//line grammar.lr:114:4
		symbol.Type = args[0].Type
		err = nil
	case _ReduceToNumberType:
		args := stack[len(stack)-1:]
		stack = stack[:len(stack)-1]
		symbol.SymbolId_ = NumberTypeType
		symbol.Type, err = reducer.ToNumberType(args[0].Value)
	case _ReduceToFuncType:
		args := stack[len(stack)-5:]
		stack = stack[:len(stack)-5]
		symbol.SymbolId_ = FuncTypeType
		symbol.Type, err = reducer.ToFuncType(args[0].Value, args[1].Value, args[2].Types, args[3].Value, args[4].Type)
	default:
		panic("Unknown reduce type: " + act.ReduceType.String())
	}

	if err != nil {
		err = fmt.Errorf("unexpected %s reduce error: %w", act.ReduceType, err)
	}

	return stack, symbol, err
}

type _ActionTableKey struct {
	_StateId
	SymbolId
}

type _ActionTableType struct{}

func (_ActionTableType) Get(
	stateId _StateId,
	symbolId SymbolId,
) (
	_Action,
	bool,
) {
	switch stateId {
	case _State1:
		switch symbolId {
		case IdentifierToken:
			return _Action{_ShiftAction, _State5, 0}, true
		case ColonToken:
			return _Action{_ShiftAction, _State3, 0}, true
		case PercentToken:
			return _Action{_ShiftAction, _State6, 0}, true
		case DefineToken:
			return _Action{_ShiftAction, _State4, 0}, true
		case LineType:
			return _Action{_ShiftAction, _State2, 0}, true
		case VariableReferenceType:
			return _Action{_ShiftAction, _State8, 0}, true
		case VariableDefinitionType:
			return _Action{_ShiftAction, _State7, 0}, true
		case RbraceToken:
			return _Action{_ShiftAndReduceAction, 0, _ReduceToRbrace}, true
		case DefinitionType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceDefinitionToLine}, true
		case RbraceType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceRbraceToLine}, true
		case LocalLabelType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceLocalLabelToLine}, true
		case TypedVariableDefinitionType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceTypedVariableDefinitionToVariableDefinition}, true
		case OperationInstructionType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceOperationInstructionToLine}, true
		case ControlFlowInstructionType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceControlFlowInstructionToLine}, true
		}
	case _State2:
		switch symbolId {
		case _EndMarker:
			return _Action{_AcceptAction, 0, 0}, true
		}
	case _State3:
		switch symbolId {
		case StringLiteralToken:
			return _Action{_ShiftAndReduceAction, 0, _ReduceStringToIdentifier}, true
		case IdentifierToken:
			return _Action{_ShiftAndReduceAction, 0, _ReduceIdentifierToIdentifier}, true
		case IdentifierType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceToLocalLabel}, true
		}
	case _State4:
		switch symbolId {
		case FuncToken:
			return _Action{_ShiftAction, _State9, 0}, true
		}
	case _State5:
		switch symbolId {
		case ColonToken:
			return _Action{_ShiftAction, _State3, 0}, true
		case AtToken:
			return _Action{_ShiftAction, _State10, 0}, true
		case PercentToken:
			return _Action{_ShiftAction, _State6, 0}, true
		case LocalLabelType:
			return _Action{_ShiftAction, _State11, 0}, true
		case IntegerLiteralToken:
			return _Action{_ShiftAndReduceAction, 0, _ReduceToIntImmediate}, true
		case FloatLiteralToken:
			return _Action{_ShiftAndReduceAction, 0, _ReduceToFloatImmediate}, true
		case GlobalLabelType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceGlobalLabelToValue}, true
		case VariableReferenceType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceVariableReferenceToValue}, true
		case ImmediateType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceImmediateToValue}, true
		case IntImmediateType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceIntImmediateToImmediate}, true
		case FloatImmediateType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceFloatImmediateToImmediate}, true
		case ValueType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceTerminalToControlFlowInstruction}, true
		}
	case _State6:
		switch symbolId {
		case StringLiteralToken:
			return _Action{_ShiftAndReduceAction, 0, _ReduceStringToIdentifier}, true
		case IdentifierToken:
			return _Action{_ShiftAndReduceAction, 0, _ReduceIdentifierToIdentifier}, true
		case IdentifierType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceToVariableReference}, true
		}
	case _State7:
		switch symbolId {
		case EqualToken:
			return _Action{_ShiftAction, _State12, 0}, true
		}
	case _State8:
		switch symbolId {
		case FuncToken:
			return _Action{_ShiftAction, _State13, 0}, true
		case IdentifierToken:
			return _Action{_ShiftAndReduceAction, 0, _ReduceToNumberType}, true
		case TypeType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceToTypedVariableDefinition}, true
		case NumberTypeType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceNumberTypeToType}, true
		case FuncTypeType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceFuncTypeToType}, true

		default:
			return _Action{_ReduceAction, 0, _ReduceInferredToVariableDefinition}, true
		}
	case _State9:
		switch symbolId {
		case AtToken:
			return _Action{_ShiftAction, _State10, 0}, true
		case GlobalLabelType:
			return _Action{_ShiftAction, _State14, 0}, true
		}
	case _State10:
		switch symbolId {
		case StringLiteralToken:
			return _Action{_ShiftAndReduceAction, 0, _ReduceStringToIdentifier}, true
		case IdentifierToken:
			return _Action{_ShiftAndReduceAction, 0, _ReduceIdentifierToIdentifier}, true
		case IdentifierType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceToGlobalLabel}, true
		}
	case _State11:
		switch symbolId {
		case CommaToken:
			return _Action{_ShiftAction, _State15, 0}, true

		default:
			return _Action{_ReduceAction, 0, _ReduceUnconditionalToControlFlowInstruction}, true
		}
	case _State12:
		switch symbolId {
		case IdentifierToken:
			return _Action{_ShiftAction, _State16, 0}, true
		case AtToken:
			return _Action{_ShiftAction, _State10, 0}, true
		case PercentToken:
			return _Action{_ShiftAction, _State6, 0}, true
		case IntegerLiteralToken:
			return _Action{_ShiftAndReduceAction, 0, _ReduceToIntImmediate}, true
		case FloatLiteralToken:
			return _Action{_ShiftAndReduceAction, 0, _ReduceToFloatImmediate}, true
		case GlobalLabelType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceGlobalLabelToValue}, true
		case VariableReferenceType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceVariableReferenceToValue}, true
		case ImmediateType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceImmediateToValue}, true
		case IntImmediateType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceIntImmediateToImmediate}, true
		case FloatImmediateType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceFloatImmediateToImmediate}, true
		case ValueType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceAssignToOperationInstruction}, true
		}
	case _State13:
		switch symbolId {
		case LparenToken:
			return _Action{_ShiftAction, _State17, 0}, true
		}
	case _State14:
		switch symbolId {
		case LparenToken:
			return _Action{_ShiftAction, _State18, 0}, true
		}
	case _State15:
		switch symbolId {
		case AtToken:
			return _Action{_ShiftAction, _State10, 0}, true
		case PercentToken:
			return _Action{_ShiftAction, _State6, 0}, true
		case ValueType:
			return _Action{_ShiftAction, _State19, 0}, true
		case IntegerLiteralToken:
			return _Action{_ShiftAndReduceAction, 0, _ReduceToIntImmediate}, true
		case FloatLiteralToken:
			return _Action{_ShiftAndReduceAction, 0, _ReduceToFloatImmediate}, true
		case GlobalLabelType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceGlobalLabelToValue}, true
		case VariableReferenceType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceVariableReferenceToValue}, true
		case ImmediateType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceImmediateToValue}, true
		case IntImmediateType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceIntImmediateToImmediate}, true
		case FloatImmediateType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceFloatImmediateToImmediate}, true
		}
	case _State16:
		switch symbolId {
		case AtToken:
			return _Action{_ShiftAction, _State10, 0}, true
		case PercentToken:
			return _Action{_ShiftAction, _State6, 0}, true
		case ValueType:
			return _Action{_ShiftAction, _State20, 0}, true
		case IntegerLiteralToken:
			return _Action{_ShiftAndReduceAction, 0, _ReduceToIntImmediate}, true
		case FloatLiteralToken:
			return _Action{_ShiftAndReduceAction, 0, _ReduceToFloatImmediate}, true
		case GlobalLabelType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceGlobalLabelToValue}, true
		case VariableReferenceType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceVariableReferenceToValue}, true
		case ImmediateType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceImmediateToValue}, true
		case IntImmediateType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceIntImmediateToImmediate}, true
		case FloatImmediateType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceFloatImmediateToImmediate}, true
		}
	case _State17:
		switch symbolId {
		case FuncToken:
			return _Action{_ShiftAction, _State13, 0}, true
		case TypesType:
			return _Action{_ShiftAction, _State22, 0}, true
		case ProperTypesType:
			return _Action{_ShiftAction, _State21, 0}, true
		case IdentifierToken:
			return _Action{_ShiftAndReduceAction, 0, _ReduceToNumberType}, true
		case TypeType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceNewToProperTypes}, true
		case NumberTypeType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceNumberTypeToType}, true
		case FuncTypeType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceFuncTypeToType}, true

		default:
			return _Action{_ReduceAction, 0, _ReduceNilToTypes}, true
		}
	case _State18:
		switch symbolId {
		case PercentToken:
			return _Action{_ShiftAction, _State6, 0}, true
		case VariableReferenceType:
			return _Action{_ShiftAction, _State25, 0}, true
		case ParametersType:
			return _Action{_ShiftAction, _State23, 0}, true
		case ProperParametersType:
			return _Action{_ShiftAction, _State24, 0}, true
		case TypedVariableDefinitionType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceNewToProperParameters}, true

		default:
			return _Action{_ReduceAction, 0, _ReduceNilToParameters}, true
		}
	case _State19:
		switch symbolId {
		case CommaToken:
			return _Action{_ShiftAction, _State26, 0}, true
		}
	case _State20:
		switch symbolId {
		case LparenToken:
			return _Action{_ShiftAction, _State28, 0}, true
		case CommaToken:
			return _Action{_ShiftAction, _State27, 0}, true

		default:
			return _Action{_ReduceAction, 0, _ReduceUnaryToOperationInstruction}, true
		}
	case _State21:
		switch symbolId {
		case CommaToken:
			return _Action{_ShiftAction, _State29, 0}, true

		default:
			return _Action{_ReduceAction, 0, _ReduceProperTypesToTypes}, true
		}
	case _State22:
		switch symbolId {
		case RparenToken:
			return _Action{_ShiftAction, _State30, 0}, true
		}
	case _State23:
		switch symbolId {
		case RparenToken:
			return _Action{_ShiftAction, _State31, 0}, true
		}
	case _State24:
		switch symbolId {
		case CommaToken:
			return _Action{_ShiftAction, _State32, 0}, true

		default:
			return _Action{_ReduceAction, 0, _ReduceProperParametersToParameters}, true
		}
	case _State25:
		switch symbolId {
		case FuncToken:
			return _Action{_ShiftAction, _State13, 0}, true
		case IdentifierToken:
			return _Action{_ShiftAndReduceAction, 0, _ReduceToNumberType}, true
		case TypeType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceToTypedVariableDefinition}, true
		case NumberTypeType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceNumberTypeToType}, true
		case FuncTypeType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceFuncTypeToType}, true
		}
	case _State26:
		switch symbolId {
		case AtToken:
			return _Action{_ShiftAction, _State10, 0}, true
		case PercentToken:
			return _Action{_ShiftAction, _State6, 0}, true
		case IntegerLiteralToken:
			return _Action{_ShiftAndReduceAction, 0, _ReduceToIntImmediate}, true
		case FloatLiteralToken:
			return _Action{_ShiftAndReduceAction, 0, _ReduceToFloatImmediate}, true
		case GlobalLabelType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceGlobalLabelToValue}, true
		case VariableReferenceType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceVariableReferenceToValue}, true
		case ImmediateType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceImmediateToValue}, true
		case IntImmediateType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceIntImmediateToImmediate}, true
		case FloatImmediateType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceFloatImmediateToImmediate}, true
		case ValueType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceConditionalToControlFlowInstruction}, true
		}
	case _State27:
		switch symbolId {
		case AtToken:
			return _Action{_ShiftAction, _State10, 0}, true
		case PercentToken:
			return _Action{_ShiftAction, _State6, 0}, true
		case IntegerLiteralToken:
			return _Action{_ShiftAndReduceAction, 0, _ReduceToIntImmediate}, true
		case FloatLiteralToken:
			return _Action{_ShiftAndReduceAction, 0, _ReduceToFloatImmediate}, true
		case GlobalLabelType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceGlobalLabelToValue}, true
		case VariableReferenceType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceVariableReferenceToValue}, true
		case ImmediateType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceImmediateToValue}, true
		case IntImmediateType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceIntImmediateToImmediate}, true
		case FloatImmediateType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceFloatImmediateToImmediate}, true
		case ValueType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceBinaryToOperationInstruction}, true
		}
	case _State28:
		switch symbolId {
		case AtToken:
			return _Action{_ShiftAction, _State10, 0}, true
		case PercentToken:
			return _Action{_ShiftAction, _State6, 0}, true
		case ArgumentsType:
			return _Action{_ShiftAction, _State33, 0}, true
		case ProperArgumentsType:
			return _Action{_ShiftAction, _State34, 0}, true
		case IntegerLiteralToken:
			return _Action{_ShiftAndReduceAction, 0, _ReduceToIntImmediate}, true
		case FloatLiteralToken:
			return _Action{_ShiftAndReduceAction, 0, _ReduceToFloatImmediate}, true
		case GlobalLabelType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceGlobalLabelToValue}, true
		case VariableReferenceType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceVariableReferenceToValue}, true
		case ImmediateType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceImmediateToValue}, true
		case IntImmediateType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceIntImmediateToImmediate}, true
		case FloatImmediateType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceFloatImmediateToImmediate}, true
		case ValueType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceNewToProperArguments}, true

		default:
			return _Action{_ReduceAction, 0, _ReduceNilToArguments}, true
		}
	case _State29:
		switch symbolId {
		case FuncToken:
			return _Action{_ShiftAction, _State13, 0}, true
		case IdentifierToken:
			return _Action{_ShiftAndReduceAction, 0, _ReduceToNumberType}, true
		case TypeType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceAddToProperTypes}, true
		case NumberTypeType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceNumberTypeToType}, true
		case FuncTypeType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceFuncTypeToType}, true

		default:
			return _Action{_ReduceAction, 0, _ReduceImproperToTypes}, true
		}
	case _State30:
		switch symbolId {
		case FuncToken:
			return _Action{_ShiftAction, _State13, 0}, true
		case IdentifierToken:
			return _Action{_ShiftAndReduceAction, 0, _ReduceToNumberType}, true
		case TypeType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceToFuncType}, true
		case NumberTypeType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceNumberTypeToType}, true
		case FuncTypeType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceFuncTypeToType}, true
		}
	case _State31:
		switch symbolId {
		case FuncToken:
			return _Action{_ShiftAction, _State13, 0}, true
		case TypeType:
			return _Action{_ShiftAction, _State35, 0}, true
		case IdentifierToken:
			return _Action{_ShiftAndReduceAction, 0, _ReduceToNumberType}, true
		case NumberTypeType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceNumberTypeToType}, true
		case FuncTypeType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceFuncTypeToType}, true
		}
	case _State32:
		switch symbolId {
		case PercentToken:
			return _Action{_ShiftAction, _State6, 0}, true
		case VariableReferenceType:
			return _Action{_ShiftAction, _State25, 0}, true
		case TypedVariableDefinitionType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceAddToProperParameters}, true

		default:
			return _Action{_ReduceAction, 0, _ReduceImproperToParameters}, true
		}
	case _State33:
		switch symbolId {
		case RparenToken:
			return _Action{_ShiftAndReduceAction, 0, _ReduceCallToOperationInstruction}, true
		}
	case _State34:
		switch symbolId {
		case CommaToken:
			return _Action{_ShiftAction, _State36, 0}, true

		default:
			return _Action{_ReduceAction, 0, _ReduceProperArgumentsToArguments}, true
		}
	case _State35:
		switch symbolId {
		case LbraceToken:
			return _Action{_ShiftAndReduceAction, 0, _ReduceFuncToDefinition}, true
		}
	case _State36:
		switch symbolId {
		case AtToken:
			return _Action{_ShiftAction, _State10, 0}, true
		case PercentToken:
			return _Action{_ShiftAction, _State6, 0}, true
		case IntegerLiteralToken:
			return _Action{_ShiftAndReduceAction, 0, _ReduceToIntImmediate}, true
		case FloatLiteralToken:
			return _Action{_ShiftAndReduceAction, 0, _ReduceToFloatImmediate}, true
		case GlobalLabelType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceGlobalLabelToValue}, true
		case VariableReferenceType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceVariableReferenceToValue}, true
		case ImmediateType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceImmediateToValue}, true
		case IntImmediateType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceIntImmediateToImmediate}, true
		case FloatImmediateType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceFloatImmediateToImmediate}, true
		case ValueType:
			return _Action{_ShiftAndReduceAction, 0, _ReduceAddToProperArguments}, true

		default:
			return _Action{_ReduceAction, 0, _ReduceImproperToArguments}, true
		}
	}

	return _Action{}, false
}

var _ActionTable = _ActionTableType{}

/*
Parser Debug States:
  State 1:
    Kernel Items:
      #accept: ^.line
    Reduce:
      (nil)
    ShiftAndReduce:
      RBRACE -> [rbrace]
      definition -> [line]
      rbrace -> [line]
      local_label -> [line]
      typed_variable_definition -> [variable_definition]
      operation_instruction -> [line]
      control_flow_instruction -> [line]
    Goto:
      IDENTIFIER -> State 5
      COLON -> State 3
      PERCENT -> State 6
      DEFINE -> State 4
      line -> State 2
      variable_reference -> State 8
      variable_definition -> State 7

  State 2:
    Kernel Items:
      #accept: ^ line., $
    Reduce:
      $ -> [#accept]
    ShiftAndReduce:
      (nil)
    Goto:
      (nil)

  State 3:
    Kernel Items:
      local_label: COLON.identifier
    Reduce:
      (nil)
    ShiftAndReduce:
      STRING_LITERAL -> [identifier]
      IDENTIFIER -> [identifier]
      identifier -> [local_label]
    Goto:
      (nil)

  State 4:
    Kernel Items:
      definition: DEFINE.FUNC global_label LPAREN parameters RPAREN type LBRACE
    Reduce:
      (nil)
    ShiftAndReduce:
      (nil)
    Goto:
      FUNC -> State 9

  State 5:
    Kernel Items:
      control_flow_instruction: IDENTIFIER.local_label
      control_flow_instruction: IDENTIFIER.local_label COMMA value COMMA value
      control_flow_instruction: IDENTIFIER.value
    Reduce:
      (nil)
    ShiftAndReduce:
      INTEGER_LITERAL -> [int_immediate]
      FLOAT_LITERAL -> [float_immediate]
      global_label -> [value]
      variable_reference -> [value]
      immediate -> [value]
      int_immediate -> [immediate]
      float_immediate -> [immediate]
      value -> [control_flow_instruction]
    Goto:
      COLON -> State 3
      AT -> State 10
      PERCENT -> State 6
      local_label -> State 11

  State 6:
    Kernel Items:
      variable_reference: PERCENT.identifier
    Reduce:
      (nil)
    ShiftAndReduce:
      STRING_LITERAL -> [identifier]
      IDENTIFIER -> [identifier]
      identifier -> [variable_reference]
    Goto:
      (nil)

  State 7:
    Kernel Items:
      operation_instruction: variable_definition.EQUAL value
      operation_instruction: variable_definition.EQUAL IDENTIFIER value
      operation_instruction: variable_definition.EQUAL IDENTIFIER value COMMA value
      operation_instruction: variable_definition.EQUAL IDENTIFIER value LPAREN arguments RPAREN
    Reduce:
      (nil)
    ShiftAndReduce:
      (nil)
    Goto:
      EQUAL -> State 12

  State 8:
    Kernel Items:
      typed_variable_definition: variable_reference.type
      variable_definition: variable_reference., *
    Reduce:
      * -> [variable_definition]
    ShiftAndReduce:
      IDENTIFIER -> [number_type]
      type -> [typed_variable_definition]
      number_type -> [type]
      func_type -> [type]
    Goto:
      FUNC -> State 13

  State 9:
    Kernel Items:
      definition: DEFINE FUNC.global_label LPAREN parameters RPAREN type LBRACE
    Reduce:
      (nil)
    ShiftAndReduce:
      (nil)
    Goto:
      AT -> State 10
      global_label -> State 14

  State 10:
    Kernel Items:
      global_label: AT.identifier
    Reduce:
      (nil)
    ShiftAndReduce:
      STRING_LITERAL -> [identifier]
      IDENTIFIER -> [identifier]
      identifier -> [global_label]
    Goto:
      (nil)

  State 11:
    Kernel Items:
      control_flow_instruction: IDENTIFIER local_label., *
      control_flow_instruction: IDENTIFIER local_label.COMMA value COMMA value
    Reduce:
      * -> [control_flow_instruction]
    ShiftAndReduce:
      (nil)
    Goto:
      COMMA -> State 15

  State 12:
    Kernel Items:
      operation_instruction: variable_definition EQUAL.value
      operation_instruction: variable_definition EQUAL.IDENTIFIER value
      operation_instruction: variable_definition EQUAL.IDENTIFIER value COMMA value
      operation_instruction: variable_definition EQUAL.IDENTIFIER value LPAREN arguments RPAREN
    Reduce:
      (nil)
    ShiftAndReduce:
      INTEGER_LITERAL -> [int_immediate]
      FLOAT_LITERAL -> [float_immediate]
      global_label -> [value]
      variable_reference -> [value]
      immediate -> [value]
      int_immediate -> [immediate]
      float_immediate -> [immediate]
      value -> [operation_instruction]
    Goto:
      IDENTIFIER -> State 16
      AT -> State 10
      PERCENT -> State 6

  State 13:
    Kernel Items:
      func_type: FUNC.LPAREN types RPAREN type
    Reduce:
      (nil)
    ShiftAndReduce:
      (nil)
    Goto:
      LPAREN -> State 17

  State 14:
    Kernel Items:
      definition: DEFINE FUNC global_label.LPAREN parameters RPAREN type LBRACE
    Reduce:
      (nil)
    ShiftAndReduce:
      (nil)
    Goto:
      LPAREN -> State 18

  State 15:
    Kernel Items:
      control_flow_instruction: IDENTIFIER local_label COMMA.value COMMA value
    Reduce:
      (nil)
    ShiftAndReduce:
      INTEGER_LITERAL -> [int_immediate]
      FLOAT_LITERAL -> [float_immediate]
      global_label -> [value]
      variable_reference -> [value]
      immediate -> [value]
      int_immediate -> [immediate]
      float_immediate -> [immediate]
    Goto:
      AT -> State 10
      PERCENT -> State 6
      value -> State 19

  State 16:
    Kernel Items:
      operation_instruction: variable_definition EQUAL IDENTIFIER.value
      operation_instruction: variable_definition EQUAL IDENTIFIER.value COMMA value
      operation_instruction: variable_definition EQUAL IDENTIFIER.value LPAREN arguments RPAREN
    Reduce:
      (nil)
    ShiftAndReduce:
      INTEGER_LITERAL -> [int_immediate]
      FLOAT_LITERAL -> [float_immediate]
      global_label -> [value]
      variable_reference -> [value]
      immediate -> [value]
      int_immediate -> [immediate]
      float_immediate -> [immediate]
    Goto:
      AT -> State 10
      PERCENT -> State 6
      value -> State 20

  State 17:
    Kernel Items:
      func_type: FUNC LPAREN.types RPAREN type
    Reduce:
      * -> [types]
    ShiftAndReduce:
      IDENTIFIER -> [number_type]
      type -> [proper_types]
      number_type -> [type]
      func_type -> [type]
    Goto:
      FUNC -> State 13
      types -> State 22
      proper_types -> State 21

  State 18:
    Kernel Items:
      definition: DEFINE FUNC global_label LPAREN.parameters RPAREN type LBRACE
    Reduce:
      * -> [parameters]
    ShiftAndReduce:
      typed_variable_definition -> [proper_parameters]
    Goto:
      PERCENT -> State 6
      variable_reference -> State 25
      parameters -> State 23
      proper_parameters -> State 24

  State 19:
    Kernel Items:
      control_flow_instruction: IDENTIFIER local_label COMMA value.COMMA value
    Reduce:
      (nil)
    ShiftAndReduce:
      (nil)
    Goto:
      COMMA -> State 26

  State 20:
    Kernel Items:
      operation_instruction: variable_definition EQUAL IDENTIFIER value., *
      operation_instruction: variable_definition EQUAL IDENTIFIER value.COMMA value
      operation_instruction: variable_definition EQUAL IDENTIFIER value.LPAREN arguments RPAREN
    Reduce:
      * -> [operation_instruction]
    ShiftAndReduce:
      (nil)
    Goto:
      LPAREN -> State 28
      COMMA -> State 27

  State 21:
    Kernel Items:
      types: proper_types., *
      types: proper_types.COMMA
      proper_types: proper_types.COMMA type
    Reduce:
      * -> [types]
    ShiftAndReduce:
      (nil)
    Goto:
      COMMA -> State 29

  State 22:
    Kernel Items:
      func_type: FUNC LPAREN types.RPAREN type
    Reduce:
      (nil)
    ShiftAndReduce:
      (nil)
    Goto:
      RPAREN -> State 30

  State 23:
    Kernel Items:
      definition: DEFINE FUNC global_label LPAREN parameters.RPAREN type LBRACE
    Reduce:
      (nil)
    ShiftAndReduce:
      (nil)
    Goto:
      RPAREN -> State 31

  State 24:
    Kernel Items:
      parameters: proper_parameters., *
      parameters: proper_parameters.COMMA
      proper_parameters: proper_parameters.COMMA typed_variable_definition
    Reduce:
      * -> [parameters]
    ShiftAndReduce:
      (nil)
    Goto:
      COMMA -> State 32

  State 25:
    Kernel Items:
      typed_variable_definition: variable_reference.type
    Reduce:
      (nil)
    ShiftAndReduce:
      IDENTIFIER -> [number_type]
      type -> [typed_variable_definition]
      number_type -> [type]
      func_type -> [type]
    Goto:
      FUNC -> State 13

  State 26:
    Kernel Items:
      control_flow_instruction: IDENTIFIER local_label COMMA value COMMA.value
    Reduce:
      (nil)
    ShiftAndReduce:
      INTEGER_LITERAL -> [int_immediate]
      FLOAT_LITERAL -> [float_immediate]
      global_label -> [value]
      variable_reference -> [value]
      immediate -> [value]
      int_immediate -> [immediate]
      float_immediate -> [immediate]
      value -> [control_flow_instruction]
    Goto:
      AT -> State 10
      PERCENT -> State 6

  State 27:
    Kernel Items:
      operation_instruction: variable_definition EQUAL IDENTIFIER value COMMA.value
    Reduce:
      (nil)
    ShiftAndReduce:
      INTEGER_LITERAL -> [int_immediate]
      FLOAT_LITERAL -> [float_immediate]
      global_label -> [value]
      variable_reference -> [value]
      immediate -> [value]
      int_immediate -> [immediate]
      float_immediate -> [immediate]
      value -> [operation_instruction]
    Goto:
      AT -> State 10
      PERCENT -> State 6

  State 28:
    Kernel Items:
      operation_instruction: variable_definition EQUAL IDENTIFIER value LPAREN.arguments RPAREN
    Reduce:
      * -> [arguments]
    ShiftAndReduce:
      INTEGER_LITERAL -> [int_immediate]
      FLOAT_LITERAL -> [float_immediate]
      global_label -> [value]
      variable_reference -> [value]
      immediate -> [value]
      int_immediate -> [immediate]
      float_immediate -> [immediate]
      value -> [proper_arguments]
    Goto:
      AT -> State 10
      PERCENT -> State 6
      arguments -> State 33
      proper_arguments -> State 34

  State 29:
    Kernel Items:
      types: proper_types COMMA., *
      proper_types: proper_types COMMA.type
    Reduce:
      * -> [types]
    ShiftAndReduce:
      IDENTIFIER -> [number_type]
      type -> [proper_types]
      number_type -> [type]
      func_type -> [type]
    Goto:
      FUNC -> State 13

  State 30:
    Kernel Items:
      func_type: FUNC LPAREN types RPAREN.type
    Reduce:
      (nil)
    ShiftAndReduce:
      IDENTIFIER -> [number_type]
      type -> [func_type]
      number_type -> [type]
      func_type -> [type]
    Goto:
      FUNC -> State 13

  State 31:
    Kernel Items:
      definition: DEFINE FUNC global_label LPAREN parameters RPAREN.type LBRACE
    Reduce:
      (nil)
    ShiftAndReduce:
      IDENTIFIER -> [number_type]
      number_type -> [type]
      func_type -> [type]
    Goto:
      FUNC -> State 13
      type -> State 35

  State 32:
    Kernel Items:
      parameters: proper_parameters COMMA., *
      proper_parameters: proper_parameters COMMA.typed_variable_definition
    Reduce:
      * -> [parameters]
    ShiftAndReduce:
      typed_variable_definition -> [proper_parameters]
    Goto:
      PERCENT -> State 6
      variable_reference -> State 25

  State 33:
    Kernel Items:
      operation_instruction: variable_definition EQUAL IDENTIFIER value LPAREN arguments.RPAREN
    Reduce:
      (nil)
    ShiftAndReduce:
      RPAREN -> [operation_instruction]
    Goto:
      (nil)

  State 34:
    Kernel Items:
      arguments: proper_arguments., *
      arguments: proper_arguments.COMMA
      proper_arguments: proper_arguments.COMMA value
    Reduce:
      * -> [arguments]
    ShiftAndReduce:
      (nil)
    Goto:
      COMMA -> State 36

  State 35:
    Kernel Items:
      definition: DEFINE FUNC global_label LPAREN parameters RPAREN type.LBRACE
    Reduce:
      (nil)
    ShiftAndReduce:
      LBRACE -> [definition]
    Goto:
      (nil)

  State 36:
    Kernel Items:
      arguments: proper_arguments COMMA., *
      proper_arguments: proper_arguments COMMA.value
    Reduce:
      * -> [arguments]
    ShiftAndReduce:
      INTEGER_LITERAL -> [int_immediate]
      FLOAT_LITERAL -> [float_immediate]
      global_label -> [value]
      variable_reference -> [value]
      immediate -> [value]
      int_immediate -> [immediate]
      float_immediate -> [immediate]
      value -> [proper_arguments]
    Goto:
      AT -> State 10
      PERCENT -> State 6

Number of states: 36
Number of shift actions: 60
Number of reduce actions: 13
Number of shift-and-reduce actions: 105
Number of shift/reduce conflicts: 0
Number of reduce/reduce conflicts: 0
Number of unoptimized states: 144
Number of unoptimized shift actions: 223
Number of unoptimized reduce actions: 161
*/
