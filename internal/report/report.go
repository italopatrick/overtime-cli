package report

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/italopatrick/overtime-cli/internal/database"
	"github.com/jung-kurt/gofpdf"
)

// GenerateMonthlyReport gera um relatório mensal de horas extras para um funcionário específico
func GenerateMonthlyReport(month time.Time, funcionarioID int) error {
	overtimes, err := database.GetOvertimeForMonth(month, funcionarioID)
	if err != nil {
		return fmt.Errorf("Erro ao obter horas extras do banco de dados: %v", err)
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 12) // Título

	// Obter o diretório de trabalho atual
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("Erro ao obter o diretório de trabalho atual: %v", err)
	}

	// Caminho relativo ao arquivo da logo
	relativePath := "assets/logo.png"

	// Construir o caminho absoluto
	logoPath := filepath.Join(wd, relativePath)

	// Verificar se o arquivo da logo existe
	if _, err := os.Stat(logoPath); os.IsNotExist(err) {
		return fmt.Errorf("Arquivo da logo não encontrado: %v", err)
	}

	pdf.Image(logoPath, 10, 10, 50, 0, false, "", 0, "")

	titleYPos := pdf.GetY() + 30

	pdf.SetY(titleYPos)
	pdf.Cell(60, 10, fmt.Sprintf("Relatorio de Horas Extras - %s", month.Format("01/2006")))
	pdf.Ln(15) // Espaçamento após o título

	for _, overtime := range overtimes {
		pdf.Cell(40, 10, fmt.Sprintf("Funcionario: %s", overtime.FuncionarioNome))
		pdf.Ln(10) // Espaçamento entre informações do funcionário e hora extra

		startTime := overtime.HoraInicio.Format("02/01/2006 15:04")
		endTime := overtime.HoraFim.Format("02/01/2006 15:04")
		pdf.Cell(40, 10, fmt.Sprintf("Hora Inicio: %s", startTime))
		pdf.Ln(7)
		pdf.Cell(40, 10, fmt.Sprintf("Horario Fim: %s", endTime))
		pdf.Ln(7)

		// Adicionando a observação
		pdf.Cell(40, 10, fmt.Sprintf("Observacao: %s", overtime.Observacao))
		pdf.Ln(15)

		// Definindo a cor de texto padrão
		pdf.SetTextColor(0, 0, 0)

		// Exibindo "Horas extras" e "Total Horas Extras" em preto
		pdf.Cell(40, 10, "Horas extras: ")
		pdf.SetTextColor(0, 128, 0)
		pdf.CellFormat(0, 10, fmt.Sprintf("%.2f", overtime.HorasExtras), "", 0, "", false, 0, "") // Adicionando o valor das horas extras na mesma linha
		pdf.SetTextColor(0, 0, 0)
		pdf.Ln(15)
	}

	pdf.Ln(10)

	pdf.SetFont("Arial", "B", 12)

	// Exibindo "Total Horas Extras" em preto
	pdf.Cell(40, 10, "Total Horas Extras: ")
	totalMinutes := getTotalMinutes(overtimes)
	hours := int(totalMinutes / 60)
	minutes := int(totalMinutes) % 60
	pdf.CellFormat(0, 10, fmt.Sprintf("%d horas e %d minutos", hours, minutes), "", 0, "", false, 0, "")

	err = pdf.OutputFileAndClose("relatorio_horas_extras.pdf")
	if err != nil {
		return fmt.Errorf("Erro ao gerar relatório em PDF: %v", err)
	}

	fmt.Println("Relatório PDF gerado com sucesso!")
	return nil
}

func getTotalMinutes(overtimes []database.Overtime) float64 {
	totalMinutes := 0.0
	for _, overtime := range overtimes {
		start := overtime.HoraInicio
		end := overtime.HoraFim

		// Verificar se o horário de término é anterior ao de início, indicando que ultrapassou o dia
		if end.Before(start) {
			end = end.AddDate(0, 0, 1) // Adicionar um dia ao horário de término
		}

		// Calcular a diferença entre os horários e subtrair a pausa
		duration := end.Sub(start) - time.Duration(overtime.Pausa)*time.Minute
		totalMinutes += duration.Minutes() // Converter a diferença para minutos
	}
	return totalMinutes
}
