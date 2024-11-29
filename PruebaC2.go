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
			fmt.Println("PULL  Dispatcher ", p.ejecuciones)
			p.ejecuciones++
			p.Procesador = true // Marcar el procesador como ocupado
			fmt.Printf("LOAD %s  %d \n", proceso.Nombre, p.ejecuciones)
			p.ejecuciones++
			salida <- proceso // Enviar el proceso al canal de salida
		} else {
			// El procesador está ocupado, esperar hasta que se libere
			fmt.Println("Procesador ocupado, esperando...")

		}
		time.Sleep(1 * time.Second)
	}

}

func (p *Procesador) EjecutarProcesos(salida chan *BCP, maxInstrucciones int, entrada chan *BCP, aleatoria int) {

	// Bucle para procesar los elementos del canal
	for proceso := range p.Proceso { // Recibir el primer proceso del canal
		// Mostrar el proceso que se está ejecutando
		contadorO = 0
		fmt.Printf("EXEC : %s Dispatcher %d \n", proceso.Nombre, p.ejecuciones)
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
				fmt.Printf("%s %d \n", lineaTexto, p.ejecuciones)
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

				if contadorO == maxInstrucciones {
					proceso.ultimalinealeida = linea
					fmt.Printf("ST %s Dispatcher %d \n", proceso.Nombre, p.ejecuciones)
					p.ejecuciones++
					entrada <- proceso // Enviar a la cola de bloqueados
					p.Procesador = false
					break

				}
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
					fmt.Printf("ST %s Dispatcher %d \n", proceso.Nombre, p.ejecuciones)
					p.ejecuciones++
					salida <- proceso // Enviar a la cola de bloqueados
					p.Procesador = false
					break
				}
				ActualizarContadores(salida, entrada)

			}
			lineaActual++
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
			fmt.Printf("EVENTO E/S %s movido a cola listo\n", proceso.Nombre)
			proceso.Estado = "listo" // Cambiar el estado a listo (o lo que corresponda)
			ColaListo <- proceso
		} else {
			// Si el proceso aún está bloqueado, reinsertarlo en el canal
			ColaBloqueados <- proceso
		}
	}
}

func main() {
	// Nombre del archivo de entrada
	nombreArchivo := "orden_creacion.txt"
	salida := "salida.txt"
	n := 1
	o = 2
	aleatoria := 5

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
	}

	p := Procesador{
		Proceso:     make(chan *BCP, n),
		Procesador:  false,
		ejecuciones: 0,
	}

	go d.PasarTiempo()
	go d.transferirProcesos(d.ColaListos, p.Proceso, &p)
	go p.EjecutarProcesos(d.ColaBloqueados, o, d.ColaListos, aleatoria)

	time.Sleep(60 * time.Second)

	file, err := os.Create(salida)
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
