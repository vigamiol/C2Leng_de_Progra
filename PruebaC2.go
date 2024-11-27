package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type BCP struct {
	Nombre string //nombre del proceso
	PID    int    //numero de proceso
	Estado string //estado del proceso (Listo, Bloqueado, Terminado, ejecutado)
	CP     int    // numero de instruccion del proceso

}

type Dispatcher struct {
	ColaListos     chan *BCP
	ColaBloqueados chan *BCP
	procesador     bool
	contador       int //contador global de instrucciones
}

type Instruccion struct {
	Numero    int    // Número de la instrucción
	Tipo      string // Tipo de instrucción: "I", "ES", "F"
	Parametro int    // Parámetro adicional (por ejemplo, tiempo de bloqueo en "ES")
}

var procesos = make(map[int][]string)
var pid int = 0

func LeerProcesosDesdeArchivo(nombreArchivo string) (map[int][]string, error) {
	// Abrir archivo
	file, err := os.Open(nombreArchivo)
	if err != nil {
		return nil, err // Devuelve error si no se puede abrir el archivo
	}
	defer file.Close()

	// Mapa para almacenar los procesos

	// Leer línea por línea
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Dividir la línea en partes: tiempo y procesos
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue // Saltar líneas inválidas o vacías
		}

		// Leer el tiempo desde la primera parte y los nombres de los procesos desde el resto
		var tiempo int
		fmt.Sscanf(parts[0], "%d", &tiempo)                       // Convertir la primera parte a entero
		procesos[tiempo] = append(procesos[tiempo], parts[1:]...) // Agregar procesos al mapa
	}

	// Verificar si hubo un error al leer el archivo
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return procesos, nil
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
		PID:    0,       // Asignar el PID más adelante
		Estado: "Listo", // Inicia en estado "Listo"
		CP:     0,       // Contador de Programa inicia en 0
	}

	return proceso, nil
}
func ObtenerValorSiExiste(mapa map[int][]string, clave int) ([]string, bool) {
	valor, existe := mapa[clave]
	return valor, existe
}

func (d *Dispatcher) CrearProcesos(tiempoProcesos map[int][]string) {

	// Verificar si el contador coincide con algún tiempo en el mapa
	if archivos, existe := tiempoProcesos[d.contador]; existe {
		// Iterar sobre los archivos de proceso para este tiempo
		for _, nombreArchivo := range archivos {
			// Crear un nuevo BCP para este proceso
			nuevoProceso := &BCP{
				Nombre: nombreArchivo,
				Estado: "Listo", // Estado inicial del proceso
				PID:    0,       // Se puede asignar un PID único si es necesario
				CP:     0,       // Contador de programa inicial
			}
			// Enviar el proceso a la cola de listos
			select {
			case d.ColaListos <- nuevoProceso:
				fmt.Printf("Proceso %s creado y agregado a cola de listos\n", nombreArchivo)
			default:
				fmt.Printf("No se pudo agregar proceso %s a la cola de listos\n", nombreArchivo)
			}
		}
	}
}

func (d *Dispatcher) PasarTiempo() {
	for {

		// Imprimir el valor del contador
		fmt.Println(d.contador)
		d.CrearProcesos(procesos)
		// Verificar si el contador ha llegado a 50 y salir del bucle
		if d.contador > 49 {
			fmt.Println("Contador alcanzó 50, terminando...")
			return
		}
		// duerme medio segundo entre cada iteración
		time.Sleep(500 * time.Millisecond)
		d.contador++
	}

}

func main() {
	// Nombre del archivo de entrada
	nombreArchivo := "orden_creacion.txt"

	// Llamar a la función para leer los procesos
	procesos, err := LeerProcesosDesdeArchivo(nombreArchivo)
	if err != nil {
		fmt.Printf("Error al leer el archivo: %v\n", err)
		return
	}

	// Imprimir el mapa de procesos
	for tiempo, nombres := range procesos {
		fmt.Printf("Tiempo: %d -> Procesos: %v\n", tiempo, nombres)
	}

	d := Dispatcher{
		ColaListos:     make(chan *BCP, 10), // Canal con capacidad para 10 procesos
		ColaBloqueados: make(chan *BCP, 10), //Canal con capacidad para 10 procesos
		contador:       0,                   // Inicializamos el contador a 0
		procesador:     false,               // Iniciamos el contador en falso(nno hay proceso en el procesador)
	}

	go d.PasarTiempo()

	time.Sleep(26 * time.Second)

	for datos := range d.ColaListos {
		fmt.Println(datos.Nombre)
	}

	file, err := os.Create("salida.txt")
	if err != nil {
		fmt.Println("Error al crear el archivo:", err)
		return
	}
	defer file.Close()

	// Crear un escritor bufio
	writer := bufio.NewWriter(file)

	// Escribir varias líneas
	for i := 1; i <= 5; i++ {
		_, err := writer.WriteString(fmt.Sprintf("Línea %d\n", i))
		if err != nil {
			fmt.Println("Error al escribir en el archivo:", err)
			return
		}
	}

	// Asegurarse de vaciar el búfer
	err = writer.Flush()
	if err != nil {
		fmt.Println("Error al vaciar el búfer:", err)
		return
	}

}
