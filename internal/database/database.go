package database

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	dbPath = "database.db"
)

func initializeDB() *sql.DB {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Erro ao abrir o banco de dados: %v", err)
	}

	// Criar tabela funcionario se não existir
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS funcionario (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            nome TEXT
        )
    `)
	if err != nil {
		log.Fatalf("Erro ao criar a tabela funcionario: %v", err)
	}

	// Verificar se a coluna 'observacao' existe na tabela 'horas_extras'
	rows, err := db.Query("PRAGMA table_info(horas_extras)")
	if err != nil {
		log.Fatalf("Erro ao verificar a estrutura da tabela horas_extras: %v", err)
	}
	defer rows.Close()

	var columnExists bool
	for rows.Next() {
		var cid int
		var name string
		var _type string
		var notnull int
		var dflt_value interface{}
		var pk int
		err := rows.Scan(&cid, &name, &_type, &notnull, &dflt_value, &pk)
		if err != nil {
			log.Fatalf("Erro ao ler informações da coluna: %v", err)
		}
		if name == "pausa" {
			columnExists = true
			break
		}
	}
	if !columnExists {
		_, err = db.Exec(`ALTER TABLE horas_extras ADD COLUMN pausa REAL`)
		if err != nil {
			log.Fatalf("Erro ao adicionar a coluna pausa à tabela horas_extras: %v", err)
		}
	}

	// Criar tabela horas_extras se não existir
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS horas_extras (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            funcionario_id INTEGER,
            horas REAL,
            hora_inicio DATETIME,
            hora_fim DATETIME,
            observacao TEXT,
			pausa REAL,
            data_registro DATETIME DEFAULT CURRENT_TIMESTAMP,
            FOREIGN KEY(funcionario_id) REFERENCES funcionario(id)
        )
    `)
	if err != nil {
		log.Fatalf("Erro ao criar a tabela horas_extras: %v", err)
	}

	return db
}

// AddUsuario adiciona um novo funcionário ao banco de dados
func AddUsuario(nome string) (int64, error) {
	db := initializeDB()
	defer db.Close()

	stmt, err := db.Prepare("INSERT INTO funcionario(nome) VALUES(?)")
	if err != nil {
		return 0, fmt.Errorf("Erro ao preparar a declaração SQL: %v", err)
	}
	defer stmt.Close()

	result, err := stmt.Exec(nome)
	if err != nil {
		return 0, fmt.Errorf("Erro ao executar a declaração SQL: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("Erro ao obter o ID do funcionário inserido: %v", err)
	}

	fmt.Println("Funcionário adicionado com sucesso!")
	return id, nil
}

// AddHorasExtras adiciona horas extras para um funcionário com hora de início e fim
func AddHorasExtras(funcionarioID int64, horaInicio, horaFim time.Time, observacao string, pausaEmMinutos float64) error {
	db := initializeDB()
	defer db.Close()

	// Converter a pausa de minutos para horas
	pausaEmHoras := pausaEmMinutos / 60.0

	// Calcular as horas totais trabalhadas
	horasTrabalhadas := horaFim.Sub(horaInicio).Hours()

	// Verificar se as horas trabalhadas são menores ou iguais à pausa
	if horasTrabalhadas <= pausaEmHoras {
		return fmt.Errorf("As horas trabalhadas são menores ou iguais à pausa")
	}

	// Calcular as horas extras subtraindo a pausa das horas totais
	horasExtras := horasTrabalhadas - pausaEmHoras

	stmt, err := db.Prepare("INSERT INTO horas_extras(funcionario_id, horas, hora_inicio, hora_fim, observacao, pausa) VALUES(?, ?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("Erro ao preparar a declaração SQL: %v", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(funcionarioID, horasExtras, horaInicio, horaFim, observacao, pausaEmMinutos)
	if err != nil {
		return fmt.Errorf("Erro ao executar a declaração SQL: %v", err)
	}

	fmt.Println("Horas extras adicionadas com sucesso!")
	return nil
}

func calculateOvertimeHours(horaInicio, horaFim time.Time, pausa float64) float64 {
	// Verificar se o horário de término é anterior ao de início, indicando que ultrapassou o dia
	if horaFim.Before(horaInicio) {
		horaFim = horaFim.AddDate(0, 0, 1) // Adicionar um dia ao horário de término
	}

	duration := horaFim.Sub(horaInicio)           // Calcular a diferença entre os horários
	totalMinutes := duration.Minutes() - pausa*60 // Subtrair a pausa em minutos

	return totalMinutes / 60 // Converter de minutos para horas
}

func calculateTotalMinutes(overtimes []Overtime) float64 {
	totalMinutes := 0.0
	for _, overtime := range overtimes {
		start := overtime.HoraInicio
		end := overtime.HoraFim

		// Verificar se o horário de término é anterior ao de início, indicando que ultrapassou o dia
		if end.Before(start) {
			end = end.AddDate(0, 0, 1) // Adicionar um dia ao horário de término
		}

		duration := end.Sub(start)         // Calcular a diferença entre os horários
		totalMinutes += duration.Minutes() // Converter a diferença para minutos
	}
	return totalMinutes
}

// GetOvertimeForMonth retorna as horas extras para um determinado mês
func GetOvertimeForMonth(month time.Time, funcionarioID int) ([]Overtime, error) {
	db := initializeDB()
	defer db.Close()

	startOfMonth := month.Format("2006-01-02")
	endOfMonth := month.AddDate(0, 1, 0).Add(-time.Second).Format("2006-01-02 15:04:05")

	query := `
        SELECT he.id, 
               he.funcionario_id,
               f.nome AS funcionario_nome, 
               he.horas,
               he.hora_inicio, 
               he.hora_fim,
               COALESCE(he.observacao,'') AS observacao,
               he.data_registro,
               COALESCE(NULLIF(he.pausa, ''), '0') AS pausa
        FROM horas_extras he
        JOIN funcionario f ON he.funcionario_id = f.id
        WHERE he.data_registro >= ? 
              AND he.data_registro <= ?
              AND he.funcionario_id = ?`

	rows, err := db.Query(query, startOfMonth, endOfMonth, funcionarioID)
	if err != nil {
		return nil, fmt.Errorf("Erro ao consultar o banco de dados: %v", err)
	}
	defer rows.Close()

	var overtimes []Overtime

	for rows.Next() {
		var overtime Overtime
		var pausaStr string
		err := rows.Scan(&overtime.ID, &overtime.FuncionarioID, &overtime.FuncionarioNome, &overtime.HorasExtras, &overtime.HoraInicio, &overtime.HoraFim, &overtime.Observacao, &overtime.DataRegistro, &pausaStr)
		if err != nil {
			return nil, fmt.Errorf("Erro ao ler o resultado da consulta: %v", err)
		}

		// Converter pausa para float64
		pausa, err := strconv.ParseFloat(pausaStr, 64)
		if err != nil {
			return nil, fmt.Errorf("Erro ao converter pausa para float64: %v", err)
		}
		overtime.Pausa = pausa

		// Verificar se o horário de término é anterior ao de início, indicando que ultrapassou o dia
		if overtime.HoraFim.Before(overtime.HoraInicio) {
			overtime.HoraFim = overtime.HoraFim.AddDate(0, 0, 1) // Adicionar um dia ao horário de término
		}

		// Calcular a diferença entre o horário de início e término em minutos
		duration := overtime.HoraFim.Sub(overtime.HoraInicio)
		totalMinutes := duration.Minutes()

		// Subtrair a pausa dos minutos totais
		totalMinutes -= overtime.Pausa

		// Formatando para horas e minutos
		hours := int(totalMinutes / 60)
		minutes := int(totalMinutes) % 60

		// Armazenar o total formatado no campo HorasExtras
		overtime.HorasExtras = float64(hours) + float64(minutes)/100

		overtimes = append(overtimes, overtime)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Erro ao ler as linhas do resultado da consulta: %v", err)
	}

	return overtimes, nil
}

func GetHorasExtrasFuncionario(funcionarioID int64, month time.Time) ([]Overtime, error) {
	db := initializeDB()
	defer db.Close()

	startOfMonth := month.Format("2006-01-02")
	endOfMonth := month.AddDate(0, 1, 0).Add(-time.Second).Format("2006-01-02 15:04:05")

	query := `
		SELECT he.id, 
			   he.funcionario_id,
			   f.nome AS funcionario_nome, 
			   he.horas,
			   he.hora_inicio, 
			   he.hora_fim,
			   COALESCE(he.observacao,'') AS observacao,
			   he.data_registro,
			   COALESCE(NULLIF(he.pausa, ''), '0') AS pausa
		FROM horas_extras he
		JOIN funcionario f ON he.funcionario_id = f.id
		WHERE he.data_registro >= ? 
			  AND he.data_registro <= ?
			  AND he.funcionario_id = ?`

	rows, err := db.Query(query, startOfMonth, endOfMonth, funcionarioID)
	if err != nil {
		return nil, fmt.Errorf("Erro ao consultar o banco de dados: %v", err)
	}
	defer rows.Close()

	var overtimes []Overtime

	for rows.Next() {
		var overtime Overtime
		var pausaStr string
		err := rows.Scan(&overtime.ID, &overtime.FuncionarioID, &overtime.FuncionarioNome, &overtime.HorasExtras, &overtime.HoraInicio, &overtime.HoraFim, &overtime.Observacao, &overtime.DataRegistro, &pausaStr)
		if err != nil {
			return nil, fmt.Errorf("Erro ao ler o resultado da consulta: %v", err)
		}

		// Converter pausa para float64
		pausa, err := strconv.ParseFloat(pausaStr, 64)
		if err != nil {
			return nil, fmt.Errorf("Erro ao converter pausa para float64: %v", err)
		}
		overtime.Pausa = pausa

		// Verificar se o horário de término é anterior ao de início, indicando que ultrapassou o dia
		if overtime.HoraFim.Before(overtime.HoraInicio) {
			overtime.HoraFim = overtime.HoraFim.AddDate(0, 0, 1) // Adicionar um dia ao horário de término
		}

		// Calcular a diferença entre o horário de início e término em minutos
		duration := overtime.HoraFim.Sub(overtime.HoraInicio)
		totalMinutes := duration.Minutes()

		// Subtrair a pausa dos minutos totais
		totalMinutes -= overtime.Pausa

		// Formatando para horas e minutos
		hours := int(totalMinutes / 60)
		minutes := int(totalMinutes) % 60

		// Armazenar o total formatado no campo HorasExtras
		overtime.HorasExtras = float64(hours) + float64(minutes)/100

		overtimes = append(overtimes, overtime)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Erro ao ler as linhas do resultado da consulta: %v", err)
	}

	return overtimes, nil
}
