package database

import (
	"database/sql"
	"fmt"
	"log"
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
		var dflt_value interface{} // Alteração aqui
		var pk int
		err := rows.Scan(&cid, &name, &_type, &notnull, &dflt_value, &pk)
		if err != nil {
			log.Fatalf("Erro ao ler informações da coluna: %v", err)
		}
		if name == "observacao" {
			columnExists = true
			break
		}
	}
	if !columnExists {
		_, err = db.Exec(`ALTER TABLE horas_extras ADD COLUMN observacao TEXT`)
		if err != nil {
			log.Fatalf("Erro ao adicionar a coluna observacao à tabela horas_extras: %v", err)
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
func AddHorasExtras(funcionarioID int64, horaInicio, horaFim time.Time, observacao string) error {
	db := initializeDB()
	defer db.Close()

	horas := horaFim.Sub(horaInicio).Hours()

	stmt, err := db.Prepare("INSERT INTO horas_extras(funcionario_id, horas, hora_inicio, hora_fim, observacao) VALUES(?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("Erro ao preparar a declaração SQL: %v", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(funcionarioID, horas, horaInicio, horaFim, observacao)
	if err != nil {
		return fmt.Errorf("Erro ao executar a declaração SQL: %v", err)
	}

	fmt.Println("Horas extras adicionadas com sucesso!")
	return nil
}

// GetOvertimeForMonth retorna as horas extras para um determinado mês
func GetOvertimeForMonth(month time.Time) ([]Overtime, error) {
	db := initializeDB()
	defer db.Close()

	startOfMonth := month.Format("2006-01-02")
	endOfMonth := month.AddDate(0, 1, 0).Add(-time.Second).Format("2006-01-02 15:04:05")

	query := `
        SELECT he.id, 
               he.funcionario_id,
               f.nome AS funcionario_nome, 
               (CAST((julianday(he.hora_fim) - julianday(he.hora_inicio)) * 24 AS INTEGER)) + 
               (CAST((julianday(he.hora_fim) - julianday(he.hora_inicio)) * 24 * 60 AS INTEGER) % 60) / 100.0 AS horas_extras,
               he.hora_inicio, 
               he.hora_fim,
			   coalesce (he.observacao,'') AS observacao,
               he.data_registro 
        FROM horas_extras he
        JOIN funcionario f ON he.funcionario_id = f.id
        WHERE he.data_registro >= ? 
              AND he.data_registro <= ?`

	rows, err := db.Query(query, startOfMonth, endOfMonth)
	if err != nil {
		return nil, fmt.Errorf("Erro ao consultar o banco de dados: %v", err)
	}
	defer rows.Close()

	var overtimes []Overtime

	for rows.Next() {
		var overtime Overtime
		err := rows.Scan(&overtime.ID, &overtime.FuncionarioID, &overtime.FuncionarioNome, &overtime.HorasExtras, &overtime.HoraInicio, &overtime.HoraFim, &overtime.Observacao, &overtime.DataRegistro)
		if err != nil {
			return nil, fmt.Errorf("Erro ao ler o resultado da consulta: %v", err)
		}
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

	rows, err := db.Query(`
		SELECT id, 
			   (CAST((julianday(hora_fim) - julianday(hora_inicio)) * 24 AS INTEGER)) + 
			   (CAST((julianday(hora_fim) - julianday(hora_inicio)) * 24 * 60 AS INTEGER) % 60) / 100.0 AS horas_extras,
			   hora_inicio, 
			   hora_fim, 
			   data_registro 
		FROM horas_extras 
		WHERE funcionario_id = ? 
			  AND data_registro >= ? 
			  AND data_registro <= ?`, funcionarioID, startOfMonth, endOfMonth)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar o banco de dados: %v", err)
	}
	defer rows.Close()

	var overtimes []Overtime

	for rows.Next() {
		var overtime Overtime
		err := rows.Scan(&overtime.ID, &overtime.HorasExtras, &overtime.HoraInicio, &overtime.HoraFim, &overtime.DataRegistro)
		if err != nil {
			return nil, fmt.Errorf("erro ao ler o resultado da consulta: %v", err)
		}
		overtimes = append(overtimes, overtime)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("erro ao ler as linhas do resultado da consulta: %v", err)
	}

	return overtimes, nil
}
