package database

import "time"

// Funcionario representa um funcionário
type Funcionario struct {
	ID   int64
	Nome string
}

// HorasExtras representa um registro de horas extras
type HorasExtras struct {
	ID              int64
	FuncionarioID   int64
	Horas           float64
	DataRegistro    time.Time
	HorasFormatadas string
}

type Overtime struct {
	ID              int64
	FuncionarioID   int64
	FuncionarioNome string
	Horas           float64
	DataRegistro    time.Time
	HoraInicio      time.Time
	HoraFim         time.Time
	HorasExtras     float64
	Observacao      string
	Pausa           float64
}
