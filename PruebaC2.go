package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type BCP struct {
	Nombre           string //nombre del proceso
	PID              int    //numero de proceso
	Estado           string //estado del proceso (Listo, Bloqueado, Terminado, ejecutando)
	ultimalinealeida int    // numero de instruccion del proceso
	tiempoListo      int
}

type Dispatcher struct {
	ColaListos     chan *BCP
	ColaBloqueados chan *BCP
	contador       int //contador global de instrucciones

}

type Instruciones struct {
	Linea     int    // Número de línea
	Tipo      string // Tipo de instrucción (I, E/S, F)
	Parametro int    // Parámetro adicional en caso de ser E/S
}

type Procesador struct {
	Proceso     chan *BCP
	Procesador  bool
	ejecuciones int
}

var procesos = make(map[int][]string)
var pid int = 1
var o int
var contadorO int = 0
var salidaArchivo *os.File

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
				Nombre:           nombreArchivo,
				Estado:           "Listo", // Estado inicial del proceso
				PID:              pid,     //Asignacion de PID
				ultimalinealeida: 0,       // Contador de programa inicial
			}
			pid++
			// Enviar el proceso a la cola de listos
			select {
			case d.ColaListos <- nuevoProceso:
			default:
				fmt.Printf("No se pudo agregar proceso %s a la cola de listos\n", nombreArchivo)
			}
		}
	}
}

func (d *Dispatcher) PasarTiempo() {
	for {

		// Imprimir el valor del contador

		d.CrearProcesos(procesos)
		// Verificar si el contador ha llegado a 50 y salir del bucle
		if d.contador > 99 {
			log.Println("Contador alcanzó 100, terminando...")
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
			log.Println("PULL  Dispatcher ", p.ejecuciones)
			p.ejecuciones++
			p.Procesador = true // Marcar el procesador como ocupado
			log.Printf("LOAD %s  %d \n", proceso.Nombre, p.ejecuciones)
			p.ejecuciones++
			salida <- proceso // Enviar el proceso al canal de salida
		} else {
			// El procesador está ocupado, esperar hasta que se libere
			log.Println("Procesador ocupado, esperando...")

		}
		time.Sleep(2 * time.Second)
	}

}

func (p *Procesador) EjecutarProcesos(salida chan *BCP, maxInstrucciones int, entrada chan *BCP) {

	// Bucle para procesar los elementos del canal
	for proceso := range p.Proceso { // Recibir el primer proceso del canal
		// Mostrar el proceso que se está ejecutando
		contadorO = 0
		log.Printf("EXEC : %s Dispatcher %d \n", proceso.Nombre, p.ejecuciones)
		p.ejecuciones++
		// Leer el archivo asociado al proceso
		archivo, err := os.Open(proceso.Nombre)
		if err != nil {
			fmt.Printf("Error al abrir el archivo %s: %v\n", proceso.Nombre, err)
			continue
		}

		// Leer y mostrar las líneas del archivo
		scanner := bufio.NewScanner(archivo)
		lineaActual := 0
		lineaInicio := proceso.ultimalinealeida

		for scanner.Scan() {
			// Si la línea es mayor o igual a la línea de inicio, procesar
			if lineaActual > lineaInicio {
				lineaTexto := scanner.Text()

				// Ignorar líneas que comienzan con '#'
				if len(lineaTexto) > 0 && lineaTexto[0] == '#' {
					continue
				}

				// Imprimir la línea que será procesada
				log.Printf("%s %d \n", lineaTexto, p.ejecuciones)
				p.ejecuciones++
				contadorO++

				partes := strings.Fields(lineaTexto)

				//convertir campo[0] a entero
				linea, err := strconv.Atoi(partes[0])
				if err != nil {
					fmt.Println("Error al convertir el string a int:", err)
				}
				//guardar instruccion(I,ES,F)
				instruccion := partes[1]
				if instruccion == "ES" {
					//convertir campo[2] en entero
					listo, er := strconv.Atoi(partes[2])
					if er != nil {
						fmt.Println("Error al convertir el string a int:", err)
					}
					// Si la instrucción es E/S, agregar el proceso a la cola de bloqueados
					proceso.Estado = "bloqueado"
					proceso.tiempoListo = listo
					proceso.ultimalinealeida = linea
					log.Printf("ST %s Dispatcher %d \n", proceso.Nombre, p.ejecuciones)
					p.ejecuciones++
					salida <- proceso // Enviar a la cola de bloqueados
					p.Procesador = false
					break
				}
				if instruccion == "F" {
					proceso.Estado = "terminado"
					log.Printf("Proceso terminado ")
					p.Procesador = false
					break
				}
				if contadorO == maxInstrucciones {
					proceso.ultimalinealeida = linea
					log.Printf("ST %s Dispatcher %d \n", proceso.Nombre, p.ejecuciones)
					p.ejecuciones++
					entrada <- proceso
					p.Procesador = false
					break

				}

				time.Sleep(1 * time.Second)

			}
			lineaActual++
			ActualizarContadores(salida, entrada)
		}

		if err := scanner.Err(); err != nil {
			fmt.Printf("Error al leer el archivo %s: %v\n", proceso.Nombre, err)
		}
		archivo.Close()

	}

}
func ActualizarContadores(ColaBloqueados chan *BCP, ColaListo chan *BCP) {
	// Crear un slice para almacenar temporalmente los procesos bloqueados
	var procesos []*BCP

	// Descargar todos los procesos del canal
	for {
		select {
		case proceso := <-ColaBloqueados:
			procesos = append(procesos, proceso)
		default:
			// Cuando el canal está vacío, salir del bucle
			goto Procesar
		}
	}

Procesar:
	// Procesar cada proceso del slice
	for _, proceso := range procesos {
		// Disminuir el contador del proceso
		proceso.tiempoListo--

		// Si el contador llega a 0, considerar desbloquear el proceso
		if proceso.tiempoListo == 0 {
			log.Printf("EVENTO E/S %s movido a cola listo\n", proceso.Nombre)
			proceso.Estado = "listo" // Cambiar el estado a listo (o lo que corresponda)
			ColaListo <- proceso
		} else {
			// Si el proceso aún está bloqueado, reinsertarlo en el canal
			ColaBloqueados <- proceso
		}
	}
}

func main() {
	// Validar argumentos de línea de comandos
	if len(os.Args) != 6 {
		fmt.Println("Uso: Orden_Ejecucion_Prog n o P archivo_orden_creacion_procesos nombre_archivo_salida")
		os.Exit(1)
	}

	// Parsear argumentos
	n, err := strconv.Atoi(os.Args[1])
	if err != nil || n != 1 {
		fmt.Println("Error: n debe ser 1 (un núcleo)")
		os.Exit(1)
	}

	o, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Println("Error: o debe ser un número entero")
		os.Exit(1)
	}

	archivoOrden := os.Args[4]
	archivoSalida := os.Args[5]

	// Abrir archivo de salida
	salidaArchivo, err = os.Create(archivoSalida)
	if err != nil {
		fmt.Println("Error al crear archivo de salida:", err)
		os.Exit(1)
	}
	defer salidaArchivo.Close()

	// Redirigir la salida a un archivo
	writer := bufio.NewWriter(salidaArchivo)
	defer writer.Flush()

	// Redirigir fmt.Printf a archivo usando log
	log.SetFlags(0)
	log.SetOutput(writer)

	// Leer procesos desde el archivo de orden
	procesos, err = LeerProcesosDesdeArchivo(archivoOrden)
	if err != nil {
		fmt.Println("Error al leer archivo de procesos:", err)
		os.Exit(1)
	}

	// Inicializar dispatcher y procesador
	d := Dispatcher{
		ColaListos:     make(chan *BCP, 10),
		ColaBloqueados: make(chan *BCP, 10),
		contador:       0,
	}

	p := Procesador{
		Proceso:     make(chan *BCP, n),
		Procesador:  false,
		ejecuciones: 0,
	}

	// Iniciar gorrutinas
	go d.PasarTiempo()
	go d.transferirProcesos(d.ColaListos, p.Proceso, &p)
	go p.EjecutarProcesos(d.ColaBloqueados, o, d.ColaListos)

	// Esperar un tiempo razonable para la simulación
	time.Sleep(60 * time.Second)

}
