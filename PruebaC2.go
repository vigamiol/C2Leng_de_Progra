package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

type BCP struct {
	Nombre string //nombre del proceso
	PID    int    //numero de proceso
	Estado string //estado del proceso (Listo, Bloqueado, Terminado, ejecutando)
	CP     int    // numero de instruccion del proceso

}

type Dispatcher struct {
	ColaListos     chan *BCP
	ColaBloqueados chan *BCP
	contador       int //contador global de instrucciones
	colaproce      bool
}

type Instruciones struct {
	Linea     int    // Número de línea
	Tipo      string // Tipo de instrucción (I, E/S, F)
	Parametro int    // Parámetro adicional en caso de ser E/S
}

type Procesador struct {
	Proceso    chan *BCP
	Procesador bool
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
				PID:    pid,     //Asignacion de PID
				CP:     0,       // Contador de programa inicial
			}
			pid++
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
		time.Sleep(1 * time.Second)
		d.contador++
	}

}

func (d *Dispatcher) transferirProcesos(entrada chan *BCP, salida chan *BCP, p *Procesador) {
	for proceso := range entrada { // Iterar sobre los procesos de entrada
		if !p.Procesador { // Verificar si el procesador está libre
			fmt.Println("PULL  Dispatcher ", d.contador)
			p.Procesador = true // Marcar el procesador como ocupado
			fmt.Printf("LOAD %s  %d \n", proceso.Nombre, d.contador)
			salida <- proceso // Enviar el proceso al canal de salida
		} else {
			// El procesador está ocupado, esperar hasta que se libere
			fmt.Println("Procesador ocupado, esperando...")

		}
		time.Sleep(5 * time.Second)
	}

}

func (p *Procesador) EjecutarProcesos(salida chan *BCP, d *Dispatcher) {
	// Bucle para procesar los elementos del canal
	for proceso := range p.Proceso { // Recibir el primer proceso del canal
		// Mostrar el proceso que se está ejecutando
		fmt.Printf("EXEC : %s Dispatcher %d \n", proceso.Nombre, d.contador)

		// Leer el archivo asociado al proceso
		file, err := os.Open(proceso.Nombre)
		if err != nil {
			fmt.Printf("Error al abrir el archivo %s: %v\n", proceso.Nombre, err)
			continue
		}

		// Leer y mostrar las líneas del archivo
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			fmt.Println(scanner.Text()) // Imprimir cada línea del archivo
		}

		if err := scanner.Err(); err != nil {
			fmt.Printf("Error al leer el archivo %s: %v\n", proceso.Nombre, err)
		}
		file.Close()

		proceso.Estado = "bloqueado"
		//solo cuando el proceso no termina
		salida <- proceso
		p.Procesador = false
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
		colaproce:      false,
	}
	p := Procesador{
		Proceso:    make(chan *BCP, 1),
		Procesador: false,
	}

	go d.PasarTiempo()
	go d.transferirProcesos(d.ColaListos, p.Proceso, &p)
	go p.EjecutarProcesos(d.ColaBloqueados, &d)

	time.Sleep(26 * time.Second)

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
