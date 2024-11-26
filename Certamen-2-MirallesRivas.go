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
	Estado        string
	CP            int
	Instrucciones []Instruccion
	InfoES        int
}

type Dispatcher struct {
	ColaListos     chan *BCP
	ColaBloqueados chan *BCP
	CPU            *BCP
}

type Instruccion struct {
	Numero    int    // Número de la instrucción
	Tipo      string // Tipo de instrucción: "I", "ES", "F"
	Parametro int    // Parámetro adicional (por ejemplo, tiempo de bloqueo en "ES")
}

type OrdenCreacion struct {
	TiempoCreacion int      // Tiempo en que se deben crear los procesos
	Archivos       []string // Archivos de los procesos a cargar
}

func leerOrdenCreacion(rutaArchivo string) ([]OrdenCreacion, error) {
	var orden []OrdenCreacion

	// Abrir el archivo
	archivo, err := os.Open(rutaArchivo)
	if err != nil {
		return nil, fmt.Errorf("error al abrir el archivo: %v", err)
	}
	defer archivo.Close()

	// Crear un lector para leer línea por línea
	scanner := bufio.NewScanner(archivo)

	for scanner.Scan() {
		linea := strings.TrimSpace(scanner.Text())

		// Ignorar líneas vacías o comentarios
		if len(linea) == 0 || strings.HasPrefix(linea, "#") {
			continue
		}

		// Dividir la línea en partes: tiempo de creación y archivos
		partes := strings.Fields(linea)
		if len(partes) < 2 {
			return nil, fmt.Errorf("línea mal formada: %s", linea)
		}

		// Convertir el tiempo de creación a entero
		tiempoCreacion, err := strconv.Atoi(partes[0])
		if err != nil {
			return nil, fmt.Errorf("error al convertir el tiempo de creación: %v", err)
		}

		// Obtener los nombres de los archivos
		archivos := partes[1:]

		// Agregar la entrada al orden
		orden = append(orden, OrdenCreacion{
			TiempoCreacion: tiempoCreacion,
			Archivos:       archivos,
		})
	}

	// Manejar errores de lectura
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error al leer el archivo: %v", err)
	}

	return orden, nil
}

func leerArchivoProceso(rutaArchivo string) (*BCP, error) {
	archivo, err := os.Open(rutaArchivo)
	if err != nil {
		return nil, fmt.Errorf("error al abrir el archivo: %v", err)
	}
	defer archivo.Close()

	scanner := bufio.NewScanner(archivo)
	var instrucciones []Instruccion

	for scanner.Scan() {
		linea := strings.TrimSpace(scanner.Text())

		// Ignorar comentarios y líneas vacías
		if strings.HasPrefix(linea, "#") || linea == "" {
			continue
		}

		// Dividir la línea en dos partes: número de instrucción y tipo (con posible parámetro)
		partes := strings.Fields(linea)
		if len(partes) < 2 {
			return nil, fmt.Errorf("línea mal formada: %s", linea)
		}

		// Convertir el número de instrucción a entero
		numero, err := strconv.Atoi(partes[0])
		if err != nil {
			return nil, fmt.Errorf("error al convertir número de instrucción: %v", err)
		}

		// Determinar el tipo de instrucción y el parámetro (si aplica)
		tipo := partes[1]
		parametro := 0
		if tipo == "ES" && len(partes) == 3 {
			parametro, err = strconv.Atoi(partes[2])
			if err != nil {
				return nil, fmt.Errorf("error al convertir parámetro de ES: %v", err)
			}
		}

		// Agregar la instrucción a la lista
		instrucciones = append(instrucciones, Instruccion{
			Numero:    numero,
			Tipo:      tipo,
			Parametro: parametro,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error al leer el archivo: %v", err)
	}

	// Crear el BCP para este proceso
	proceso := &BCP{
		PID:           0,       // Asignar el PID más adelante
		Estado:        "Listo", // Inicia en estado "Listo"
		CP:            0,       // Contador de Programa inicia en
		Instrucciones: instrucciones,
		InfoES:        0, // Tiempo de E/S en 0 inicialmente
	}

	return proceso, nil
}

func (d *Dispatcher) iniciar() {
	for {
		// Verificar si hay un proceso en la cola de listos
		proceso := <-d.ColaListos
		proceso.Estado = "Ejecutando"
		d.CPU = proceso

		// Ejecutar el proceso
		for proceso.CP < len(proceso.Instrucciones) {
			inst := proceso.Instrucciones[proceso.CP]
			fmt.Printf("Ejecutando Proceso %d, Instrucción: %d %s\n", proceso.PID, inst.Numero, inst.Tipo)

			if inst.Tipo == "I" {
				// Instrucción normal, avanzar al siguiente paso
				proceso.CP++
			} else if inst.Tipo == "ES" {
				// Instrucción de E/S, mover el proceso a la cola de bloqueados
				proceso.Estado = "Bloqueado"
				proceso.InfoES = inst.Parametro // Asignar el tiempo de espera para E/S
				d.ColaBloqueados <- proceso
				break
			} else if inst.Tipo == "F" {
				// Instrucción de finalización, marcar como finalizado
				proceso.Estado = "Finalizado"
				fmt.Printf("Proceso %d finalizado.\n", proceso.PID)
				break
			}
		}

		// Si el proceso no está finalizado, devolverlo a la cola de listos
		if proceso.Estado != "Finalizado" {
			d.ColaListos <- proceso
		}
	}
}

func manejarBloqueados(dispatcher *Dispatcher) {
	for {
		proceso := <-dispatcher.ColaBloqueados

		// Simular la espera de E/S
		if proceso.InfoES > 0 {
			proceso.InfoES-- // Decrementar el tiempo de E/S
		} else {
			// E/S completada, mover a la cola de listos
			proceso.Estado = "Listo"
			dispatcher.ColaListos <- proceso
			fmt.Printf("Proceso %d movido a la Cola de Listos tras completar E/S.\n", proceso.PID)
		}
	}
}

func main() {
	orden, err := leerOrdenCreacion("orden_creacion.txt")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Crear las colas de procesos
	colaListos := make(chan *BCP, 10)     // Cola de procesos listos
	colaBloqueados := make(chan *BCP, 10) // Cola de procesos bloqueados

	// Crear Dispatcher
	dispatcher := &Dispatcher{
		ColaListos:     colaListos,
		ColaBloqueados: colaBloqueados,
		CPU:            nil,
	}

	// Cargar los procesos y agregarlos a la cola de listos
	for _, entrada := range orden {
		for _, archivo := range entrada.Archivos {
			// Leer el archivo de cada proceso
			proceso, err := leerArchivoProceso(archivo)
			if err != nil {
				fmt.Printf("Error al leer el proceso %s: %v\n", archivo, err)
				continue
			}
			proceso.PID = len(colaListos) + 1 // Asignar un PID único

			// Agregar el proceso a la cola de listos
			colaListos <- proceso
			fmt.Printf("Proceso %d agregado a la Cola de Listos\n", proceso.PID)
		}
	}

	// Iniciar el Dispatcher (simular la ejecución)
	go dispatcher.iniciar()

	// Iniciar la simulación de procesos bloqueados
	go manejarBloqueados(dispatcher)

	// Mantener el programa en ejecución
	select {} // Esto mantiene el programa corriendo indefinidamente
}
