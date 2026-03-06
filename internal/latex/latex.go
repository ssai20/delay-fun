package latex

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func latexTableInitial(fileLocation string, title string) error {
	file, _ := os.OpenFile(fileLocation, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.FileMode(0644))
	defer file.Close()
	writer := bufio.NewWriter(file)
	defer writer.Flush()

	lines := []string{
		"\\documentclass[10pt,twoside]{article}",
		"\\usepackage{inputenc}",
		"\\usepackage[russian]{babel}",
		"\\newcommand{\\eps}{\\varepsilon}",
		"\\begin{document}",
		"",
		"\\begin{table} [!htb]",
		"    \\caption {" + title + "}",
		"        \\begin{center}",
		"\\begin{tabular}{|c|c|c|c|c|c|c}",
		"\\cline{1-6} $\\varepsilon$ & \\multicolumn{5}{c|}{$N$} \\\\",
		"\\cline{2-6} &$128$ & $256$ & $512$  & $1024$& $2048$\\\\",
		"",
	}

	for _, line := range lines {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("ошибка записи: %w", err)
		}
	}
	return nil
}

func latexTable(fileLocation string, residual, oa [][]string) error {
	file, err := os.OpenFile(fileLocation, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	epsilons := []string{
		"$1$", "$10^{-1}$", "$10^{-2}$", "$10^{-3}$",
		"$10^{-4}$", "$10^{-5}$", "$10^{-6}$", "$10^{-7}$", "$10^{-8}$",
	}

	for i := 0; i < len(epsilons); i++ {
		if _, err := writer.WriteString("\\cline{1-6}\n"); err != nil {
			return err
		}
		if _, err := writer.WriteString(epsilons[i] + "\n"); err != nil {
			return err
		}

		// Запись residual строки
		resLine := "&$" + strings.Join(residual[i], "$&$") + "$\\\\\n"
		if _, err := writer.WriteString(resLine); err != nil {
			return err
		}

		// Запись oa строки
		oaLine := "&$" + strings.Join(oa[i], "$&$") + "$\\\\\n"
		if _, err := writer.WriteString(oaLine); err != nil {
			return err
		}
	}

	if _, err := writer.WriteString("\\cline{1-6}\n"); err != nil {
		return err
	}

	return nil
}

func latexTableEnd(fileLocation string) error {
	file, err := os.OpenFile(fileLocation, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	lines := []string{
		"\\cline{1-6}",
		"        \\end{tabular}",
		"    \\end{center}",
		"\\end{table}",
		"\\end{document}",
		"",
	}

	for _, line := range lines {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("ошибка записи: %w", err)
		}
	}

	return nil
}

func compileAndOpenPDFFile(fileLocation string) error {
	pdfFile := strings.Replace(fileLocation, ".tex", ".pdf", 1)
	directoryOfFile := filepath.Dir(fileLocation)

	// Компиляция PDF
	cmd1 := exec.Command("pdflatex", "--output-directory="+directoryOfFile, fileLocation)
	cmd1.Stdout = os.Stdout
	cmd1.Stderr = os.Stderr

	if err := cmd1.Run(); err != nil {
		return fmt.Errorf("ошибка компиляции PDF: %w", err)
	}

	// Проверяем, что PDF создался
	if _, err := os.Stat(pdfFile); os.IsNotExist(err) {
		return fmt.Errorf("PDF файл не найден: %s", pdfFile)
	}

	// Отделяем процесс от текущей группы процессов
	cmd2 := exec.Command("xdg-open", pdfFile)
	cmd2.Stdin = nil
	cmd2.Stdout = nil
	cmd2.Stderr = nil
	cmd2.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd2.Start(); err != nil {
		return fmt.Errorf("ошибка открытия PDF: %w", err)
	}

	// Не ждём завершения
	go cmd2.Wait()

	return nil
}

func Latex(fileLocation string, title string, residual, oa [][]string) error {
	err := latexTableInitial(fileLocation, title)
	if err != nil {
		fmt.Errorf("ошибка открытия файла: %w", err)
	}
	err = latexTable(fileLocation, residual, oa)
	if err != nil {
		fmt.Errorf("ошибка открытия файла: %w", err)
	}
	err = latexTableEnd(fileLocation)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла: %w", err)
	}
	err = compileAndOpenPDFFile(fileLocation)
	if err != nil {
		return err
	}
	return nil
}
