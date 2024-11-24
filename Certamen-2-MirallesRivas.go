package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type BCP struct {
	PID           int
	Estado        string   // "Listo", "Bloqueado", "Ejecutando", "Finalizado"
	CP            int      // Contador de Programa (número de la instrucción que se ejecuta)
	Instrucciones []string // Lista de instrucciones del proceso
	InfoES        int      // Tiempo restante de E/S si está bloqueado
}

type ColaProcesos struct {
	Procesos []BCP
}

type OrdenCreacion struct {
	Tiempo   int      // Tiempo (ciclos de CPU) en que se deben crear los procesos
	Procesos []string // Lista de archivos de procesos a crear
}

func (c *ColaProcesos) AgregarProceso(proceso BCP) {
	c.Procesos = append(c.Procesos, proceso)
}

func (c *ColaProcesos) SacarProceso() (BCP, bool) {
	if len(c.Procesos) == 0 {
		return BCP{}, false // Cola vacía
	}
	proceso := c.Procesos[0]
	c.Procesos = c.Procesos[1:]
	return proceso, true
}

func LeerOrdenCreacion(nombreArchivo string) ([]OrdenCreacion, error) {
	var ordenes []OrdenCreacion
	file, err := os.Open(nombreArchivo)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		linea := scanner.Text()
		if strings.HasPrefix(linea, "#") || strings.TrimSpace(linea) == "" {
			continue
		}
		partes := strings.Fields(linea)
		tiempo, _ := strconv.Atoi(partes[0]) // Tiempo de creación
		procesos := partes[1:]               // Archivos de procesos

		ordenes = append(ordenes, OrdenCreacion{Tiempo: tiempo, Procesos: procesos})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return ordenes, nil
}

func LeerProceso(nombreArchivo string) (BCP, error) {
	file, err := os.Open(nombreArchivo)
	if err != nil {
		return BCP{}, err
	}
	defer file.Close()

	var proceso BCP
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		linea := scanner.Text()
		if strings.HasPrefix(linea, "#") {
			continue // Ignorar comentarios
		}

		if proceso.PID == 0 {
			proceso.PID = 1 // Asignamos un ID de proceso
			proceso.Estado = "Listo"
		}
		proceso.Instrucciones = append(proceso.Instrucciones, linea)
	}

	return proceso, nil
}

func EjecutarProceso(proceso *BCP, colaListos *ColaProcesos, colaBloqueados *ColaProcesos) {
	for _, instruccion := range proceso.Instrucciones {
		fmt.Println("Ejecutando:", instruccion)

		if instruccion == "ES" {
			proceso.Estado = "Bloqueado"
			colaBloqueados.AgregarProceso(*proceso)
			fmt.Println("Proceso bloqueado por E/S")
			return
		}

		if instruccion == "F" {
			proceso.Estado = "Finalizado"
			fmt.Println("Proceso finalizado:", proceso.PID)
			return
		}
	}
}
