package main

import "fmt"

func Tokenize(tokenStream []string) (*CommandV2, error) {
	if len(tokenStream) == 0 {
		return nil, nil
	}

	if IsOperator(tokenStream[0]) {
		return nil, fmt.Errorf("syntax error: unexpected end of file after operator %s", tokenStream[0])
	}

	current := &CommandV2{
		Args: []string{},
		// Redirect: &RedirectConfig{},
	}

	i := 0
	for i < len(tokenStream) {
		token := tokenStream[i]
		op := Operator(token)

		// at the time of implementation, the only Operator Codecrafters tests for is "|"
		if op == OperatorPipe || op == OperatorSequentialExecution ||
			op == OperatorLogicalAnd || op == OperatorLogicalOr || op == OperatorBackgroundExecution {
			current.NextOp = op
			if i+1 < len(tokenStream) {
				current.NextCmd = &CommandV2{
					Args:     []string{},
					Redirect: &RedirectConfig{},
				}

				current = current.NextCmd
			}
			i++
			continue
		}

		if op == OperatorRTruncate || op == OperatorRTruncate1 ||
			op == OperatorRAppend || op == OperatorRAppend1 ||
			op == OperatorRErrorTruncate || op == OperatorRErrorAppend ||
			op == OperatorROutErrorAppend || op == OperatorHereDocument ||
			op == OperatorHereString {
			if i+1 >= len(tokenStream) {
				return nil, fmt.Errorf("syntax error: unexpected end of file after redirection operator %s", token)
			}

			if IsOperator(tokenStream[i+1]) {
				return nil, fmt.Errorf("syntax error: unexpected end of file after redirection operator %s", token)
			}

			current.Redirect = &RedirectConfig{
				Type:     Int(token),
				FilePath: tokenStream[i+1],
			}
			i += 2
			continue
		}

		current.Args = append(current.Args, token)
		i++
	}

	if len(current.Args) == 0 && current.Redirect == nil {
		return nil, fmt.Errorf("syntax error: empty command stream")
	}

	return current, nil
}

type Operator string

// Redirect Operator
const (
	OperatorRTruncate       Operator = ">"
	OperatorRTruncate1      Operator = "1>"
	OperatorRAppend         Operator = ">>"
	OperatorRAppend1        Operator = "1>>"
	OperatorRErrorTruncate  Operator = "2>"
	OperatorRErrorAppend    Operator = "2>>"
	OperatorROutErrorAppend Operator = "&>"
	OperatorHereDocument    Operator = "<<"
	OperatorHereString      Operator = "<<<"
)

// Redirect Operator

const (
	OperatorPipe                Operator = "|"
	OperatorSequentialExecution Operator = ";"
	OperatorLogicalAnd          Operator = "&&"
	OperatorLogicalOr           Operator = "||"
	OperatorBackgroundExecution Operator = "&"
)

// Arithmetic and Comparison Operators
const (
	OperatorAdd      Operator = "+"
	OperatorSubtract Operator = "-"
	OperatorMultiply Operator = "*"
	OperatorDivide   Operator = "/"
	OperatorModulo   Operator = "%"
	OperatorEqual    Operator = "=="
	OperatorNotEqual Operator = "!="
)

// String implements the fmt.Stringer interface.
// Since Operator is already a string, it simply converts the type.
func (o Operator) String() string {
	return string(o)
}

// IsOperator is a helper function to quickly check if a string is a known operator.
func IsOperator(s string) bool {
	switch Operator(s) {
	case OperatorRTruncate, OperatorRTruncate1, OperatorRAppend, OperatorRAppend1,
		OperatorRErrorTruncate, OperatorRErrorAppend, OperatorROutErrorAppend,
		OperatorHereDocument, OperatorHereString, OperatorPipe, OperatorSequentialExecution,
		OperatorLogicalAnd, OperatorLogicalOr, OperatorBackgroundExecution,
		OperatorAdd, OperatorSubtract, OperatorMultiply, OperatorDivide, OperatorModulo,
		OperatorEqual, OperatorNotEqual:
		return true
	}
	return false
}
