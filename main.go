package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/italopatrick/workhours-cli/internal/database"
	"github.com/italopatrick/workhours-cli/internal/report"
	"github.com/manifoldco/promptui"
)

func main() {
	for {
		prompt := promptui.Select{
			Label: "Selecione uma opção",
			Items: []string{"Adicionar funcionário", "Adicionar horas extras", "Ver minhas horas extras", "Gerar relatório", "Sair"},
		}

		_, result, err := prompt.Run()

		if err != nil {
			fmt.Printf("Prompt falhou %v\n", err)
			return
		}

		switch result {
		case "Adicionar funcionário":
			addFuncionario()
		case "Adicionar horas extras":
			addHorasExtras()
		case "Ver minhas horas extras":
			viewMyOvertime()
		case "Gerar relatório":
			generateMonthlyReport()
		case "Sair":
			fmt.Println("Saindo...")
			return
		}
	}
}

func addFuncionario() {
	prompt := promptui.Prompt{
		Label: "Nome do funcionário",
	}

	nome, err := prompt.Run()
	if err != nil {
		fmt.Printf("Erro ao ler o nome do funcionário: %v\n", err)
		return
	}

	_, err = database.AddUsuario(nome)
	if err != nil {
		fmt.Printf("Erro ao adicionar funcionário: %v\n", err)
		return
	}
}

func addHorasExtras() {
	prompt := promptui.Prompt{
		Label: "Código do funcionário",
	}

	codigoFuncionarioStr, err := prompt.Run()
	if err != nil {
		fmt.Printf("Erro ao ler o código do funcionário: %v\n", err)
		return
	}

	codigoFuncionario, err := strconv.ParseInt(codigoFuncionarioStr, 10, 64)
	if err != nil {
		fmt.Printf("Erro ao converter o código do funcionário para inteiro: %v\n", err)
		return
	}

	prompt = promptui.Prompt{
		Label: "Data e hora de início (AAAA-MM-DD HH:MM)",
	}

	horaInicioStr, err := prompt.Run()
	if err != nil {
		fmt.Printf("Erro ao ler a hora de início: %v\n", err)
		return
	}

	horaInicio, err := time.Parse("2006-01-02 15:04", horaInicioStr)
	if err != nil {
		fmt.Printf("Erro ao converter a hora de início para o formato correto: %v\n", err)
		return
	}

	prompt = promptui.Prompt{
		Label: "Data e hora de término (AAAA-MM-DD HH:MM)",
	}

	horaFimStr, err := prompt.Run()
	if err != nil {
		fmt.Printf("Erro ao ler a hora de término: %v\n", err)
		return
	}

	horaFim, err := time.Parse("2006-01-02 15:04", horaFimStr)
	if err != nil {
		fmt.Printf("Erro ao converter a hora de término para o formato correto: %v\n", err)
		return
	}

	prompt = promptui.Prompt{
		Label: "Observação",
	}

	observacao, err := prompt.Run()
	if err != nil {
		fmt.Printf("Erro ao ler a observação: %v\n", err)
		return
	}

	err = database.AddHorasExtras(codigoFuncionario, horaInicio, horaFim, observacao)
	if err != nil {
		fmt.Printf("Erro ao adicionar horas extras: %v\n", err)
		return
	}

	fmt.Println("Horas extras adicionadas com sucesso!")
}

func viewMyOvertime() {
	// Solicitar código do funcionário
	promptCodigo := promptui.Prompt{
		Label: "Código do funcionário",
	}
	codigoFuncionarioStr, err := promptCodigo.Run()
	if err != nil {
		fmt.Printf("Erro ao ler o código do funcionário: %v\n", err)
		return
	}
	codigoFuncionario, err := strconv.ParseInt(codigoFuncionarioStr, 10, 64)
	if err != nil {
		fmt.Printf("Erro ao converter o código do funcionário para inteiro: %v\n", err)
		return
	}

	// Solicitar mês
	promptMes := promptui.Prompt{
		Label: "Mês (AAAA-MM)",
	}
	mesStr, err := promptMes.Run()
	if err != nil {
		fmt.Printf("Erro ao ler o mês: %v\n", err)
		return
	}
	mes, err := time.Parse("2006-01", mesStr)
	if err != nil {
		fmt.Println("Formato de mês inválido. Use AAAA-MM")
		return
	}

	// Obter horas extras do funcionário para o mês fornecido
	overtimes, err := database.GetHorasExtrasFuncionario(codigoFuncionario, mes)
	if err != nil {
		fmt.Printf("Erro ao buscar horas extras do funcionário: %v\n", err)
		return
	}

	if len(overtimes) == 0 {
		fmt.Println("Nenhuma hora extra encontrada para este funcionário neste mês.")
		return
	}

	// Exibir as horas extras
	fmt.Printf("Horas extras do funcionário %d no mês %s:\n", codigoFuncionario, mes.Format("01/2006"))
	for _, overtime := range overtimes {
		fmt.Printf("Data e hora de início: %s\n", overtime.HoraInicio.Format("2006-01-02 15:04"))
		fmt.Printf("Data e hora de término: %s\n", overtime.HoraFim.Format("2006-01-02 15:04"))
		fmt.Printf("Horas extras: %.2f\n\n", overtime.HorasExtras)
	}
}

func generateMonthlyReport() {
	promptFuncionario := promptui.Prompt{
		Label: "ID do Funcionário",
	}
	funcionarioIDStr, err := promptFuncionario.Run()
	if err != nil {
		fmt.Printf("Prompt falhou %v\n", err)
		return
	}
	funcionarioID, err := strconv.Atoi(funcionarioIDStr)
	if err != nil {
		fmt.Printf("Erro ao converter o ID do funcionário para inteiro: %v\n", err)
		return
	}

	promptMonth := promptui.Prompt{
		Label: "Mês (AAAA-MM)",
	}
	monthStr, err := promptMonth.Run()
	if err != nil {
		fmt.Printf("Prompt falhou %v\n", err)
		return
	}
	month, err := time.Parse("2006-01", monthStr)
	if err != nil {
		fmt.Println("Formato de mês inválido. Use AAAA-MM")
		return
	}

	err = report.GenerateMonthlyReport(month, funcionarioID)
	if err != nil {
		fmt.Printf("Erro ao gerar relatório: %v\n", err)
		return
	}

	fmt.Println("Relatório gerado com sucesso!")
}
