package factors

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// parseOrExpression 解析或表达式
func (p *ExpressionParser) parseOrExpression() (*ParsedExpression, error) {
	left, err := p.parseAndExpression()
	if err != nil {
		return nil, err
	}

	for p.position < p.length && p.peek() == "||" {
		p.consume("||")
		right, err := p.parseAndExpression()
		if err != nil {
			return nil, err
		}

		left = &ParsedExpression{
			Type:     ExprTypeOperator,
			Value:    "||",
			Children: []*ParsedExpression{left, right},
		}
	}

	return left, nil
}

// parseAndExpression 解析与表达式
func (p *ExpressionParser) parseAndExpression() (*ParsedExpression, error) {
	left, err := p.parseEqualityExpression()
	if err != nil {
		return nil, err
	}

	for p.position < p.length && p.peek() == "&&" {
		p.consume("&&")
		right, err := p.parseEqualityExpression()
		if err != nil {
			return nil, err
		}

		left = &ParsedExpression{
			Type:     ExprTypeOperator,
			Value:    "&&",
			Children: []*ParsedExpression{left, right},
		}
	}

	return left, nil
}

// parseEqualityExpression 解析相等性表达式
func (p *ExpressionParser) parseEqualityExpression() (*ParsedExpression, error) {
	left, err := p.parseRelationalExpression()
	if err != nil {
		return nil, err
	}

	for p.position < p.length {
		op := p.peek()
		if op == "==" || op == "!=" {
			p.consume(op)
			right, err := p.parseRelationalExpression()
			if err != nil {
				return nil, err
			}

			left = &ParsedExpression{
				Type:     ExprTypeOperator,
				Value:    op,
				Children: []*ParsedExpression{left, right},
			}
		} else {
			break
		}
	}

	return left, nil
}

// parseRelationalExpression 解析关系表达式
func (p *ExpressionParser) parseRelationalExpression() (*ParsedExpression, error) {
	left, err := p.parseAdditiveExpression()
	if err != nil {
		return nil, err
	}

	for p.position < p.length {
		op := p.peek()
		if op == "<" || op == ">" || op == "<=" || op == ">=" {
			p.consume(op)
			right, err := p.parseAdditiveExpression()
			if err != nil {
				return nil, err
			}

			left = &ParsedExpression{
				Type:     ExprTypeOperator,
				Value:    op,
				Children: []*ParsedExpression{left, right},
			}
		} else {
			break
		}
	}

	return left, nil
}

// parseAdditiveExpression 解析加减表达式
func (p *ExpressionParser) parseAdditiveExpression() (*ParsedExpression, error) {
	left, err := p.parseMultiplicativeExpression()
	if err != nil {
		return nil, err
	}

	for p.position < p.length {
		op := p.peek()
		if op == "+" || op == "-" {
			p.consume(op)
			right, err := p.parseMultiplicativeExpression()
			if err != nil {
				return nil, err
			}

			left = &ParsedExpression{
				Type:     ExprTypeOperator,
				Value:    op,
				Children: []*ParsedExpression{left, right},
			}
		} else {
			break
		}
	}

	return left, nil
}

// parseMultiplicativeExpression 解析乘除表达式
func (p *ExpressionParser) parseMultiplicativeExpression() (*ParsedExpression, error) {
	left, err := p.parseExponentialExpression()
	if err != nil {
		return nil, err
	}

	for p.position < p.length {
		op := p.peek()
		if op == "*" || op == "/" || op == "%" {
			p.consume(op)
			right, err := p.parseExponentialExpression()
			if err != nil {
				return nil, err
			}

			left = &ParsedExpression{
				Type:     ExprTypeOperator,
				Value:    op,
				Children: []*ParsedExpression{left, right},
			}
		} else {
			break
		}
	}

	return left, nil
}

// parseExponentialExpression 解析指数表达式
func (p *ExpressionParser) parseExponentialExpression() (*ParsedExpression, error) {
	left, err := p.parseUnaryExpression()
	if err != nil {
		return nil, err
	}

	for p.position < p.length {
		op := p.peek()
		if op == "^" || op == "**" {
			p.consume(op)
			right, err := p.parseUnaryExpression()
			if err != nil {
				return nil, err
			}

			left = &ParsedExpression{
				Type:     ExprTypeOperator,
				Value:    op,
				Children: []*ParsedExpression{left, right},
			}
		} else {
			break
		}
	}

	return left, nil
}

// parseUnaryExpression 解析一元表达式
func (p *ExpressionParser) parseUnaryExpression() (*ParsedExpression, error) {
	op := p.peek()
	if op == "+" || op == "-" || op == "!" {
		p.consume(op)
		expr, err := p.parseUnaryExpression()
		if err != nil {
			return nil, err
		}

		return &ParsedExpression{
			Type:     ExprTypeOperator,
			Value:    op,
			Children: []*ParsedExpression{expr},
		}, nil
	}

	return p.parsePrimaryExpression()
}

// parsePrimaryExpression 解析基本表达式
func (p *ExpressionParser) parsePrimaryExpression() (*ParsedExpression, error) {
	p.skipWhitespace()

	if p.position >= p.length {
		return nil, fmt.Errorf("unexpected end of expression")
	}

	// 括号表达式
	if p.peek() == "(" {
		p.consume("(")
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if p.peek() != ")" {
			return nil, fmt.Errorf("expected ')', got '%s'", p.peek())
		}
		p.consume(")")
		return expr, nil
	}

	// 函数调用或技术指标
	if p.isIdentifier() {
		return p.parseIdentifierExpression()
	}

	// 数字常量
	if p.isNumber() {
		return p.parseNumber()
	}

	// 字符串常量
	if p.peek() == "\"" || p.peek() == "'" {
		return p.parseString()
	}

	return nil, fmt.Errorf("unexpected token: %s", p.peek())
}

// parseIdentifierExpression 解析标识符表达式
func (p *ExpressionParser) parseIdentifierExpression() (*ParsedExpression, error) {
	identifier := p.parseIdentifier()

	// 检查是否是函数调用
	if p.peek() == "(" {
		return p.parseFunctionCall(identifier)
	}

	// 检查是否是数组访问（滞后操作）
	if p.peek() == "[" {
		return p.parseArrayAccess(identifier)
	}

	// 普通变量
	return &ParsedExpression{
		Type:  ExprTypeVariable,
		Value: identifier,
	}, nil
}

// parseFunctionCall 解析函数调用
func (p *ExpressionParser) parseFunctionCall(functionName string) (*ParsedExpression, error) {
	p.consume("(")

	var args []*ParsedExpression
	var params map[string]float64

	// 解析参数
	if p.peek() != ")" {
		for {
			arg, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			args = append(args, arg)

			if p.peek() == "," {
				p.consume(",")
			} else {
				break
			}
		}
	}

	if p.peek() != ")" {
		return nil, fmt.Errorf("expected ')', got '%s'", p.peek())
	}
	p.consume(")")

	// 提取数字参数
	params = make(map[string]float64)
	for i, arg := range args {
		if arg.Type == ExprTypeConstant {
			if value, ok := arg.Value.(float64); ok {
				switch functionName {
				case "SMA", "EMA", "RSI", "STDDEV", "MAX", "MIN", "RANK", "DELAY", "DELTA", "TS_SUM", "TS_MAX", "TS_MIN":
					if i == 0 {
						params["period"] = value
					}
				case "MACD":
					switch i {
					case 0:
						params["fast_period"] = value
					case 1:
						params["slow_period"] = value
					case 2:
						params["signal_period"] = value
					}
				case "BB_UPPER", "BB_LOWER":
					switch i {
					case 0:
						params["period"] = value
					case 1:
						params["std"] = value
					}
				}
			}
		}
	}

	// 判断是函数还是技术指标
	indicatorNames := []string{"SMA", "EMA", "RSI", "MACD", "STDDEV", "MAX", "MIN", "RANK", "DELAY", "DELTA", "TS_SUM", "TS_MAX", "TS_MIN", "BB_UPPER", "BB_LOWER"}
	isIndicator := false
	for _, name := range indicatorNames {
		if strings.ToUpper(functionName) == name {
			isIndicator = true
			break
		}
	}

	if isIndicator {
		return &ParsedExpression{
			Type:     ExprTypeIndicator,
			Value:    strings.ToUpper(functionName),
			Children: args,
			Params:   params,
		}, nil
	} else {
		return &ParsedExpression{
			Type:     ExprTypeFunction,
			Value:    strings.ToUpper(functionName),
			Children: args,
			Params:   params,
		}, nil
	}
}

// parseArrayAccess 解析数组访问（滞后操作）
func (p *ExpressionParser) parseArrayAccess(variableName string) (*ParsedExpression, error) {
	p.consume("[")

	indexExpr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if p.peek() != "]" {
		return nil, fmt.Errorf("expected ']', got '%s'", p.peek())
	}
	p.consume("]")

	// 如果索引是常量，转换为DELAY指标
	if indexExpr.Type == ExprTypeConstant {
		if period, ok := indexExpr.Value.(float64); ok {
			return &ParsedExpression{
				Type:  ExprTypeIndicator,
				Value: "DELAY",
				Children: []*ParsedExpression{
					{Type: ExprTypeVariable, Value: variableName},
				},
				Params: map[string]float64{"period": period},
			}, nil
		}
	}

	return nil, fmt.Errorf("array access with non-constant index not supported")
}

// parseNumber 解析数字
func (p *ExpressionParser) parseNumber() (*ParsedExpression, error) {
	start := p.position

	// 跳过数字字符
	for p.position < p.length && (unicode.IsDigit(rune(p.expression[p.position])) || p.expression[p.position] == '.') {
		p.position++
	}

	numberStr := p.expression[start:p.position]
	value, err := strconv.ParseFloat(numberStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid number: %s", numberStr)
	}

	return &ParsedExpression{
		Type:  ExprTypeConstant,
		Value: value,
	}, nil
}

// parseString 解析字符串
func (p *ExpressionParser) parseString() (*ParsedExpression, error) {
	quote := p.expression[p.position]
	p.position++ // 跳过开始引号

	start := p.position
	for p.position < p.length && p.expression[p.position] != quote {
		p.position++
	}

	if p.position >= p.length {
		return nil, fmt.Errorf("unterminated string")
	}

	value := p.expression[start:p.position]
	p.position++ // 跳过结束引号

	return &ParsedExpression{
		Type:  ExprTypeConstant,
		Value: value,
	}, nil
}

// parseIdentifier 解析标识符
func (p *ExpressionParser) parseIdentifier() string {
	start := p.position

	// 第一个字符必须是字母或下划线
	if p.position < p.length && (unicode.IsLetter(rune(p.expression[p.position])) || p.expression[p.position] == '_') {
		p.position++
	}

	// 后续字符可以是字母、数字或下划线
	for p.position < p.length && (unicode.IsLetter(rune(p.expression[p.position])) || unicode.IsDigit(rune(p.expression[p.position])) || p.expression[p.position] == '_') {
		p.position++
	}

	return p.expression[start:p.position]
}

// peek 查看当前位置的token
func (p *ExpressionParser) peek() string {
	p.skipWhitespace()

	if p.position >= p.length {
		return ""
	}

	// 检查双字符操作符
	if p.position+1 < p.length {
		twoChar := p.expression[p.position : p.position+2]
		switch twoChar {
		case "==", "!=", "<=", ">=", "&&", "||", "**":
			return twoChar
		}
	}

	// 单字符操作符
	char := p.expression[p.position]
	switch char {
	case '+', '-', '*', '/', '^', '%', '(', ')', '[', ']', ',', '<', '>', '!', '"', '\'':
		return string(char)
	}

	// 标识符或数字
	if unicode.IsLetter(rune(char)) || char == '_' {
		return p.peekIdentifier()
	}

	if unicode.IsDigit(rune(char)) || char == '.' {
		return p.peekNumber()
	}

	return string(char)
}

// peekIdentifier 查看标识符
func (p *ExpressionParser) peekIdentifier() string {
	start := p.position
	pos := p.position

	// 第一个字符必须是字母或下划线
	if pos < p.length && (unicode.IsLetter(rune(p.expression[pos])) || p.expression[pos] == '_') {
		pos++
	}

	// 后续字符可以是字母、数字或下划线
	for pos < p.length && (unicode.IsLetter(rune(p.expression[pos])) || unicode.IsDigit(rune(p.expression[pos])) || p.expression[pos] == '_') {
		pos++
	}

	return p.expression[start:pos]
}

// peekNumber 查看数字
func (p *ExpressionParser) peekNumber() string {
	start := p.position
	pos := p.position

	// 跳过数字字符
	for pos < p.length && (unicode.IsDigit(rune(p.expression[pos])) || p.expression[pos] == '.') {
		pos++
	}

	return p.expression[start:pos]
}

// consume 消费指定的token
func (p *ExpressionParser) consume(expected string) error {
	actual := p.peek()
	if actual != expected {
		return fmt.Errorf("expected '%s', got '%s'", expected, actual)
	}

	p.position += len(expected)
	return nil
}

// skipWhitespace 跳过空白字符
func (p *ExpressionParser) skipWhitespace() {
	for p.position < p.length && unicode.IsSpace(rune(p.expression[p.position])) {
		p.position++
	}
}

// isIdentifier 检查当前位置是否是标识符
func (p *ExpressionParser) isIdentifier() bool {
	p.skipWhitespace()
	if p.position >= p.length {
		return false
	}

	char := rune(p.expression[p.position])
	return unicode.IsLetter(char) || char == '_'
}

// isNumber 检查当前位置是否是数字
func (p *ExpressionParser) isNumber() bool {
	p.skipWhitespace()
	if p.position >= p.length {
		return false
	}

	char := rune(p.expression[p.position])
	return unicode.IsDigit(char) || char == '.'
}
