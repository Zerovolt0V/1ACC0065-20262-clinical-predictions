// Entrenamiento secuencial de la red neuronal (un solo hilo).
// Uso: go run secuencial/secuencial.go [epocas] [repeticiones]
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
	"time"
)

const rutaDatos = "data/healthcare_synthetic_1M.csv"
const ocultas = 64
const clases = 3
const tasa = 0.01
const semilla = 42
const fraccionPrueba = 0.2

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

var entradas int
var X [][]float64
var Y []int
var Xp [][]float64
var Yp []int
var crudosPrueba [][]int

var W1 [][]float64
var b1 []float64
var W2 [][]float64
var b2 []float64

func cargarDatos() {
	archivo, err := os.Open(rutaDatos)
	if err != nil {
		fmt.Println("No se pudo abrir el archivo:", err)
		os.Exit(1)
	}
	defer archivo.Close()

	lector := csv.NewReader(archivo)
	lector.ReuseRecord = true
	lector.Read()

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
		x := make([]float64, entradas)
		for k, c := range colsNumericas {
			x[k], _ = strconv.ParseFloat(registro[c], 64)
		}
		pos := len(colsNumericas)
		for k, c := range colsCategoricas {
			codigo, _ := strconv.Atoi(registro[c])
			x[pos+codigo] = 1
			pos += tamanos[k]
		}
		y, _ := strconv.Atoi(registro[colResultado])
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

	n := len(filas)
	nPrueba := int(float64(n) * fraccionPrueba)
	X, Y = filas[:n-nPrueba], clasesFila[:n-nPrueba]
	Xp, Yp = filas[n-nPrueba:], clasesFila[n-nPrueba:]
	crudosPrueba = crudos[n-nPrueba:]
}

func inicializarPesos() {
	gen := rand.New(rand.NewSource(semilla))
	W1 = make([][]float64, entradas)
	for i := range W1 {
		W1[i] = make([]float64, ocultas)
		for j := range W1[i] {
			W1[i][j] = gen.NormFloat64() * math.Sqrt(2.0/float64(entradas))
		}
	}
	b1 = make([]float64, ocultas)
	W2 = make([][]float64, ocultas)
	for i := range W2 {
		W2[i] = make([]float64, clases)
		for j := range W2[i] {
			W2[i][j] = gen.NormFloat64() * math.Sqrt(2.0/float64(ocultas))
		}
	}
	b2 = make([]float64, clases)
}

func propagar(x []float64, w1 [][]float64, bb1 []float64, w2 [][]float64, bb2 []float64, h []float64, o []float64) {
	for j := 0; j < ocultas; j++ {
		s := bb1[j]
		for i := 0; i < entradas; i++ {
			s += x[i] * w1[i][j]
		}
		if s > 0 {
			h[j] = s
		} else {
			h[j] = 0
		}
	}
	maximo := math.Inf(-1)
	for k := 0; k < clases; k++ {
		s := bb2[k]
		for j := 0; j < ocultas; j++ {
			s += h[j] * w2[j][k]
		}
		o[k] = s
		maximo = math.Max(maximo, s)
	}
	suma := 0.0
	for k := 0; k < clases; k++ {
		o[k] = math.Exp(o[k] - maximo)
		suma += o[k]
	}
	for k := 0; k < clases; k++ {
		o[k] /= suma
	}
}

func retropropagar(x []float64, y int, w1 [][]float64, bb1 []float64, w2 [][]float64, bb2 []float64, h []float64, o []float64, dh []float64) {
	for k := 0; k < clases; k++ {
		do := o[k]
		if k == y {
			do -= 1
		}
		o[k] = do
	}
	for j := 0; j < ocultas; j++ {
		g := 0.0
		for k := 0; k < clases; k++ {
			g += w2[j][k] * o[k]
			w2[j][k] -= tasa * h[j] * o[k]
		}
		if h[j] > 0 {
			dh[j] = g
		} else {
			dh[j] = 0
		}
	}
	for k := 0; k < clases; k++ {
		bb2[k] -= tasa * o[k]
	}
	for i := 0; i < entradas; i++ {
		if x[i] == 0 {
			continue
		}
		for j := 0; j < ocultas; j++ {
			w1[i][j] -= tasa * x[i] * dh[j]
		}
	}
	for j := 0; j < ocultas; j++ {
		bb1[j] -= tasa * dh[j]
	}
}

func entrenarEpoca() float64 {
	h := make([]float64, ocultas)
	o := make([]float64, clases)
	dh := make([]float64, ocultas)
	perdida := 0.0
	for i := range X {
		propagar(X[i], W1, b1, W2, b2, h, o)
		perdida += -math.Log(o[Y[i]] + 1e-12)
		retropropagar(X[i], Y[i], W1, b1, W2, b2, h, o, dh)
	}
	return perdida / float64(len(X))
}

func predecir() [][]float64 {
	h := make([]float64, ocultas)
	prob := make([][]float64, len(Xp))
	for i := range Xp {
		prob[i] = make([]float64, clases)
		propagar(Xp[i], W1, b1, W2, b2, h, prob[i])
	}
	return prob
}

func clasePredicha(p []float64) int {
	mejor := 0
	for k := 1; k < clases; k++ {
		if p[k] > p[mejor] {
			mejor = k
		}
	}
	return mejor
}

func mostrarResultados(prob [][]float64, nombreArchivo string) {
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
	fmt.Printf("\n=== Predicción sobre %d pacientes de prueba ===\n", n)
	fmt.Printf("Exactitud: %.2f %%\n", 100*float64(aciertos)/float64(n))
	for k := 0; k < clases; k++ {
		fmt.Printf("  %-10s %7d pacientes (%.1f %%)\n", nombreClase[k], conteo[k], 100*float64(conteo[k])/float64(n))
	}

	orden := make([]int, n)
	for i := range orden {
		orden[i] = i
	}
	sort.Slice(orden, func(a, b int) bool { return prob[orden[a]][0] > prob[orden[b]][0] })

	fmt.Println("\n=== Cola de priorización: 10 pacientes con mayor probabilidad de examen ANORMAL ===")
	fmt.Printf("%3s %9s %5s %-13s %-10s %-12s %9s %11s  %s\n", "#", "Paciente", "Edad", "Condición", "Admisión", "Medicamento", "Estancia", "P(Anormal)", "Predicción")
	for r := 0; r < 10 && r < n; r++ {
		i := orden[r]
		d := crudosPrueba[i]
		fmt.Printf("%3d %9d %5d %-13s %-10s %-12s %6d d %10.3f   %s\n",
			r+1, d[0], d[1], nombreCondicion[d[2]], nombreAdmision[d[3]], nombreMedicamento[d[4]], d[5], prob[i][0], nombreClase[clasePredicha(prob[i])])
	}

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
	epocas, repeticiones := 2, 1
	if len(os.Args) > 1 {
		epocas, _ = strconv.Atoi(os.Args[1])
	}
	if len(os.Args) > 2 {
		repeticiones, _ = strconv.Atoi(os.Args[2])
	}

	fmt.Println("=== Entrenamiento SECUENCIAL de la red neuronal ===")
	inicioCarga := time.Now()
	cargarDatos()
	fmt.Printf("Datos cargados en %.1f s | entrenamiento: %d filas | prueba: %d filas | entradas: %d | CPUs: %d\n",
		time.Since(inicioCarga).Seconds(), len(X), len(Xp), entradas, runtime.NumCPU())

	fmt.Printf("MARCA_INICIO_MS=%d\n", time.Now().UnixMilli())
	for r := 1; r <= repeticiones; r++ {
		fmt.Printf("\nRepetición %d/%d (%d épocas, 1 hilo)\n", r, repeticiones, epocas)
		inicializarPesos()
		inicio := time.Now()
		for e := 1; e <= epocas; e++ {
			inicioEpoca := time.Now()
			perdida := entrenarEpoca()
			fmt.Printf("  Época %d/%d  pérdida = %.6f  (%.2f s)\n", e, epocas, perdida, time.Since(inicioEpoca).Seconds())
		}
		fmt.Printf("  TIEMPO_MS=%d\n", time.Since(inicio).Milliseconds())
	}

	fmt.Printf("MARCA_FIN_MS=%d\n", time.Now().UnixMilli())

	prob := predecir()
	mostrarResultados(prob, "predicciones_secuencial.csv")
}
