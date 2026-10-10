# Análisis del código con IA — informe de brechas (GAPs)

**Proyecto:** Predicción clínica hospitalaria con redes neuronales concurrentes en Go — CC65, Trabajo Parcial 2026-20, UPC.
**Repositorio analizado:** https://github.com/Zerovolt0V/1ACC0065-20262-clinical-predictions · rama `develop`, commit `0e3eaed`.
**Modelo:** Claude Opus 5 (`claude-opus-5`, Anthropic), ejecutado desde Claude Code el 10 de octubre de 2026.
**Prompt:** [`PROMPT-ANALISIS-IA.md`](PROMPT-ANALISIS-IA.md).
**Archivos revisados:** `secuencial/secuencial.go` (312 líneas), `concurrente/concurrente.go` (527), `bench/run_bench.ps1` (91),
`bench/analyze.py` (197), `promela/entrenamiento.pml` (82), `README.md`, `docs/GUIA-EJECUCION.md`, `.gitignore`.

**Evidencia adicional ejecutada sobre el código:**

| Herramienta | Resultado |
|---|---|
| `go vet secuencial/secuencial.go` · `go vet concurrente/concurrente.go` | Sin advertencias (código de salida 0). |
| `gofmt -l secuencial concurrente` | Lista ambos archivos. Causa: finales de línea CRLF; el contenido en sí está formateado (`gofmt -d` sobre el texto sin `\r` no produce diferencias). |
| `go run -race concurrente/concurrente.go 4 1 1` | **No pudo ejecutarse:** `go: -race requires cgo; enable cgo by setting CGO_ENABLED=1`. En Windows el detector de carreras necesita un compilador C (gcc/MinGW), que no está instalado. |

---

## 1. Resumen ejecutivo

El código cumple su propósito y la parte concurrente está bien construida: las únicas escrituras compartidas (acumuladores de
pesos y contador de aciertos) ocurren dentro de secciones críticas protegidas por `sync.Mutex`, la barrera de época con
`sync.WaitGroup` garantiza que el promedio solo se calcula cuando todas las goroutines terminaron, y cada goroutine escribe
únicamente en su copia local de pesos o en su rango disjunto de `prob`. Esa estructura coincide con el modelo verificado en Spin.
Las brechas se concentran en tres frentes: **(1) robustez ante entradas** (argumentos de línea de comandos y CSV se convierten con
`strconv` ignorando el error, y un `workers = 0` provoca un pánico por división entre cero), **(2) reproducibilidad del entorno**
(sin `go.mod`, finales de línea CRLF en todos los archivos, detector de carreras no ejecutable en Windows, sin pruebas automatizadas) y
**(3) duplicación**: siete funciones idénticas en los dos programas, consecuencia de la restricción de un archivo por programa.
Ninguna brecha compromete la corrección de la concurrencia ni los resultados de speedup medidos.

## 2. Tabla de hallazgos

Severidad: **Alta** = puede abortar el programa o producir resultados incorrectos sin aviso · **Media** = afecta mantenibilidad,
reproducibilidad o rendimiento de forma apreciable · **Baja** = mejora de calidad sin impacto funcional.

| ID | Dimensión | Severidad | Ubicación | Descripción | Riesgo | Recomendación |
|---|---|---|---|---|---|---|
| G1 | Calidad / Seguridad | Alta | `concurrente.go:493-501`, `secuencial.go` (main) | Los argumentos se convierten con `strconv.Atoi` descartando el error (`workers, _ = ...`). Un valor no numérico deja `0`; un `0` o negativo en `workers` llega a `tam := len(X) / workers` (`concurrente.go:346`) → pánico por división entre cero; `epocas = 0` entrena nada sin avisar. | Caída del programa o ejecución vacía con entradas malformadas; no hay límite superior a `workers` (sobresuscripción silenciosa). | Validar: `if err != nil \|\| workers < 1 \|\| epocas < 1 \|\| repeticiones < 1 { imprimir uso y salir }`; acotar `workers` a `runtime.NumCPU()` o advertir si lo supera. |
| G2 | Seguridad / Calidad | Alta | `concurrente.go:90-114` (`cargarDatos`), equivalente en `secuencial.go` | El CSV se procesa confiando en su forma: `registro[c]` sin comprobar el número de columnas; `ParseFloat`/`Atoi` ignoran el error (un campo corrupto se convierte en 0 en silencio); `x[pos+codigo] = 1` (línea 103) con un código fuera del rango de `tamanos[k]` escribe en la columna equivocada o provoca índice fuera de rango. | Un archivo distinto al generado por `data_augmentation.py` puede abortar el programa o entrenar con datos corruptos sin ningún mensaje. | Verificar `len(registro) == 14` por fila, comprobar el error de cada conversión y validar `0 <= codigo < tamanos[k]`; abortar con un mensaje que indique la fila. |
| G3 | Mantenibilidad | Media | `secuencial.go` y `concurrente.go` | Siete funciones son copias idénticas en ambos archivos: `cargarDatos`, `inicializarPesos`, `propagar`, `retropropagar`, `clasePredicha`, `mostrarResultados` y las constantes/arreglos de configuración. | Cualquier corrección debe hacerse dos veces; ya hoy divergen en comentarios. | Cuando la restricción del curso lo permita, mover el código común a un paquete interno (`internal/red`) o a un archivo compartido del mismo paquete; mientras tanto, documentar la duplicación y mantener un diff de control. |
| G4 | Reproducibilidad | Media | raíz del repositorio | No existe `go.mod`: los programas se ejecutan con `go run archivo.go` en modo GOPATH/legacy, sin versión mínima de Go declarada ni posibilidad de `go build ./...`, `go test ./...` o análisis estático unificado. | Comportamiento dependiente de la instalación local; dificulta CI y herramientas. | `go mod init clinical-predictions` y `go 1.22` como mínimo; no requiere dependencias externas. |
| G5 | Reproducibilidad | Media | todos los archivos de texto | Finales de línea CRLF en `.go`, `.py`, `.ps1` y `.pml` (editados en Windows) y sin `.gitattributes`; `gofmt -l` marca los archivos solo por eso. | Diffs ruidosos, conflictos de merge entre integrantes con sistemas distintos, herramientas de formato que fallan. | Añadir `.gitattributes` con `* text=auto eol=lf` y renormalizar (`git add --renormalize .`). |
| G6 | Calidad / Pruebas | Media | repositorio | No hay pruebas automatizadas (ni unitarias de `propagar`/`retropropagar`, ni de equivalencia secuencial = concurrente con W = 1) ni integración continua; `go vet`/`-race` se ejecutan a mano. El detector de carreras no puede correr en Windows sin cgo. | Una regresión en la sección crítica o en el gradiente pasaría inadvertida. | Añadir `*_test.go` con: gradiente numérico vs. analítico en una red pequeña; W = 1 reproduce la pérdida del secuencial; `-race` en un subconjunto. Ejecutarlos en GitHub Actions sobre Linux (donde `-race` funciona sin instalar nada). |
| G7 | Concurrencia / Rendimiento | Media | `concurrente.go:317-335` | La sección crítica suma las 3 587 celdas de la copia local de cada worker bajo el mismo `Mutex`: con W workers son W sumas serializadas (parte de la fracción secuencial de Amdahl, ≈ 5 %). | Limita el speedup máximo cuando W crece; es la razón principal de la distancia a la recta ideal. | Alternativas: cada worker deja su copia en `copias[w]` (sin candado) y el principal promedia en paralelo por filas de `W1`; o reducir con un árbol de sumas. |
| G8 | Concurrencia | Media | `concurrente.go:317-335` | El orden en que los workers entran a la sección crítica no es determinista, y la suma en coma flotante no es asociativa: dos corridas con el mismo W y la misma semilla pueden diferir en los últimos decimales de los pesos. | La reproducibilidad bit a bit solo está garantizada para W = 1; con W > 1 es reproducible "hasta redondeo", lo que debe documentarse. | Acumular en `copias[w]` y sumar siempre en orden de índice de worker (determinista), o documentar explícitamente la tolerancia. |
| G9 | Concurrencia | Baja | `concurrente.go:53-54, 338-356, 403-420` | `wg` y `mu` son globales reutilizados por el entrenamiento y por el worker pool de predicción; funciona porque las fases nunca se solapan, pero el acoplamiento es implícito. `tam := len(X) / workers` da al último worker el residuo (desbalance ≤ W−1 filas, despreciable). | Un cambio futuro que solape fases crearía una carrera difícil de ver. | Un `WaitGroup` por fase (o local a la función) y reparto por bloques `inicio = w*n/W`. |
| G10 | Concurrencia / Patrones | Baja | `concurrente.go` (diseño) | El patrón Pipeline se aplica como organización por etapas (cargar → entrenar → predecir → priorizar), no con channels productor–consumidor; la decisión fue deliberada (channels no vistos en clase) pero deja sin explorar la sincronización por mini-lotes. | Convergencia por época algo peor con W grande (pérdida 1.052 secuencial vs 1.103 con W = 128). | Prototipar una variante con `chan` de lotes y agregación cada k lotes para comparar convergencia y speedup. |
| G11 | Seguridad / Calidad | Media | `concurrente.go:476-486`, `secuencial.go` (`mostrarResultados`) | `os.Create(nombreArchivo)` sobrescribe `predicciones_*.csv` en el directorio actual sin aviso, y si falla (`err != nil`) el programa no informa nada: la rama `else` no existe. | Pérdida silenciosa de resultados anteriores o de la salida completa (p. ej., disco lleno o carpeta sin permisos). | Informar el error por `os.Stderr`, aceptar la ruta de salida como argumento y no sobrescribir sin `--force`. |
| G12 | Calidad | Media | `concurrente.go:18`, `secuencial.go` | La ruta del dataset es una constante relativa al directorio de trabajo (`data/healthcare_synthetic_1M.csv`); ejecutar desde otra carpeta termina con `os.Exit(1)`. | Fricción de uso y de automatización. | Aceptar `-datos <ruta>` o resolver la ruta relativa al ejecutable. |
| G13 | Calidad | Baja | `concurrente.go:125-134` | Normalización min-max sin proteger la división cuando `maximo == minimo` (columna constante) → `NaN` en todas las filas. | Con el dataset actual no ocurre; con otro dataset produciría pérdida `NaN` sin diagnóstico. | `if maximo > minimo { ... } else { x[k] = 0 }`. |
| G14 | Calidad | Baja | `concurrente.go:78`, `secuencial.go` | `lector.Read()` para saltar la cabecera descarta el error y no comprueba que sea realmente la cabecera esperada. | Un archivo sin cabecera perdería la primera fila de datos en silencio. | Validar los nombres de columna contra la lista esperada. |
| G15 | Rendimiento | Baja | `concurrente.go:460-464` | `sort.Slice` sobre los 210 000 pacientes de prueba en el hilo principal; fuera del cronómetro, pero es el único paso de la fase de salida no paralelizado. | ≈ 0.1 s; sin impacto en el speedup medido. | Mantener; si se necesitara, ordenar parcialmente (top-k con un heap). |
| G16 | Calidad | Baja | `concurrente.go` (comentarios) | Comentarios con erratas y estilo mixto (p. ej., "positivcos", "evaaluacion", "proporciaonal"; algunos `//texto` sin espacio) tras la edición manual. | Solo legibilidad. | Pasar un corrector y seguir el estilo `// Texto` de Go. |
| G17 | Reproducibilidad | Baja | `bench/run_bench.ps1:52-57` | Si la salida del proceso no contiene `MARCA_INICIO_MS`/`MARCA_FIN_MS` (p. ej., el programa abortó), `[int64]("")` lanza una excepción y el barrido se detiene sin indicar qué configuración falló. | Un barrido largo puede perderse por una configuración inválida. | Comprobar `Success` de las coincidencias y registrar la configuración fallida antes de continuar. |
| G18 | Verificación | Baja | `promela/entrenamiento.pml` | El modelo cubre el entrenamiento (W = 3, E = 2) pero no el worker pool de predicción (acumulación de `aciertos`), que usa el mismo mutex. | La fase de predicción queda verificada solo por analogía. | Añadir un segundo `proctype Predictor` con el mismo semáforo y `assert` sobre el total de aciertos. |

**Totales:** 2 de severidad alta, 8 media, 8 baja.

## 3. Detalle de los hallazgos de severidad alta

### G1 — Argumentos sin validar

```go
// concurrente.go:493-501
if len(os.Args) > 1 {
    workers, _ = strconv.Atoi(os.Args[1])
}
...
// concurrente.go:346
tam := len(X) / workers
```

`go run concurrente/concurrente.go abc` deja `workers = 0` y el programa termina con `panic: runtime error: integer divide by zero`
después de haber tardado ~1 s en cargar el CSV. Corrección propuesta:

```go
workers, err := strconv.Atoi(os.Args[1])
if err != nil || workers < 1 {
    fmt.Println("Uso: go run concurrente/concurrente.go [workers>=1] [epocas>=1] [repeticiones>=1]")
    os.Exit(2)
}
if workers > runtime.NumCPU() {
    fmt.Printf("Aviso: %d workers superan los %d hilos lógicos (sobresuscripción)\n", workers, runtime.NumCPU())
}
```

### G2 — CSV sin validación

```go
// concurrente.go:100-105
for k, c := range colsCategoricas {
    codigo, _ := strconv.Atoi(registro[c])
    x[pos+codigo] = 1
    pos += tamanos[k]
}
```

Si `registro[c]` no es numérico, `codigo = 0` y el paciente queda codificado como la primera categoría sin ningún aviso; si el código
excede `tamanos[k]-1`, el `1` cae en la posición de la siguiente variable (dato corrupto) o fuera del vector (pánico). Corrección:

```go
if len(registro) != 14 { fmt.Printf("fila %d: %d columnas, se esperaban 14\n", nfila, len(registro)); os.Exit(1) }
codigo, err := strconv.Atoi(registro[c])
if err != nil || codigo < 0 || codigo >= tamanos[k] {
    fmt.Printf("fila %d, columna %d: código inválido %q\n", nfila, c, registro[c]); os.Exit(1)
}
```

## 4. Lo que está bien (mantener)

- **Sección crítica mínima y correcta.** Toda escritura compartida está bajo `mu` (`concurrente.go:317-335` y `397-399`); las
  lecturas de `W1/b1/W2/b2` por los workers ocurren después de que el principal terminó de escribirlas (orden garantizado por
  `wg.Wait()` de la época anterior). Coincide con el modelo verificado en Spin (0 errores en 4 135 estados).
- **Sin memoria compartida en la fase de cómputo.** Copias locales de pesos por goroutine y rangos disjuntos de `X`, `Xp` y `prob`:
  el patrón que evita carreras sin candados.
- **`defer wg.Done()`** al inicio de cada goroutine: el `Done` se ejecuta aunque la función retorne por otro camino.
- **Reproducibilidad del experimento.** Semilla fija, mismo orden de muestras, equivalencia bit a bit entre secuencial y concurrente con
  W = 1, un proceso por configuración y media recortada: metodología correcta para medir speedup.
- **Softmax numéricamente estable** (resta del máximo) y pérdida con `+1e-12` para evitar `log(0)`.
- **Omisión del gradiente para entradas en cero** (`concurrente.go:280-282`): aprovecha el one-hot y reduce ≈ 10× el costo de la
  actualización de `W1`.
- **Benchmark honesto:** calentamiento descartado, muestreo de CPU y memoria por proceso, resultados y salidas versionados.

## 5. Plan de remediación priorizado

| Orden | Hallazgos | Esfuerzo | Efecto |
|---|---|---|---|
| 1 | G1, G2, G11, G13, G14 | 1–2 h | El programa deja de poder abortar o corromper datos en silencio. |
| 2 | G4, G5 | 15 min | `go.mod` y `.gitattributes`: entorno reproducible y diffs limpios. |
| 3 | G6 | 2–3 h | Pruebas de gradiente y de equivalencia + GitHub Actions con `go vet` y `-race` en Linux. |
| 4 | G3, G9, G12, G16 | 2 h | Código común compartido, `WaitGroup` por fase, ruta de datos configurable, comentarios revisados. |
| 5 | G7, G8, G10, G18 | 1 día | Agregación paralela y determinista, variante con channels y mini-lotes, modelo Promela de la predicción. |
| 6 | G15, G17 | 30 min | Pulido. |

## 6. Conclusión

El código está en un nivel adecuado para un proyecto académico de concurrencia: la parte que el curso evalúa —exclusión mutua,
barreras, worker pool, medición de speedup— es correcta, verificada formalmente y medida con rigor. Para un hipotético Entregable 4 u
operación real, las prioridades serían la validación de entradas (G1, G2), la reproducibilidad del entorno (G4–G6) y la eliminación
de la duplicación (G3). Las mejoras de rendimiento (G7, G10) tienen sentido solo si se necesita escalar más allá de los 24 hilos
lógicos del equipo de pruebas, porque hoy el límite lo marca el hardware, no el código.
