package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Estructura para representar el Bloque de Control de Proceso (BCP)
type BCP struct {
	Nombre           string
	Estado           string
	ContadorPrograma int
	InstruccionesES  int
	Instrucciones    []Instruccion
}

// Estructura para representar una Instrucción
type Instruccion struct {
	Numero int
	Tipo   string
	Param  int
}

// Estructura del Dispatcher
type Dispatcher struct {
	ColaProcesos   chan *BCP
	ColaBloqueados chan *BCP
}

// Leer orden de creación de procesos
func LeerOrdenCreacion(archivo string) ([][]string, error) {
	var ordenes [][]string

	file, err := os.Open(archivo)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		linea := strings.TrimSpace(scanner.Text())

		// Ignorar comentarios
		if strings.HasPrefix(linea, "#") {
			continue
		}

		partes := strings.Fields(linea)
		if len(partes) > 1 {
			ordenes = append(ordenes, partes)
		}
	}

	return ordenes, nil
}

// Leer archivo de un proceso
func LeerProceso(archivo string) (*BCP, error) {
	file, err := os.Open(archivo)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Leer nombre del proceso (primera línea)
	scanner.Scan()
	nombreProceso := strings.TrimSpace(scanner.Text())
	fmt.Printf("Nombre del proceso: %s\n", nombreProceso)

	// Leer instrucciones
	var instrucciones []Instruccion
	for scanner.Scan() {
		linea := strings.TrimSpace(scanner.Text())

		// Ignorar comentarios
		if strings.HasPrefix(linea, "#") {
			continue
		}

		partes := strings.Fields(linea)
		if len(partes) >= 2 {
			numero, _ := strconv.Atoi(partes[0])
			tipo := partes[1]

			param := 0
			if tipo == "ES" && len(partes) > 2 {
				param, _ = strconv.Atoi(partes[2])
			}

			instrucciones = append(instrucciones, Instruccion{
				Numero: numero,
				Tipo:   tipo,
				Param:  param,
			})
		}
	}

	return &BCP{
		Nombre:           nombreProceso,
		Estado:           "Listo",
		ContadorPrograma: 0,
		InstruccionesES:  0,
		Instrucciones:    instrucciones,
	}, nil
}

func CrearProcesos(ordenes [][]string, cicloActual int, dispatcher *Dispatcher) {
	for _, orden := range ordenes {
		// Extraer el ciclo en el que deben crearse los procesos
		cicloCreacion, _ := strconv.Atoi(orden[0])

		// Si coincide con el ciclo actual
		if cicloCreacion == cicloActual {
			for _, archivoProceso := range orden[1:] { // Iterar sobre los procesos a crear
				proceso, err := LeerProceso(archivoProceso)
				if err != nil {
					fmt.Printf("Error al leer el proceso %s: %v\n", archivoProceso, err)
					continue
				}

				// Agregar el proceso a la cola de listos del dispatcher
				dispatcher.ColaProcesos <- proceso
				fmt.Printf("Ciclo %d: Proceso %s creado y agregado a la cola de listos\n", cicloActual, proceso.Nombre)
			}
		}
	}
}

func SimularDespachador(ordenes [][]string, ciclosMaximos int) {
	dispatcher := &Dispatcher{
		ColaProcesos:   make(chan *BCP, 100), // Cola de listos con capacidad 1
		ColaBloqueados: make(chan *BCP, 100),
	}

	// Ciclo general
	for ciclo := 1; ciclo <= ciclosMaximos; ciclo++ {
		fmt.Printf("Ciclo %d\n", ciclo)

		// Crear procesos según el ciclo actual
		CrearProcesos(ordenes, ciclo, dispatcher)

		// Procesar un proceso en la cola de listos
		select {
		case proceso := <-dispatcher.ColaProcesos:
			fmt.Printf("Ciclo %d: Ejecutando proceso %s\n", ciclo, proceso.Nombre)

			// Simular ejecución de una instrucción
			if proceso.ContadorPrograma < len(proceso.Instrucciones) {
				instruccion := proceso.Instrucciones[proceso.ContadorPrograma]
				fmt.Printf("  Ejecutando instrucción: %v\n", instruccion)

				switch instruccion.Tipo {
				case "I":
					// Instrucción inmediata, avanzar contador
					proceso.ContadorPrograma++
					// Reemplazar el proceso por sí mismo en la cola de listos
					select {
					case dispatcher.ColaProcesos <- proceso:
					default:
						// Si no hay espacio en la cola, significa que el proceso terminó
					}
				case "ES":
					// Instrucción de entrada/salida, bloquear proceso
					fmt.Printf("  Proceso %s bloqueado por %d ciclos\n", proceso.Nombre, instruccion.Param)
					proceso.Estado = "Bloqueado"
					dispatcher.ColaBloqueados <- proceso
				case "F":
					// Instrucción final, proceso terminado
					fmt.Printf("  Proceso %s finalizado\n", proceso.Nombre)
				}
			} else {
				fmt.Printf("  Proceso %s sin instrucciones restantes\n", proceso.Nombre)
			}

		default:
			// No hay procesos listos para ejecutar en este ciclo
			fmt.Println("  No hay procesos listos en este ciclo")
		}
	}
}

func main() {
	ordenes, err := LeerOrdenCreacion("orden_creacion.txt")
	if err != nil {
		fmt.Println("Error al leer el archivo de orden de creación:", err)
		return
	}

	// Simular despachador general
	SimularDespachador(ordenes, 60)
}
