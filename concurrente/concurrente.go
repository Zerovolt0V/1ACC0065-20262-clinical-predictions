// Entrenamiento concurrente SPMD: W goroutines, sync.WaitGroup y sync.Mutex.
// Uso: go run concurrente/concurrente.go [workers] [epocas] [repeticiones]
package main

import (
	"encoding/csv"
	"fmt"
	"math"
	"math/rand"
	"os"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"time"
)

const rutaDatos = "data/healthcare_synthetic_1M.csv"
const ocultas = 64         //neuronas ocultas
const clases = 3           //3 clases de salida 0 = Anormal, 1 = Inconcluso, 2 = Normal
const tasa = 0.01          // tasa de aprendizaje (SGD)
const semilla = 42         // misma inicialización en todas las corridas
const fraccionPrueba = 0.2 //conjunto de entrenamiento 80% conjunto de prueba 20%

// columnas del CSV
const colEdad, colGenero, colSangre, colCondicion, colSeguro = 0, 1, 2, 3, 4
const colMonto, colAdmision, colMedicamento, colResultado = 5, 6, 7, 8
const colEstancia, colDiaSemana, colMes, colAnio, colId = 9, 10, 11, 12, 13

var colsCategoricas = []int{colGenero, colSangre, colCondicion, colSeguro, colAdmision, colMedicamento, colDiaSemana, colMes}
var tamanos = []int{2, 8, 6, 5, 3, 5, 7, 12}
var colsNumericas = []int{colEdad, colMonto, colEstancia, colAnio}

var nombreClase = []string{"Anormal", "Inconcluso", "Normal"}
var nombreCondicion = []string{"Arthritis", "Asthma", "Cancer", "Diabetes", "Hypertension", "Obesity"}
var nombreAdmision = []string{"Elective", "Emergency", "Urgent"}
var nombreMedicamento = []string{"Aspirin", "Ibuprofen", "Lipitor", "Paracetamol", "Penicillin"}

//Son 52 neuronas de entrada, luego una capa oculta de 64 neuronas y 3 salidas

var entradas int         // número de entradas de la red (4 numéricas + 48 one-hot = 52)
var X [][]float64        // características de entrenamiento
var Y []int              // clase real de entrenamiento
var Xp [][]float64       // características de prueba
var Yp []int             // clase real de prueba
var crudosPrueba [][]int // [id, edad, condición, admisión, medicamento, estancia] de cada fila de prueba

var W1 [][]float64 // entradas x ocultas
var b1 []float64   //  + ocultas
var W2 [][]float64 // ocultas x clases
var b2 []float64   //  + clases

var wg sync.WaitGroup
var mu sync.Mutex

// W1 pesos que se van a multiplicar desde las neuronas de entrada hasta la capa oculta, b1 se suma a este valor
var sumaW1 [][]float64
var sumab1 []float64

// igual con W2 y b2 pero de la capa oculta de 64 neuronas a la capa de salida de 3 neuronas
var sumaW2 [][]float64
var sumab2 []float64
var sumaPerdida float64

var prob [][]float64
var aciertos int //para medir la exactitud del modelo

func cargarDatos() {
	archivo, err := os.Open(rutaDatos)
	if err != nil {
		fmt.Println("No se pudo abrir el archivo:", err)
		os.Exit(1)
	}
	defer archivo.Close()

	lector := csv.NewReader(archivo)
	lector.ReuseRecord = true
	lector.Read() //saltarse la primera linea del csv que contiene el nombre de cada columna

	//Calcular neuronas de entrada: 4 variables numéricas y 8 categóricas en one hot encoding - 48 = 52
	entradas = len(colsNumericas)
	for _, t := range tamanos {
		entradas += t
	}

	var filas [][]float64
	var clasesFila []int
	var crudos [][]int
	for {
		registro, err := lector.Read()
		if err != nil {
			break
		}
		//toma las variables numericas y las guarda tal cual float64
		x := make([]float64, entradas)
		for k, c := range colsNumericas {
			x[k], _ = strconv.ParseFloat(registro[c], 64)
		}
		// one hot encoding de categóricas usando los tamaños definidos en "tamanos"
		pos := len(colsNumericas)
		for k, c := range colsCategoricas {
			codigo, _ := strconv.Atoi(registro[c])
			x[pos+codigo] = 1
			pos += tamanos[k]
		}
		//Guarda el resultado real en y
		y, _ := strconv.Atoi(registro[colResultado])
		//guarda los datos crudos del paciente para imprimir nombres reales en la tabla final de priorizacion
		id, _ := strconv.Atoi(registro[colId])
		edad, _ := strconv.Atoi(registro[colEdad])
		cond, _ := strconv.Atoi(registro[colCondicion])
		adm, _ := strconv.Atoi(registro[colAdmision])
		med, _ := strconv.Atoi(registro[colMedicamento])
		est, _ := strconv.Atoi(registro[colEstancia])

		filas = append(filas, x)
		clasesFila = append(clasesFila, y)
		crudos = append(crudos, []int{id, edad, cond, adm, med, est})
	}

	//Normalización Min-Max
	//una red neuronal confunde un valor 2024 por ejemplo junto a 2, ya que los números grandes pueden
	//dominar matemáticamente a los pequeños, este bloque busca el valor máximo y mínimo de cada columna numérica
	//luego aplica una fórmula para comprimir esos valores en una escala proporciaonal de 0 a 1
	for k := range colsNumericas {
		minimo, maximo := filas[0][k], filas[0][k]
		for _, x := range filas {
			minimo = math.Min(minimo, x[k])
			maximo = math.Max(maximo, x[k])
		}
		for _, x := range filas {
			x[k] = (x[k] - minimo) / (maximo - minimo)
		}
	}

	//Separación de datos de entrenamiento con datos de prueba. fraccionPrueba = 0.2

	n := len(filas)
	nPrueba := int(float64(n) * fraccionPrueba)

	//80% para entrenar
	X, Y = filas[:n-nPrueba], clasesFila[:n-nPrueba]
	//20% para pruebas
	Xp, Yp = filas[n-nPrueba:], clasesFila[n-nPrueba:]
	crudosPrueba = crudos[n-nPrueba:]
}

// Inicializar red neuronal
// Para que una red neuronal aprenda, sus pesos necesitan valores aleatorios. lo hacemos con semilla 42
func inicializarPesos() {
	gen := rand.New(rand.NewSource(semilla))
	W1 = make([][]float64, entradas) //se reserva memoria
	for i := range W1 {
		W1[i] = make([]float64, ocultas)
		for j := range W1[i] {
			//Inicialización de He (no son aleatorios simples, sino que escalan con la cantidad de entradas)
			W1[i][j] = gen.NormFloat64() * math.Sqrt(2.0/float64(entradas))
		}
	}
	b1 = make([]float64, ocultas)   //se reserva memoria y se rellena con 0s
	W2 = make([][]float64, ocultas) //se reserva memoria
	for i := range W2 {
		W2[i] = make([]float64, clases)
		for j := range W2[i] {
			W2[i][j] = gen.NormFloat64() * math.Sqrt(2.0/float64(ocultas)) //Inicialización de He
		}
	}
	b2 = make([]float64, clases) //se reserva memoria y se rellena con ceros

	//Preparacion de los acumuladores globales
	sumaW1 = copiarMatriz(W1)
	sumab1 = copiarVector(b1)
	sumaW2 = copiarMatriz(W2)
	sumab2 = copiarVector(b2)
}

//Funciones Helper para matrices 2D y vectores 1D

func copiarMatriz(m [][]float64) [][]float64 {
	c := make([][]float64, len(m))
	for i := range m {
		c[i] = make([]float64, len(m[i]))
		copy(c[i], m[i])
	}
	return c
}

func copiarVector(v []float64) []float64 {
	c := make([]float64, len(v))
	copy(c, v)
	return c
}

func ponerCeroMatriz(m [][]float64) {
	for i := range m {
		for j := range m[i] {
			m[i][j] = 0
		}
	}
}

func ponerCeroVector(v []float64) {
	for i := range v {
		v[i] = 0
	}
}

// Motor matemático de la red neuronal

// Propagar: Deducir una respuesta (Paso hacia adelante)
func propagar(x []float64, w1 [][]float64, bb1 []float64, w2 [][]float64, bb2 []float64, h []float64, o []float64) {
	//Cálculo de la capa oculta y función ReLU
	for j := 0; j < ocultas; j++ {
		s := bb1[j]
		for i := 0; i < entradas; i++ {
			// multiplicación de los valores de entrada del paciente por la matriz de pesos de la primera capa, sumando el sesgo bb1
			s += x[i] * w1[i][j]
		}
		// Función ReLU, deja pasar los números positivcos tal como están y convierte cualquier número negativo en 0
		if s > 0 {
			h[j] = s //h es la capa oculta
		} else {
			h[j] = 0
		}
		//ReLU permite a la red aprender patrones no lineales.
	}
	//Cálculo de la salida y la función Softmax
	maximo := math.Inf(-1)
	for k := 0; k < clases; k++ {
		s := bb2[k]
		for j := 0; j < ocultas; j++ {
			s += h[j] * w2[j][k] //toma h(capa oculta) y los multiplica por w2 y suma bb2, obtiene 3 números
		}
		o[k] = s
		maximo = math.Max(maximo, s)
	}
	// Función Softmax: Convierte los números crudos en porcentajes entre 0 y 1
	suma := 0.0
	for k := 0; k < clases; k++ {
		o[k] = math.Exp(o[k] - maximo)
		suma += o[k]
	}
	for k := 0; k < clases; k++ {
		o[k] /= suma
	}
}

// Retropropagar: Proceso de aprender de los errores cometidos (Paso hacia atrás)
// usa el algoritmo de Descenso de Gradiente Estocástico (SGD)
func retropropagar(x []float64, y int, w1 [][]float64, bb1 []float64, w2 [][]float64, bb2 []float64, h []float64, o []float64, dh []float64) {
	//qué tan lejos estuvo la predicc del resultado real
	for k := 0; k < clases; k++ {
		do := o[k]
		if k == y {
			do -= 1
		}
		o[k] = do
	}
	//Ajusta pesos de salida w2 con la tasa de aprendizaje y el error calcula en g
	for j := 0; j < ocultas; j++ {
		g := 0.0
		for k := 0; k < clases; k++ {
			g += w2[j][k] * o[k]
			w2[j][k] -= tasa * h[j] * o[k]
		}
		//derivada de la función ReLU, si la neurona tiene valor positivo, pasa intacto, si es 0, su gradiente g vuelve 0
		//y no propaga ajustes
		if h[j] > 0 {
			dh[j] = g
		} else {
			dh[j] = 0
		}
	}
	//Ajuste de los sesgos de salida (bb2) con la tasa
	for k := 0; k < clases; k++ {
		bb2[k] -= tasa * o[k]
	}
	//Ajuste de la primera capa w1
	for i := 0; i < entradas; i++ {
		if x[i] == 0 {
			continue // para ahorrar calculos
		}
		for j := 0; j < ocultas; j++ {
			w1[i][j] -= tasa * x[i] * dh[j]
		}
	}
	//ajuste de los sesgos iniciales bb1
	for j := 0; j < ocultas; j++ {
		bb1[j] -= tasa * dh[j]
	}
}

// SPMD: motor de ejecución de cada goroutine
func trabajador(id, inicio, fin int) {
	defer wg.Done() // cuando termine la funcion avisa a WaitGroup que ya finalizó

	//copias de pesos y sesgos globales en su memoria local
	w1 := copiarMatriz(W1)
	bb1 := copiarVector(b1)
	w2 := copiarMatriz(W2)
	bb2 := copiarVector(b2)

	h := make([]float64, ocultas)  //vector vacio para almacenar temp la capa oculta
	o := make([]float64, clases)   //vector de 3 espacios donde se guardan las 3 probabilidades finales (Anormal, Inconcluso, Normal)
	dh := make([]float64, ocultas) //vector de 64 donde se guarda la porcion de error (solo para retropropagar) para ajustar w1
	perdida := 0.0                 //perdida parcial del worker, lo que se sumara al final protegido por mutex
	for i := inicio; i < fin; i++ {
		propagar(X[i], w1, bb1, w2, bb2, h, o) //predicción
		perdida += -math.Log(o[Y[i]] + 1e-12)
		retropropagar(X[i], Y[i], w1, bb1, w2, bb2, h, o, dh) // aprendizaje
	}

	//Las Sumas:
	//Se protege por exclusión mutua (sync Mutex)
	//sumaW1,sumab1, sumaW2, sumab2 y sumaPerdida son recursos compartidos, el primer Worker que llega
	//los cierra hasta que haga sus sumas, luego cuando termina lo abre para que el siguiente entre
	mu.Lock() // sección crítica: acumuladores compartidos
	for i := range w1 {
		for j := range w1[i] {
			sumaW1[i][j] += w1[i][j]
		}
	}
	for j := range bb1 {
		sumab1[j] += bb1[j]
	}
	for j := range w2 {
		for k := range w2[j] {
			sumaW2[j][k] += w2[j][k]
		}
	}
	for k := range bb2 {
		sumab2[k] += bb2[k]
	}
	sumaPerdida += perdida
	mu.Unlock() //Aquí se abre
}

func entrenarEpocaConcurrente(workers int) float64 {
	//Limpieza de los acumuladores globales por época
	ponerCeroMatriz(sumaW1)
	ponerCeroVector(sumab1)
	ponerCeroMatriz(sumaW2)
	ponerCeroVector(sumab2)
	sumaPerdida = 0

	tam := len(X) / workers //cuantos pacientes tocan por cada goroutine
	wg.Add(workers)         //segun el numero de workers, cuantos tiene que esperar a que terminen
	for w := 0; w < workers; w++ {
		inicio := w * tam
		fin := inicio + tam
		if w == workers-1 { //si sobran pacientes, aseguramos que el último trabajador procese hasta la última fila
			fin = len(X)
		}
		go trabajador(w+1, inicio, fin) //lanzamiento de los workers
	}
	wg.Wait() // barrera de época, no avanza hasta que todos los trabajadores hagan Wg.Done()

	//Cálculo de Promedios: segun las sumas de todos los workers entre el numero de workers

	for i := range W1 {
		for j := range W1[i] {
			W1[i][j] = sumaW1[i][j] / float64(workers)
		}
	}
	for j := range b1 {
		b1[j] = sumab1[j] / float64(workers)
	}
	for j := range W2 {
		for k := range W2[j] {
			W2[j][k] = sumaW2[j][k] / float64(workers)
		}
	}
	for k := range b2 {
		b2[k] = sumab2[k] / float64(workers)
	}
	return sumaPerdida / float64(len(X))
}

// Una vez que termina el entrenamiento, evaluamos con los 210000 pacientes del grupo
// de prueba y medir que tan precisos son los aprendizajes
// worker de evaluación
func predictor(id, inicio, fin int) {
	defer wg.Done()
	h := make([]float64, ocultas)
	aciertosLocal := 0
	for i := inicio; i < fin; i++ {
		//Lee los pesos globales directamente, ya que la red ya no está aprendiendo
		//no hay necesidad de mutex
		propagar(Xp[i], W1, b1, W2, b2, h, prob[i]) //Xp es el conjunto de prueba
		//se compara Prob[i]:resultados con la verdad del conjunto de prueba Yp[i] y se cuenta
		if clasePredicha(prob[i]) == Yp[i] {
			aciertosLocal++
		}
	}
	//la suma de todos los workers si estará en condición de carrera con los demás
	//usamos mutex
	mu.Lock()
	aciertos += aciertosLocal
	mu.Unlock()
}

// Coordinador de evaluacion
func predecirConcurrente(workers int) [][]float64 {
	prob = make([][]float64, len(Xp)) // hacer espacio para los 210000 pacientes de prueba
	for i := range prob {
		prob[i] = make([]float64, clases) // hacer espacio para las 3 clases Anormal Inconcluso Normal
	}
	aciertos = 0             //Reinicia aciertos
	tam := len(Xp) / workers // cuantos pacientes le toca a cada worker
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		//definir por cada worker su paciente inicial, paciente final y garantizar que el ultimo worker complete todos los pacientes
		inicio := w * tam
		fin := inicio + tam
		if w == workers-1 {
			fin = len(Xp)
		}
		go predictor(w+1, inicio, fin) //lanzar la goroutine del worker de evaaluacion
	}
	wg.Wait()
	fmt.Printf("Worker pool de predicción: %d workers, %d aciertos de %d\n", workers, aciertos, len(Xp))
	return prob //matriz prob de porcentajes
}

// busca el maximo de porcentaje de Test Results
// [0.85, 0.10, 0.05]. en este caso el indice 0 es mayor así que devuelve 0
func clasePredicha(p []float64) int {
	mejor := 0
	for k := 1; k < clases; k++ {
		if p[k] > p[mejor] {
			mejor = k
		}
	}
	return mejor
}

// muestra la información útil tras todo el trabajo de la red neuronal
func mostrarResultados(prob [][]float64, nombreArchivo string) {
	//Recorrer pacientes de prueba(prob), suma a cada clase de Test results y si acertó suma a aciertos
	aciertos := 0
	conteo := make([]int, clases)
	for i := range prob {
		c := clasePredicha(prob[i])
		conteo[c]++
		if c == Yp[i] {
			aciertos++
		}
	}
	n := len(prob)
	//Imprime porcentaje de exactitud global y cuantos pacientes cayeron en cada clase
	fmt.Printf("\n=== Predicción sobre %d pacientes de prueba ===\n", n)
	fmt.Printf("Exactitud: %.2f %%\n", 100*float64(aciertos)/float64(n))
	for k := 0; k < clases; k++ {
		fmt.Printf("  %-10s %7d pacientes (%.1f %%)\n", nombreClase[k], conteo[k], 100*float64(conteo[k])/float64(n))
	}

	//MUESTRA DE LA PRIORIZACION
	//Crea una lista con los números de los pacientes y ordena de mayor a menor con Slice prob[...][0]
	//el 0 es la categoría Anormal
	orden := make([]int, n)
	for i := range orden {
		orden[i] = i
	}
	sort.Slice(orden, func(a, b int) bool { return prob[orden[a]][0] > prob[orden[b]][0] })

	//Aquí imprime los 10 primeros de la lista ya ordenada con una tabla formateada
	fmt.Println("\n=== Cola de priorización: 10 pacientes con mayor probabilidad de examen ANORMAL ===")
	fmt.Printf("%3s %9s %5s %-13s %-10s %-12s %9s %11s  %s\n", "#", "Paciente", "Edad", "Condición", "Admisión", "Medicamento", "Estancia", "P(Anormal)", "Predicción")
	for r := 0; r < 10 && r < n; r++ {
		i := orden[r]
		d := crudosPrueba[i]
		fmt.Printf("%3d %9d %5d %-13s %-10s %-12s %6d d %10.3f   %s\n",
			r+1, d[0], d[1], nombreCondicion[d[2]], nombreAdmision[d[3]], nombreMedicamento[d[4]], d[5], prob[i][0], nombreClase[clasePredicha(prob[i])])
	}
	//Aquí exporta en .csv el reporte completo
	f, err := os.Create(nombreArchivo)
	if err == nil {
		defer f.Close()
		fmt.Fprintln(f, "prioridad,paciente,edad,condicion,admision,medicamento,estancia,p_anormal,p_inconcluso,p_normal,prediccion,real")
		for r, i := range orden {
			d := crudosPrueba[i]
			fmt.Fprintf(f, "%d,%d,%d,%s,%s,%s,%d,%.4f,%.4f,%.4f,%s,%s\n", r+1, d[0], d[1], nombreCondicion[d[2]], nombreAdmision[d[3]],
				nombreMedicamento[d[4]], d[5], prob[i][0], prob[i][1], prob[i][2], nombreClase[clasePredicha(prob[i])], nombreClase[Yp[i]])
		}
		fmt.Printf("\nPredicciones completas guardadas en %s\n", nombreArchivo)
	}
}

func main() {
	workers, epocas, repeticiones := runtime.NumCPU(), 2, 1 //runtime.NumCPU() es todos los nucleos del procesador
	//Verifica si le pasamos parametros por consola
	//como: 'go run concurrente.go 16 3 10' para usar 16 workers, 3 épocas y 10 repeticiones
	if len(os.Args) > 1 {
		workers, _ = strconv.Atoi(os.Args[1])
	}
	if len(os.Args) > 2 {
		epocas, _ = strconv.Atoi(os.Args[2])
	}
	if len(os.Args) > 3 {
		repeticiones, _ = strconv.Atoi(os.Args[3])
	}

	fmt.Printf("=== Entrenamiento CONCURRENTE (SPMD) de la red neuronal con %d workers ===\n", workers)
	inicioCarga := time.Now() //Para contar cuanto demoró en cargar los datos
	cargarDatos()             // Transforma todo el csv a datos para la red neuronal
	fmt.Printf("Datos cargados en %.1f s | entrenamiento: %d filas | prueba: %d filas | entradas: %d | CPUs: %d\n",
		time.Since(inicioCarga).Seconds(), len(X), len(Xp), entradas, runtime.NumCPU())

	//Tiempos para el benchmark
	fmt.Printf("MARCA_INICIO_MS=%d\n", time.Now().UnixMilli())
	for r := 1; r <= repeticiones; r++ {
		fmt.Printf("\nRepetición %d/%d (%d épocas, %d goroutines, %d filas por goroutine)\n", r, repeticiones, epocas, workers, len(X)/workers)
		inicializarPesos() //Random de los pesos pero con semilla 42
		inicio := time.Now()
		for e := 1; e <= epocas; e++ { //Epocas
			inicioEpoca := time.Now()                    //Medir el tiempo de cada epoca
			perdida := entrenarEpocaConcurrente(workers) //guardar el error para luego mostrarlo, además entrenar la red
			fmt.Printf("  Época %d/%d  pérdida = %.6f  (%.2f s)\n", e, epocas, perdida, time.Since(inicioEpoca).Seconds())
		}
		fmt.Printf("  TIEMPO_MS=%d\n", time.Since(inicio).Milliseconds())
	}

	fmt.Printf("MARCA_FIN_MS=%d\n", time.Now().UnixMilli())

	prob := predecirConcurrente(workers)                    // predicción sobre los pacientes de prueba
	mostrarResultados(prob, "predicciones_concurrente.csv") //muestra los resultados
}
