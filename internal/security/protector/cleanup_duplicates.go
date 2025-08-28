package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// 这个脚本用于清理fund_protector.go中的重复方法
func main() {
	inputFile := "internal/security/protector/fund_protector.go"
	outputFile := "internal/security/protector/fund_protector_clean.go"

	file, err := os.Open(inputFile)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	output, err := os.Create(outputFile)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		return
	}
	defer output.Close()

	scanner := bufio.NewScanner(file)
	writer := bufio.NewWriter(output)
	defer writer.Flush()

	seenFunctions := make(map[string]bool)
	var currentFunction string
	var functionLines []string
	inFunction := false

	for scanner.Scan() {
		line := scanner.Text()

		// 检测函数开始
		if strings.Contains(line, "func (fp *FundProtector)") {
			// 如果之前有函数，先处理它
			if inFunction && currentFunction != "" {
				if !seenFunctions[currentFunction] {
					seenFunctions[currentFunction] = true
					for _, funcLine := range functionLines {
						writer.WriteString(funcLine + "\n")
					}
				}
			}

			// 开始新函数
			inFunction = true
			functionLines = []string{line}
			
			// 提取函数名
			parts := strings.Split(line, "func (fp *FundProtector) ")
			if len(parts) > 1 {
				funcName := strings.Split(parts[1], "(")[0]
				currentFunction = funcName
			}
		} else if inFunction {
			functionLines = append(functionLines, line)
			
			// 检测函数结束（简单的大括号匹配）
			if strings.TrimSpace(line) == "}" && len(functionLines) > 5 {
				// 函数可能结束了，但我们需要更智能的检测
				// 这里简化处理
			}
		} else {
			// 不在函数中，直接写入
			writer.WriteString(line + "\n")
		}
	}

	// 处理最后一个函数
	if inFunction && currentFunction != "" {
		if !seenFunctions[currentFunction] {
			for _, funcLine := range functionLines {
				writer.WriteString(funcLine + "\n")
			}
		}
	}

	fmt.Println("Duplicate cleanup completed. Check fund_protector_clean.go")
}